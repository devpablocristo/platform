//! Background worker periódico con graceful shutdown via tokio::select!

use std::future::Future;
use std::time::Duration;
use tokio::time::{interval, MissedTickBehavior};

/// Ejecuta `f` periódicamente hasta que `shutdown` se resuelva.
/// Análogo de Go's `worker.RunPeriodic` pero con `tokio::select!`.
pub async fn run_periodic<F, Fut>(
    name: &str,
    period: Duration,
    shutdown: impl Future<Output = ()>,
    f: F,
) where
    F: Fn() -> Fut,
    Fut: Future<Output = ()>,
{
    if period.is_zero() {
        tracing::warn!(worker = name, "period is zero, not starting");
        return;
    }

    tracing::info!(worker = name, period_ms = period.as_millis() as u64, "starting");

    let mut tick = interval(period);
    tick.set_missed_tick_behavior(MissedTickBehavior::Skip);

    tokio::pin!(shutdown);

    loop {
        tokio::select! {
            _ = &mut shutdown => {
                tracing::info!(worker = name, "shutdown signal received");
                break;
            }
            _ = tick.tick() => {
                tracing::debug!(worker = name, "tick");
                f().await;
            }
        }
    }
}

/// Ejecuta `f` periódicamente con un resultado fallible.
/// Loguea errores sin detener el loop.
pub async fn run_periodic_fallible<F, Fut, E>(
    name: &str,
    period: Duration,
    shutdown: impl Future<Output = ()>,
    f: F,
) where
    F: Fn() -> Fut,
    Fut: Future<Output = Result<(), E>>,
    E: std::fmt::Display,
{
    run_periodic(name, period, shutdown, || async {
        if let Err(err) = f().await {
            tracing::error!(worker = name, error = %err, "tick failed");
        }
    })
    .await;
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::atomic::{AtomicU32, Ordering};
    use std::sync::Arc;

    #[tokio::test]
    async fn runs_and_shuts_down() {
        let counter = Arc::new(AtomicU32::new(0));
        let c = counter.clone();

        let (tx, rx) = tokio::sync::oneshot::channel::<()>();

        let handle = tokio::spawn(async move {
            run_periodic("test", Duration::from_millis(10), async { rx.await.ok(); }, || {
                let c = c.clone();
                async move { c.fetch_add(1, Ordering::Relaxed); }
            })
            .await;
        });

        tokio::time::sleep(Duration::from_millis(55)).await;
        tx.send(()).ok();
        handle.await.unwrap();

        assert!(counter.load(Ordering::Relaxed) >= 3);
    }
}
