pub mod application;
pub mod domain;
pub mod infrastructure;

pub use application::decision_engine::{DecisionEngine, DecisionEngineError, DecisionInput};
pub use application::evidence_service::{EvidenceService, EvidenceServiceError};
pub use application::ports::{EvidenceSigner, PolicyEvaluationError, PolicyMatcher};
pub use domain::approval::{
    approve, reject, ApprovalConfig, ApprovalError, ApprovalOutcome, BreakGlassRule,
};
pub use domain::evidence::{EvidencePack, TimelineEvent, EVIDENCE_PACK_VERSION};
pub use domain::model::*;
pub use domain::risk::{
    evaluate_risk, tier_for_action, DecisionRecommendation, RiskConfig, RiskHistory,
};
pub use infrastructure::policy::cel_matcher::CelPolicyMatcher;
pub use infrastructure::policy::deterministic_matcher::DeterministicPolicyMatcher;
pub use infrastructure::signing::hmac_sha256::HmacSha256Signer;
