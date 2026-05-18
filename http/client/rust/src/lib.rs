//! Cliente HTTP async con retry, headers dinámicos y body size limit.

use bytes::Bytes;
use reqwest::{Client, Method, Response};
use serde::Serialize;
use std::time::Duration;
use tokio::time::sleep;

#[derive(Debug, thiserror::Error)]
pub enum CallerError {
    #[error("request failed: {0}")]
    Request(#[from] reqwest::Error),
    #[error("serialize body: {0}")]
    Serialize(#[from] serde_json::Error),
    #[error("body too large: {size} > {limit}")]
    BodyTooLarge { size: u64, limit: u64 },
}

/// Cliente HTTP reutilizable para comunicación service-to-service.
#[derive(Debug, Clone)]
pub struct Caller {
    client: Client,
    base_url: String,
    default_headers: http::HeaderMap,
    max_retries: u32,
    retry_base_delay: Duration,
    max_body_size: Option<u64>,
}

pub struct CallerBuilder {
    base_url: String,
    timeout: Duration,
    headers: http::HeaderMap,
    max_retries: u32,
    retry_base_delay: Duration,
    max_body_size: Option<u64>,
}

impl CallerBuilder {
    pub fn new(base_url: impl Into<String>) -> Self {
        Self {
            base_url: base_url.into(),
            timeout: Duration::from_secs(10),
            headers: http::HeaderMap::new(),
            max_retries: 0,
            retry_base_delay: Duration::from_millis(200),
            max_body_size: None,
        }
    }

    pub fn timeout(mut self, timeout: Duration) -> Self { self.timeout = timeout; self }

    pub fn header(mut self, key: &str, value: &str) -> Self {
        if let (Ok(name), Ok(val)) = (
            key.parse::<http::header::HeaderName>(),
            value.parse::<http::header::HeaderValue>(),
        ) {
            self.headers.insert(name, val);
        }
        self
    }

    pub fn max_retries(mut self, n: u32) -> Self { self.max_retries = n; self }
    pub fn retry_base_delay(mut self, d: Duration) -> Self { self.retry_base_delay = d; self }
    pub fn max_body_size(mut self, bytes: u64) -> Self { self.max_body_size = Some(bytes); self }

    pub fn build(self) -> Result<Caller, CallerError> {
        let client = Client::builder().timeout(self.timeout).build()?;
        Ok(Caller {
            client,
            base_url: self.base_url.trim_end_matches('/').to_owned(),
            default_headers: self.headers,
            max_retries: self.max_retries,
            retry_base_delay: self.retry_base_delay,
            max_body_size: self.max_body_size,
        })
    }
}

impl Caller {
    pub fn builder(base_url: impl Into<String>) -> CallerBuilder {
        CallerBuilder::new(base_url)
    }

    /// Ejecuta una petición JSON con retry en 5xx/network errors.
    pub async fn do_json<B: Serialize>(
        &self,
        method: Method,
        path: &str,
        body: Option<&B>,
        extra_headers: Option<&http::HeaderMap>,
    ) -> Result<(u16, Bytes), CallerError> {
        let url = self.join(path);
        let max_attempts = 1 + self.max_retries;
        let mut last_err: Option<CallerError> = None;

        for attempt in 0..max_attempts {
            if attempt > 0 {
                let delay = self.retry_delay(attempt);
                sleep(delay).await;
            }

            match self.do_once(&url, method.clone(), body, extra_headers).await {
                Ok((status, bytes)) => {
                    if status >= 500 && attempt + 1 < max_attempts {
                        tracing::debug!(status, attempt, path, "5xx, retrying");
                        continue;
                    }
                    return Ok((status, bytes));
                }
                Err(e) => {
                    tracing::debug!(attempt, path, error = %e, "request failed, retrying");
                    last_err = Some(e);
                }
            }
        }

        Err(last_err.expect("at least one attempt"))
    }

    async fn do_once<B: Serialize>(
        &self,
        url: &str,
        method: Method,
        body: Option<&B>,
        extra_headers: Option<&http::HeaderMap>,
    ) -> Result<(u16, Bytes), CallerError> {
        let mut req = self.client.request(method, url);

        for (key, value) in &self.default_headers {
            req = req.header(key, value);
        }
        if let Some(extra) = extra_headers {
            for (key, value) in extra {
                req = req.header(key, value);
            }
        }
        if let Some(b) = body {
            req = req.json(b);
        }

        let resp: Response = req.send().await?;
        let status = resp.status().as_u16();
        let bytes = resp.bytes().await?;

        if let Some(limit) = self.max_body_size {
            if bytes.len() as u64 > limit {
                return Err(CallerError::BodyTooLarge { size: bytes.len() as u64, limit });
            }
        }

        Ok((status, bytes))
    }

    fn join(&self, path: &str) -> String {
        if path.starts_with("http://") || path.starts_with("https://") {
            return path.to_owned();
        }
        let clean = if path.starts_with('/') { path.to_owned() } else { format!("/{path}") };
        format!("{}{}", self.base_url, clean)
    }

    fn retry_delay(&self, attempt: u32) -> Duration {
        let base = self.retry_base_delay.as_secs_f64();
        let delay = base * 2f64.powi(attempt as i32 - 1);
        Duration::from_secs_f64(delay.min(10.0))
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn join_with_path() {
        let c = Caller::builder("https://api.example.com").build().unwrap();
        assert_eq!(c.join("/v1/users"), "https://api.example.com/v1/users");
    }

    #[test]
    fn join_absolute_url() {
        let c = Caller::builder("https://api.example.com").build().unwrap();
        assert_eq!(c.join("https://other.com/hook"), "https://other.com/hook");
    }

    #[test]
    fn retry_delay_exponential() {
        let c = Caller::builder("http://localhost").build().unwrap();
        let d1 = c.retry_delay(1);
        let d2 = c.retry_delay(2);
        assert!(d2 > d1);
    }
}
