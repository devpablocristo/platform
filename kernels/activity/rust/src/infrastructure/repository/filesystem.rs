use std::fs::{self, File, OpenOptions};
use std::io::{Read, Seek, SeekFrom, Write};
use std::path::{Path, PathBuf};

use fs2::FileExt;
use serde::de::DeserializeOwned;

use crate::application::ports::{AuditRepository, RepositoryError, TimelineRepository};
use crate::domain::audit::AuditEntry;
use crate::domain::timeline::TimelineEntry;

#[derive(Debug, Clone)]
pub struct FileSystemAuditRepository {
    root: PathBuf,
}

impl FileSystemAuditRepository {
    pub fn new(root: impl Into<PathBuf>) -> Self {
        Self { root: root.into() }
    }

    fn tenant_path(&self, tenant_id: &str) -> PathBuf {
        self.root
            .join("audit")
            .join(format!("{}.jsonl", safe_segment(tenant_id)))
    }
}

impl AuditRepository for FileSystemAuditRepository {
    fn last_hash(&self, tenant_id: &str) -> Result<String, RepositoryError> {
        let path = self.tenant_path(tenant_id);
        let items = read_jsonl::<AuditEntry>(path.as_path())?;
        Ok(items
            .last()
            .map(|entry| entry.hash.clone())
            .unwrap_or_default())
    }

    fn append(&self, entry: AuditEntry) -> Result<AuditEntry, RepositoryError> {
        let path = self.tenant_path(entry.tenant_id.as_str());
        ensure_parent(path.as_path())?;

        let mut file = OpenOptions::new()
            .read(true)
            .append(true)
            .create(true)
            .open(path.as_path())
            .map_err(|error| RepositoryError::Io(error.to_string()))?;
        file.lock_exclusive()
            .map_err(|error| RepositoryError::Io(error.to_string()))?;

        let result = append_audit_entry(&mut file, &entry);
        let unlock_result = file.unlock();
        if let Err(error) = unlock_result {
            return Err(RepositoryError::Io(error.to_string()));
        }
        result?;
        Ok(entry)
    }

    fn list(&self, tenant_id: &str, limit: usize) -> Result<Vec<AuditEntry>, RepositoryError> {
        let path = self.tenant_path(tenant_id);
        let mut items = read_jsonl::<AuditEntry>(path.as_path())?;
        if limit > 0 && items.len() > limit {
            items.truncate(limit);
        }
        Ok(items)
    }
}

#[derive(Debug, Clone)]
pub struct FileSystemTimelineRepository {
    root: PathBuf,
}

impl FileSystemTimelineRepository {
    pub fn new(root: impl Into<PathBuf>) -> Self {
        Self { root: root.into() }
    }

    fn tenant_path(&self, tenant_id: &str) -> PathBuf {
        self.root
            .join("timeline")
            .join(format!("{}.jsonl", safe_segment(tenant_id)))
    }
}

impl TimelineRepository for FileSystemTimelineRepository {
    fn append(&self, entry: TimelineEntry) -> Result<TimelineEntry, RepositoryError> {
        let path = self.tenant_path(entry.tenant_id.as_str());
        ensure_parent(path.as_path())?;

        let mut file = OpenOptions::new()
            .read(true)
            .append(true)
            .create(true)
            .open(path.as_path())
            .map_err(|error| RepositoryError::Io(error.to_string()))?;
        file.lock_exclusive()
            .map_err(|error| RepositoryError::Io(error.to_string()))?;

        let result = append_json_line(&mut file, &entry);
        let unlock_result = file.unlock();
        if let Err(error) = unlock_result {
            return Err(RepositoryError::Io(error.to_string()));
        }
        result?;
        Ok(entry)
    }

    fn list(
        &self,
        tenant_id: &str,
        entity_type: &str,
        entity_id: &str,
        limit: usize,
    ) -> Result<Vec<TimelineEntry>, RepositoryError> {
        let path = self.tenant_path(tenant_id);
        let mut items = read_jsonl::<TimelineEntry>(path.as_path())?
            .into_iter()
            .filter(|entry| entry.entity_type == entity_type && entry.entity_id == entity_id)
            .collect::<Vec<_>>();
        if limit > 0 && items.len() > limit {
            items.truncate(limit);
        }
        Ok(items)
    }
}

fn append_audit_entry(file: &mut File, entry: &AuditEntry) -> Result<(), RepositoryError> {
    let items = read_jsonl_from_file::<AuditEntry>(file)?;
    let current_hash = items
        .last()
        .map(|item| item.hash.as_str())
        .unwrap_or_default();
    if current_hash != entry.prev_hash {
        return Err(RepositoryError::Conflict(format!(
            "expected prev_hash {:?}, found {:?}",
            entry.prev_hash, current_hash
        )));
    }

    append_json_line(file, entry)
}

fn append_json_line<T>(file: &mut File, value: &T) -> Result<(), RepositoryError>
where
    T: serde::Serialize,
{
    file.seek(SeekFrom::End(0))
        .map_err(|error| RepositoryError::Io(error.to_string()))?;
    serde_json::to_writer(&mut *file, value)
        .map_err(|error| RepositoryError::Serialization(error.to_string()))?;
    file.write_all(b"\n")
        .map_err(|error| RepositoryError::Io(error.to_string()))?;
    file.flush()
        .map_err(|error| RepositoryError::Io(error.to_string()))?;
    file.sync_data()
        .map_err(|error| RepositoryError::Io(error.to_string()))
}

fn read_jsonl<T>(path: &Path) -> Result<Vec<T>, RepositoryError>
where
    T: DeserializeOwned,
{
    if !path.exists() {
        return Ok(Vec::new());
    }

    let mut file = File::open(path).map_err(|error| RepositoryError::Io(error.to_string()))?;
    read_jsonl_from_file(&mut file)
}

fn read_jsonl_from_file<T>(file: &mut File) -> Result<Vec<T>, RepositoryError>
where
    T: DeserializeOwned,
{
    file.seek(SeekFrom::Start(0))
        .map_err(|error| RepositoryError::Io(error.to_string()))?;
    let mut body = String::new();
    file.read_to_string(&mut body)
        .map_err(|error| RepositoryError::Io(error.to_string()))?;

    let mut items = Vec::new();
    for line in body.lines().filter(|line| !line.trim().is_empty()) {
        items.push(
            serde_json::from_str(line)
                .map_err(|error| RepositoryError::Serialization(error.to_string()))?,
        );
    }
    Ok(items)
}

fn ensure_parent(path: &Path) -> Result<(), RepositoryError> {
    let Some(parent) = path.parent() else {
        return Ok(());
    };
    fs::create_dir_all(parent).map_err(|error| RepositoryError::Io(error.to_string()))
}

fn safe_segment(value: &str) -> String {
    let mut output = String::with_capacity(value.len());
    for ch in value.trim().chars() {
        if ch.is_ascii_alphanumeric() || matches!(ch, '-' | '_' | '.') {
            output.push(ch);
        } else {
            output.push('_');
        }
    }

    if output.is_empty() {
        "default".to_string()
    } else {
        output
    }
}
