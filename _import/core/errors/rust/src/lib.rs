//! Errores de dominio categorizados + normalización HTTP.
//!
//! Análogo Rust de `domainerr` + `httperr` de Go. Usa `thiserror` para derive
//! y enums exhaustivos en vez de Kind strings.

use http::StatusCode;
use serde::Serialize;

/// Error de dominio categorizado. Exhaustive matching en compile-time.
#[derive(Debug, Clone, thiserror::Error)]
pub enum DomainError {
    #[error("unauthorized: {0}")]
    Unauthorized(String),

    #[error("forbidden: {0}")]
    Forbidden(String),

    #[error("not found: {0}")]
    NotFound(String),

    #[error("validation: {0}")]
    Validation(String),

    #[error("conflict: {0}")]
    Conflict(String),

    #[error("business rule: {0}")]
    BusinessRule(String),

    #[error("unavailable: {0}")]
    Unavailable(String),

    #[error("upstream: {0}")]
    UpstreamError(String),

    #[error("internal: {0}")]
    Internal(String),
}

impl DomainError {
    /// Mensaje user-facing (sin el prefijo de categoría).
    pub fn message(&self) -> &str {
        match self {
            Self::Unauthorized(m)
            | Self::Forbidden(m)
            | Self::NotFound(m)
            | Self::Validation(m)
            | Self::Conflict(m)
            | Self::BusinessRule(m)
            | Self::Unavailable(m)
            | Self::UpstreamError(m)
            | Self::Internal(m) => m,
        }
    }

    /// Código de error como string (para JSON).
    pub fn code(&self) -> &'static str {
        match self {
            Self::Unauthorized(_) => "UNAUTHORIZED",
            Self::Forbidden(_) => "FORBIDDEN",
            Self::NotFound(_) => "NOT_FOUND",
            Self::Validation(_) => "VALIDATION_ERROR",
            Self::Conflict(_) => "CONFLICT",
            Self::BusinessRule(_) => "BUSINESS_RULE",
            Self::Unavailable(_) => "UNAVAILABLE",
            Self::UpstreamError(_) => "UPSTREAM_ERROR",
            Self::Internal(_) => "INTERNAL",
        }
    }

    /// HTTP status correspondiente.
    pub fn status(&self) -> StatusCode {
        match self {
            Self::Unauthorized(_) => StatusCode::UNAUTHORIZED,
            Self::Forbidden(_) => StatusCode::FORBIDDEN,
            Self::NotFound(_) => StatusCode::NOT_FOUND,
            Self::Validation(_) => StatusCode::BAD_REQUEST,
            Self::Conflict(_) => StatusCode::CONFLICT,
            Self::BusinessRule(_) => StatusCode::UNPROCESSABLE_ENTITY,
            Self::Unavailable(_) => StatusCode::SERVICE_UNAVAILABLE,
            Self::UpstreamError(_) => StatusCode::BAD_GATEWAY,
            Self::Internal(_) => StatusCode::INTERNAL_SERVER_ERROR,
        }
    }

    /// Convierte a respuesta JSON `{error: {code, message}}`.
    pub fn to_api_error(&self) -> ApiError {
        ApiError {
            code: self.code().to_owned(),
            message: self.message().to_owned(),
        }
    }

    /// `NotFound` con formato "resource 'id' not found".
    pub fn not_found_f(resource: &str, id: &str) -> Self {
        if id.is_empty() {
            Self::NotFound(format!("{resource} not found"))
        } else {
            Self::NotFound(format!("{resource} '{id}' not found"))
        }
    }
}

/// Error HTTP canónico expuesto al cliente.
#[derive(Debug, Clone, Serialize)]
pub struct ApiError {
    pub code: String,
    pub message: String,
}

/// Envelope `{error: {code, message}}` para respuestas JSON.
#[derive(Debug, Clone, Serialize)]
pub struct ErrorEnvelope {
    pub error: ApiError,
}

/// Normaliza cualquier `DomainError` a `(StatusCode, ApiError)`.
pub fn normalize(err: &DomainError) -> (StatusCode, ApiError) {
    (err.status(), err.to_api_error())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn not_found_produces_404() {
        let err = DomainError::NotFound("party missing".into());
        assert_eq!(err.status(), StatusCode::NOT_FOUND);
        assert_eq!(err.code(), "NOT_FOUND");
        assert_eq!(err.message(), "party missing");
    }

    #[test]
    fn business_rule_produces_422() {
        let err = DomainError::BusinessRule("credit note is not active".into());
        assert_eq!(err.status(), StatusCode::UNPROCESSABLE_ENTITY);
        assert_eq!(err.code(), "BUSINESS_RULE");
    }

    #[test]
    fn not_found_f_formats_with_id() {
        let err = DomainError::not_found_f("party", "abc-123");
        assert_eq!(err.message(), "party 'abc-123' not found");
    }

    #[test]
    fn not_found_f_formats_without_id() {
        let err = DomainError::not_found_f("party", "");
        assert_eq!(err.message(), "party not found");
    }

    #[test]
    fn normalize_produces_correct_tuple() {
        let err = DomainError::Conflict("duplicate".into());
        let (status, api) = normalize(&err);
        assert_eq!(status, StatusCode::CONFLICT);
        assert_eq!(api.code, "CONFLICT");
        assert_eq!(api.message, "duplicate");
    }

    #[test]
    fn exhaustive_match() {
        // Compila solo si cubrimos todas las variantes
        let err = DomainError::Unauthorized("no token".into());
        match err {
            DomainError::Unauthorized(_) => {}
            DomainError::Forbidden(_) => {}
            DomainError::NotFound(_) => {}
            DomainError::Validation(_) => {}
            DomainError::Conflict(_) => {}
            DomainError::BusinessRule(_) => {}
            DomainError::Unavailable(_) => {}
            DomainError::UpstreamError(_) => {}
            DomainError::Internal(_) => {}
        }
    }
}
