use chrono::{DateTime, Utc};
use thiserror::Error;

use crate::domain::audit::AuditEntry;
use crate::domain::timeline::TimelineEntry;

#[derive(Debug, Error, Clone, PartialEq, Eq)]
pub enum RepositoryError {
    #[error("repository conflict: {0}")]
    Conflict(String),
    #[error("repository io: {0}")]
    Io(String),
    #[error("repository serialization: {0}")]
    Serialization(String),
    #[error("{0}")]
    Operation(String),
}

pub trait AuditRepository {
    fn last_hash(&self, tenant_id: &str) -> Result<String, RepositoryError>;
    fn append(&self, entry: AuditEntry) -> Result<AuditEntry, RepositoryError>;
    fn list(&self, tenant_id: &str, limit: usize) -> Result<Vec<AuditEntry>, RepositoryError>;
}

pub trait TimelineRepository {
    fn append(&self, entry: TimelineEntry) -> Result<TimelineEntry, RepositoryError>;
    fn list(
        &self,
        tenant_id: &str,
        entity_type: &str,
        entity_id: &str,
        limit: usize,
    ) -> Result<Vec<TimelineEntry>, RepositoryError>;
}

pub trait IdGenerator {
    fn new_id(&self) -> String;
}

pub trait Clock {
    fn now(&self) -> DateTime<Utc>;
}
