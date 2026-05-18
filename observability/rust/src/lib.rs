//! Structured logging JSON + request ID propagation via `tracing`.

use tracing_subscriber::{fmt, EnvFilter, layer::SubscriberExt, util::SubscriberInitExt};

pub const REQUEST_ID_HEADER: &str = "x-request-id";

/// Inicializa logging JSON con filtro de env (RUST_LOG).
pub fn init_json_logging(service: &str) {
    let filter = EnvFilter::try_from_default_env()
        .unwrap_or_else(|_| EnvFilter::new("info"));

    tracing_subscriber::registry()
        .with(filter)
        .with(
            fmt::layer()
                .json()
                .with_target(true)
                .with_thread_ids(false)
                .with_file(false)
                .with_line_number(false)
                .flatten_event(true),
        )
        .init();

    tracing::info!(service = service, "logging initialized");
}

/// Genera un request ID aleatorio.
pub fn new_request_id() -> String {
    uuid::Uuid::new_v4().simple().to_string()[..24].to_owned()
}

/// Extrae request ID de headers HTTP o genera uno nuevo.
pub fn extract_or_generate_request_id(headers: &http::HeaderMap) -> String {
    headers
        .get(REQUEST_ID_HEADER)
        .and_then(|v| v.to_str().ok())
        .map(|s| s.trim().to_owned())
        .filter(|s| !s.is_empty())
        .unwrap_or_else(new_request_id)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn new_request_id_is_24_chars() {
        let id = new_request_id();
        assert_eq!(id.len(), 24);
    }

    #[test]
    fn extract_uses_header_when_present() {
        let mut headers = http::HeaderMap::new();
        headers.insert(REQUEST_ID_HEADER, "my-id-123".parse().unwrap());
        assert_eq!(extract_or_generate_request_id(&headers), "my-id-123");
    }

    #[test]
    fn extract_generates_when_missing() {
        let headers = http::HeaderMap::new();
        let id = extract_or_generate_request_id(&headers);
        assert_eq!(id.len(), 24);
    }
}
