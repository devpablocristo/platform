use std::sync::{Mutex, MutexGuard};

use crate::application::ports::{AuditRepository, RepositoryError, TimelineRepository};
use crate::domain::audit::AuditEntry;
use crate::domain::timeline::TimelineEntry;

#[derive(Debug, Default)]
pub struct InMemoryAuditRepository {
    items: Mutex<Vec<AuditEntry>>,
}

impl AuditRepository for InMemoryAuditRepository {
    fn last_hash(&self, tenant_id: &str) -> Result<String, RepositoryError> {
        let items = lock(&self.items)?;
        Ok(items
            .iter()
            .rev()
            .find(|entry| entry.tenant_id == tenant_id)
            .map(|entry| entry.hash.clone())
            .unwrap_or_default())
    }

    fn append(&self, entry: AuditEntry) -> Result<AuditEntry, RepositoryError> {
        let mut items = lock(&self.items)?;
        items.push(entry.clone());
        Ok(entry)
    }

    fn list(&self, tenant_id: &str, limit: usize) -> Result<Vec<AuditEntry>, RepositoryError> {
        let items = lock(&self.items)?;
        let mut filtered = items
            .iter()
            .filter(|entry| entry.tenant_id == tenant_id)
            .cloned()
            .collect::<Vec<_>>();
        if limit > 0 && filtered.len() > limit {
            filtered.truncate(limit);
        }
        Ok(filtered)
    }
}

#[derive(Debug, Default)]
pub struct InMemoryTimelineRepository {
    items: Mutex<Vec<TimelineEntry>>,
}

impl TimelineRepository for InMemoryTimelineRepository {
    fn append(&self, entry: TimelineEntry) -> Result<TimelineEntry, RepositoryError> {
        let mut items = lock(&self.items)?;
        items.push(entry.clone());
        Ok(entry)
    }

    fn list(
        &self,
        tenant_id: &str,
        entity_type: &str,
        entity_id: &str,
        limit: usize,
    ) -> Result<Vec<TimelineEntry>, RepositoryError> {
        let items = lock(&self.items)?;
        let mut filtered = items
            .iter()
            .filter(|entry| {
                entry.tenant_id == tenant_id
                    && entry.entity_type == entity_type
                    && entry.entity_id == entity_id
            })
            .cloned()
            .collect::<Vec<_>>();
        if limit > 0 && filtered.len() > limit {
            filtered.truncate(limit);
        }
        Ok(filtered)
    }
}

fn lock<T>(mutex: &Mutex<T>) -> Result<MutexGuard<'_, T>, RepositoryError> {
    mutex
        .lock()
        .map_err(|_| RepositoryError::Operation("mutex poisoned".to_string()))
}
