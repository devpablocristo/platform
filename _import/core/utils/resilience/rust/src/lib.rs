//! Retry con backoff exponencial y circuit breaker async.

use std::future::Future;
use std::sync::atomic::{AtomicU32, AtomicU64, Ordering};
use std::sync::Arc;
use std::time::{Duration, Instant};
use tokio::time::sleep;

// --- Retry ---

/// Configuración de retry con backoff exponencial.
#[derive(Debug, Clone)]
pub struct RetryConfig {
    pub attempts: u32,
    pub initial_delay: Duration,
    pub max_delay: Duration,
    pub multiplier: f64,
}

impl Default for RetryConfig {
    fn default() -> Self {
        Self {
            attempts: 3,
            initial_delay: Duration::from_millis(200),
            max_delay: Duration::from_secs(2),
            multiplier: 2.0,
        }
    }
}

/// Ejecuta `f` con retry. `should_retry` decide si reintentar.
pub async fn retry<T, E, F, Fut>(
    config: &RetryConfig,
    should_retry: impl Fn(&E) -> bool,
    f: F,
) -> Result<T, E>
where
    F: Fn() -> Fut,
    Fut: Future<Output = Result<T, E>>,
{
    let mut delay = config.initial_delay;
    let mut last_err: Option<E> = None;

    for attempt in 0..config.attempts {
        match f().await {
            Ok(val) => return Ok(val),
            Err(err) => {
                if attempt + 1 >= config.attempts || !should_retry(&err) {
                    return Err(err);
                }
                last_err = Some(err);
                sleep(delay).await;
                delay = next_delay(delay, config);
            }
        }
    }

    Err(last_err.expect("at least one attempt"))
}

fn next_delay(current: Duration, config: &RetryConfig) -> Duration {
    let next = Duration::from_secs_f64(current.as_secs_f64() * config.multiplier);
    next.min(config.max_delay)
}

// --- Circuit Breaker ---

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum CircuitState {
    Closed,
    Open,
    HalfOpen,
}

#[derive(Debug, thiserror::Error)]
#[error("circuit breaker is open")]
pub struct CircuitOpenError;

/// Circuit breaker async thread-safe.
#[derive(Debug, Clone)]
pub struct CircuitBreaker {
    inner: Arc<CircuitInner>,
}

#[derive(Debug)]
struct CircuitInner {
    failure_threshold: u32,
    recovery_timeout: Duration,
    failures: AtomicU32,
    last_failure_epoch_ms: AtomicU64,
}

impl CircuitBreaker {
    pub fn new(failure_threshold: u32, recovery_timeout: Duration) -> Self {
        Self {
            inner: Arc::new(CircuitInner {
                failure_threshold,
                recovery_timeout,
                failures: AtomicU32::new(0),
                last_failure_epoch_ms: AtomicU64::new(0),
            }),
        }
    }

    pub fn state(&self) -> CircuitState {
        let failures = self.inner.failures.load(Ordering::Relaxed);
        if failures < self.inner.failure_threshold {
            return CircuitState::Closed;
        }
        let last = self.inner.last_failure_epoch_ms.load(Ordering::Relaxed);
        let now = epoch_ms();
        if now.saturating_sub(last) >= self.inner.recovery_timeout.as_millis() as u64 {
            CircuitState::HalfOpen
        } else {
            CircuitState::Open
        }
    }

    /// Ejecuta `f` si el circuit está closed o half-open.
    pub async fn call<F, Fut, T, E>(&self, f: F) -> Result<T, CircuitCallError<E>>
    where
        F: FnOnce() -> Fut,
        Fut: Future<Output = Result<T, E>>,
    {
        match self.state() {
            CircuitState::Open => return Err(CircuitCallError::Open(CircuitOpenError)),
            CircuitState::HalfOpen | CircuitState::Closed => {}
        }

        match f().await {
            Ok(val) => {
                self.inner.failures.store(0, Ordering::Relaxed);
                Ok(val)
            }
            Err(err) => {
                self.inner.failures.fetch_add(1, Ordering::Relaxed);
                self.inner
                    .last_failure_epoch_ms
                    .store(epoch_ms(), Ordering::Relaxed);
                Err(CircuitCallError::Inner(err))
            }
        }
    }
}

impl Default for CircuitBreaker {
    fn default() -> Self {
        Self::new(5, Duration::from_secs(30))
    }
}

#[derive(Debug, thiserror::Error)]
pub enum CircuitCallError<E> {
    #[error("circuit breaker is open")]
    Open(CircuitOpenError),
    #[error(transparent)]
    Inner(E),
}

fn epoch_ms() -> u64 {
    Instant::now().elapsed().as_millis() as u64
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::atomic::AtomicU32;

    #[tokio::test]
    async fn retry_succeeds_on_second_attempt() {
        let counter = Arc::new(AtomicU32::new(0));
        let c = counter.clone();

        async fn attempt(c: Arc<AtomicU32>) -> Result<String, String> {
            if c.fetch_add(1, Ordering::Relaxed) == 0 {
                Err("transient".into())
            } else {
                Ok("ok".into())
            }
        }

        let config = RetryConfig {
            attempts: 3,
            initial_delay: Duration::from_millis(1),
            ..Default::default()
        };
        let result = retry(&config, |_: &String| true, || attempt(c.clone())).await;
        assert_eq!(result, Ok("ok".to_string()));
        assert_eq!(counter.load(Ordering::Relaxed), 2);
    }

    #[tokio::test]
    async fn retry_exhausts_attempts() {
        async fn always_fail() -> Result<(), String> {
            Err("fail".into())
        }

        let config = RetryConfig {
            attempts: 2,
            initial_delay: Duration::from_millis(1),
            ..Default::default()
        };
        let result = retry(&config, |_: &String| true, always_fail).await;
        assert_eq!(result, Err("fail".to_string()));
    }

    #[test]
    fn circuit_breaker_starts_closed() {
        let cb = CircuitBreaker::default();
        assert_eq!(cb.state(), CircuitState::Closed);
    }

    #[tokio::test]
    async fn circuit_breaker_opens_after_failures() {
        let cb = CircuitBreaker::new(2, Duration::from_secs(60));
        for _ in 0..2 {
            let _: Result<(), _> = cb.call(|| async { Err::<(), _>("fail") }).await;
        }
        assert_eq!(cb.state(), CircuitState::Open);
    }

    #[tokio::test]
    async fn circuit_breaker_resets_on_success() {
        let cb = CircuitBreaker::new(3, Duration::from_secs(60));
        let _: Result<(), CircuitCallError<&str>> = cb.call(|| async { Err::<(), &str>("fail") }).await;
        let _: Result<&str, CircuitCallError<&str>> = cb.call(|| async { Ok::<&str, &str>("ok") }).await;
        assert_eq!(cb.state(), CircuitState::Closed);
    }
}
