use chrono::{DateTime, Utc};
use thiserror::Error;

use crate::domain::evidence::{EvidencePack, TimelineEvent, EVIDENCE_PACK_VERSION};
use crate::domain::model::{ApprovalRecord, Evaluation, Request};

use super::ports::{EvidenceSigner, EvidenceSigningError};

#[derive(Debug, Error)]
pub enum EvidenceServiceError {
    #[error("evidence signing failed: {0}")]
    Signing(#[from] EvidenceSigningError),
}

pub struct EvidenceService<S> {
    signer: Option<S>,
}

impl<S> EvidenceService<S>
where
    S: EvidenceSigner,
{
    pub fn new(signer: Option<S>) -> Self {
        Self { signer }
    }

    pub fn build(
        &self,
        request: Request,
        evaluation: Evaluation,
        approval: Option<ApprovalRecord>,
        timeline: Vec<TimelineEvent>,
        now: DateTime<Utc>,
    ) -> Result<EvidencePack, EvidenceServiceError> {
        let mut pack = EvidencePack {
            version: EVIDENCE_PACK_VERSION.to_string(),
            generated_at: now,
            request,
            evaluation,
            approval,
            timeline,
            signature: None,
        };

        if let Some(signer) = &self.signer {
            signer.sign(&mut pack)?;
        }

        Ok(pack)
    }
}
