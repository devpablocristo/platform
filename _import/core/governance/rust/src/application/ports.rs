use chrono::{DateTime, Utc};
use thiserror::Error;

use crate::domain::evidence::EvidencePack;
use crate::domain::model::{Policy, Request};

#[derive(Debug, Error, Clone, PartialEq, Eq)]
pub enum PolicyEvaluationError {
    #[error("unsupported policy expression: {0}")]
    UnsupportedExpression(String),
    #[error("policy evaluation failed: {0}")]
    EvaluationFailed(String),
}

#[derive(Debug, Error, Clone, PartialEq, Eq)]
pub enum EvidenceSigningError {
    #[error("signing key is required")]
    SigningKeyRequired,
    #[error("serialize evidence pack: {0}")]
    Serialization(String),
}

pub trait PolicyMatcher {
    fn matches(
        &self,
        request: &Request,
        policy: &Policy,
        now: DateTime<Utc>,
    ) -> Result<bool, PolicyEvaluationError>;
}

pub trait EvidenceSigner {
    fn sign(&self, pack: &mut EvidencePack) -> Result<(), EvidenceSigningError>;
}
