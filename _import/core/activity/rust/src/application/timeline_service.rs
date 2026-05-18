use crate::domain::shared::{first_non_empty, trim_json_map};
use crate::domain::timeline::{TimelineEntry, TimelineRecordInput};

use super::ports::{Clock, IdGenerator, RepositoryError, TimelineRepository};
use thiserror::Error;

#[derive(Debug, Error)]
pub enum TimelineServiceError {
    #[error("{0}")]
    Validation(String),
    #[error(transparent)]
    Repository(#[from] RepositoryError),
}

pub struct TimelineService<R, I, C> {
    repo: R,
    id_generator: I,
    clock: C,
}

impl<R, I, C> TimelineService<R, I, C>
where
    R: TimelineRepository,
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

    pub fn list(
        &self,
        tenant_id: &str,
        entity_type: &str,
        entity_id: &str,
        limit: usize,
    ) -> Result<Vec<TimelineEntry>, TimelineServiceError> {
        let tenant_id = require_non_empty("tenant_id", tenant_id)?;
        let entity_type = require_non_empty("entity_type", entity_type)?;
        let entity_id = require_non_empty("entity_id", entity_id)?;
        self.repo
            .list(
                tenant_id.as_str(),
                entity_type.as_str(),
                entity_id.as_str(),
                limit,
            )
            .map_err(Into::into)
    }

    pub fn record(
        &self,
        input: TimelineRecordInput,
    ) -> Result<TimelineEntry, TimelineServiceError> {
        let entry = TimelineEntry {
            id: first_non_empty(&[input.id.as_str(), self.id_generator.new_id().as_str()]),
            tenant_id: require_non_empty("tenant_id", input.tenant_id.as_str())?,
            entity_type: require_non_empty("entity_type", input.entity_type.as_str())?,
            entity_id: require_non_empty("entity_id", input.entity_id.as_str())?,
            event_type: require_non_empty("event_type", input.event_type.as_str())?,
            title: require_non_empty("title", input.title.as_str())?,
            description: input.description.trim().to_string(),
            actor: input.actor.trim().to_string(),
            metadata: trim_json_map(&input.metadata),
            created_at: input.created_at.unwrap_or_else(|| self.clock.now()),
        };
        self.repo.append(entry).map_err(Into::into)
    }
}

fn require_non_empty(field: &str, value: &str) -> Result<String, TimelineServiceError> {
    let trimmed = value.trim();
    if trimmed.is_empty() {
        return Err(TimelineServiceError::Validation(format!(
            "{field} is required"
        )));
    }
    Ok(trimmed.to_string())
}
