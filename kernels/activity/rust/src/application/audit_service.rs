use crate::domain::audit::{
    actor_label, build_hash, export_csv, export_jsonl, normalize_actor_type, sanitize_payload,
    AuditEntry, AuditExportError, AuditLogInput,
};
use crate::domain::shared::first_non_empty;

use super::ports::{AuditRepository, Clock, IdGenerator, RepositoryError};
use thiserror::Error;

#[derive(Debug, Error)]
pub enum AuditServiceError {
    #[error("{0}")]
    Validation(String),
    #[error(transparent)]
    Repository(#[from] RepositoryError),
    #[error(transparent)]
    Export(#[from] AuditExportError),
    #[error("hash build failed: {0}")]
    Hash(String),
}

pub struct AuditService<R, I, C> {
    repo: R,
    id_generator: I,
    clock: C,
}

impl<R, I, C> AuditService<R, I, C>
where
    R: AuditRepository,
    I: IdGenerator,
    C: Clock,
{
    pub fn new(repo: R, id_generator: I, clock: C) -> Self {
        Self {
            repo,
            id_generator,
            clock,
        }
    }

    pub fn append(&self, input: AuditLogInput) -> Result<AuditEntry, AuditServiceError> {
        let tenant_id = require_non_empty("tenant_id", input.tenant_id.as_str())?;
        let action = require_non_empty("action", input.action.as_str())?;
        let resource_type = require_non_empty("resource_type", input.resource_type.as_str())?;
        let prev_hash = self.repo.last_hash(tenant_id.as_str())?;

        let mut entry = AuditEntry {
            id: first_non_empty(&[input.id.as_str(), self.id_generator.new_id().as_str()]),
            tenant_id,
            actor: input.actor.legacy.trim().to_string(),
            actor_type: normalize_actor_type(input.actor.actor_type.as_str()),
            actor_id: input.actor.id.trim().to_string(),
            actor_label: actor_label(&input.actor),
            action,
            resource_type,
            resource_id: input.resource_id.trim().to_string(),
            payload: sanitize_payload(&input.payload),
            prev_hash,
            hash: String::new(),
            created_at: input.created_at.unwrap_or_else(|| self.clock.now()),
        };
        entry.hash =
            build_hash(&entry).map_err(|error| AuditServiceError::Hash(error.to_string()))?;
        self.repo.append(entry).map_err(Into::into)
    }

    pub fn list(
        &self,
        tenant_id: &str,
        limit: usize,
    ) -> Result<Vec<AuditEntry>, AuditServiceError> {
        let tenant_id = require_non_empty("tenant_id", tenant_id)?;
        self.repo
            .list(tenant_id.as_str(), limit)
            .map_err(Into::into)
    }

    pub fn export_csv(&self, tenant_id: &str, limit: usize) -> Result<String, AuditServiceError> {
        let items = self.list(tenant_id, limit)?;
        export_csv(&items).map_err(Into::into)
    }

    pub fn export_jsonl(&self, tenant_id: &str, limit: usize) -> Result<String, AuditServiceError> {
        let items = self.list(tenant_id, limit)?;
        export_jsonl(&items).map_err(Into::into)
    }
}

fn require_non_empty(field: &str, value: &str) -> Result<String, AuditServiceError> {
    let trimmed = value.trim();
    if trimmed.is_empty() {
        return Err(AuditServiceError::Validation(format!(
            "{field} is required"
        )));
    }
    Ok(trimmed.to_string())
}
