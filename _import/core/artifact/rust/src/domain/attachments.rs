use chrono::{DateTime, Duration, Utc};
use serde::{Deserialize, Serialize};
use thiserror::Error;

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct UploadRequest {
    pub storage_key: String,
    pub upload_url: String,
    pub expires_at: DateTime<Utc>,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct DownloadLink {
    pub url: String,
    pub expires_at: DateTime<Utc>,
}

#[derive(Debug, Error)]
pub enum AttachmentsError {
    #[error("{0}")]
    Validation(String),
}

pub fn build_storage_key(
    tenant_id: &str,
    attachable_type: &str,
    attachable_id: &str,
    file_name: &str,
    now: DateTime<Utc>,
) -> Result<String, AttachmentsError> {
    let tenant_id = tenant_id.trim();
    let attachable_type = sanitize_segment(attachable_type);
    let attachable_id = attachable_id.trim();

    if tenant_id.is_empty() || attachable_type.is_empty() || attachable_id.is_empty() {
        return Err(AttachmentsError::Validation(
            "tenant_id, attachable_type and attachable_id are required".to_string(),
        ));
    }

    let timestamp = now
        .timestamp_nanos_opt()
        .unwrap_or_else(|| now.timestamp_micros() * 1000);

    Ok(format!(
        "{tenant_id}/{attachable_type}/{attachable_id}/{timestamp}-{}",
        sanitize_file_name(file_name)
    ))
}

pub fn request_upload(
    base_url: &str,
    storage_key: &str,
    ttl: Option<Duration>,
    now: DateTime<Utc>,
) -> Result<UploadRequest, AttachmentsError> {
    let storage_key = storage_key.trim();
    if storage_key.is_empty() {
        return Err(AttachmentsError::Validation(
            "storage_key is required".to_string(),
        ));
    }

    let ttl = ttl.unwrap_or_else(|| Duration::minutes(15));
    let base_url = base_url.trim().trim_end_matches('/');

    Ok(UploadRequest {
        storage_key: storage_key.to_string(),
        upload_url: format!("{base_url}/uploads/{storage_key}"),
        expires_at: now + ttl,
    })
}

pub fn build_download_link(
    base_url: &str,
    id: &str,
    ttl: Option<Duration>,
    now: DateTime<Utc>,
) -> Result<DownloadLink, AttachmentsError> {
    let id = id.trim();
    if id.is_empty() {
        return Err(AttachmentsError::Validation(
            "attachment id is required".to_string(),
        ));
    }

    let ttl = ttl.unwrap_or_else(|| Duration::minutes(15));
    let base_url = base_url.trim().trim_end_matches('/');

    Ok(DownloadLink {
        url: format!("{base_url}/attachments/{id}/download"),
        expires_at: now + ttl,
    })
}

pub fn sanitize_file_name(value: &str) -> String {
    let trimmed = value.trim();
    if trimmed.is_empty() {
        return "file.bin".to_string();
    }

    trimmed.replace("..", "").replace(['/', '\\'], "-")
}

fn sanitize_segment(value: &str) -> String {
    value.trim().to_lowercase().replace(['/', '\\', ' '], "-")
}
