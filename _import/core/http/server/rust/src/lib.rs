//! Axum HTTP server con CORS, security headers, health endpoints y graceful shutdown.

use axum::{
    extract::Json,
    http::{HeaderValue, Method, StatusCode},
    response::IntoResponse,
    routing::get,
    Router,
};
use serde::Serialize;
use tokio::net::TcpListener;
use tower_http::cors::{Any, CorsLayer};

/// Respuesta JSON de error — análogo a `ErrorResponse` de Go.
#[derive(Debug, Serialize)]
pub struct ErrorResponse {
    pub code: String,
    pub message: String,
}

impl IntoResponse for ErrorResponse {
    fn into_response(self) -> axum::response::Response {
        let status = StatusCode::INTERNAL_SERVER_ERROR;
        (status, Json(self)).into_response()
    }
}

/// Wrapper para convertir `DomainError` a respuesta Axum (orphan rule).
pub struct DomainErrorResponse(pub errors::DomainError);

impl IntoResponse for DomainErrorResponse {
    fn into_response(self) -> axum::response::Response {
        let err = self.0;
        let status = StatusCode::from_u16(err.status().as_u16()).unwrap_or(StatusCode::INTERNAL_SERVER_ERROR);
        let body = ErrorResponse {
            code: err.code().to_owned(),
            message: err.message().to_owned(),
        };
        (status, Json(body)).into_response()
    }
}

impl From<errors::DomainError> for DomainErrorResponse {
    fn from(err: errors::DomainError) -> Self {
        Self(err)
    }
}

/// Health response.
#[derive(Serialize)]
struct HealthStatus {
    status: &'static str,
}

/// Crea un Router con /healthz y /readyz.
pub fn health_router() -> Router {
    Router::new()
        .route("/healthz", get(|| async { Json(HealthStatus { status: "ok" }) }))
        .route("/readyz", get(|| async { Json(HealthStatus { status: "ready" }) }))
}

/// Crea un CorsLayer permisivo (desarrollo).
pub fn permissive_cors() -> CorsLayer {
    CorsLayer::new()
        .allow_origin(Any)
        .allow_methods([Method::GET, Method::POST, Method::PATCH, Method::DELETE, Method::OPTIONS])
        .allow_headers(Any)
}

/// Crea un CorsLayer con orígenes explícitos.
pub fn cors_with_origins(origins: &[&str]) -> CorsLayer {
    let origins: Vec<HeaderValue> = origins
        .iter()
        .filter_map(|o| o.parse().ok())
        .collect();

    CorsLayer::new()
        .allow_origin(origins)
        .allow_methods([Method::GET, Method::POST, Method::PATCH, Method::DELETE, Method::OPTIONS])
        .allow_headers(Any)
        .allow_credentials(true)
}

/// Levanta el server con graceful shutdown (SIGTERM/SIGINT).
pub async fn serve(listener: TcpListener, app: Router) -> std::io::Result<()> {
    tracing::info!(addr = %listener.local_addr()?, "http server starting");

    axum::serve(listener, app)
        .with_graceful_shutdown(shutdown_signal())
        .await
}

async fn shutdown_signal() {
    let ctrl_c = async {
        tokio::signal::ctrl_c().await.expect("install ctrl+c handler");
    };

    #[cfg(unix)]
    let terminate = async {
        tokio::signal::unix::signal(tokio::signal::unix::SignalKind::terminate())
            .expect("install signal handler")
            .recv()
            .await;
    };

    #[cfg(not(unix))]
    let terminate = std::future::pending::<()>();

    tokio::select! {
        _ = ctrl_c => { tracing::info!("ctrl+c received, shutting down"); }
        _ = terminate => { tracing::info!("SIGTERM received, shutting down"); }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn health_router_creates() {
        let _ = health_router();
    }

    #[test]
    fn domain_error_has_correct_code() {
        let err = errors::DomainError::NotFound("item".into());
        assert_eq!(err.code(), "NOT_FOUND");
    }
}
