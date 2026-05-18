use std::collections::BTreeMap;

use chrono::{DateTime, Utc};
use csv::Writer;
use serde::{Deserialize, Serialize};
use serde_json::Value;
use sha2::{Digest, Sha256};
use thiserror::Error;

use super::shared::{trim_json_map, ActorRef};

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct AuditEntry {
    pub id: String,
    pub tenant_id: String,
    #[serde(skip_serializing_if = "String::is_empty", default)]
    pub actor: String,
    #[serde(skip_serializing_if = "String::is_empty", default)]
    pub actor_type: String,
    #[serde(skip_serializing_if = "String::is_empty", default)]
    pub actor_id: String,
    #[serde(skip_serializing_if = "String::is_empty", default)]
    pub actor_label: String,
    pub action: String,
    pub resource_type: String,
    #[serde(skip_serializing_if = "String::is_empty", default)]
    pub resource_id: String,
    #[serde(skip_serializing_if = "BTreeMap::is_empty", default)]
    pub payload: BTreeMap<String, Value>,
    #[serde(skip_serializing_if = "String::is_empty", default)]
    pub prev_hash: String,
    pub hash: String,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct AuditLogInput {
    #[serde(skip_serializing_if = "String::is_empty", default)]
    pub id: String,
    pub tenant_id: String,
    #[serde(default)]
    pub actor: ActorRef,
    pub action: String,
    pub resource_type: String,
    #[serde(skip_serializing_if = "String::is_empty", default)]
    pub resource_id: String,
    #[serde(skip_serializing_if = "BTreeMap::is_empty", default)]
    pub payload: BTreeMap<String, Value>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub created_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Error)]
pub enum AuditExportError {
    #[error("csv export failed: {0}")]
    Csv(#[from] csv::Error),
    #[error("json export failed: {0}")]
    Json(#[from] serde_json::Error),
    #[error("utf-8 export failed: {0}")]
    Utf8(#[from] std::string::FromUtf8Error),
}

pub fn build_hash(entry: &AuditEntry) -> Result<String, serde_json::Error> {
    #[derive(Serialize)]
    struct HashMaterial<'a> {
        #[serde(skip_serializing_if = "str::is_empty")]
        prev_hash: &'a str,
        #[serde(skip_serializing_if = "str::is_empty")]
        actor: &'a str,
        #[serde(skip_serializing_if = "str::is_empty")]
        actor_type: &'a str,
        #[serde(skip_serializing_if = "str::is_empty")]
        actor_id: &'a str,
        #[serde(skip_serializing_if = "str::is_empty")]
        actor_label: &'a str,
        action: &'a str,
        resource_type: &'a str,
        #[serde(skip_serializing_if = "str::is_empty")]
        resource_id: &'a str,
        #[serde(skip_serializing_if = "BTreeMap::is_empty")]
        payload: &'a BTreeMap<String, Value>,
    }

    let body = serde_json::to_vec(&HashMaterial {
        prev_hash: entry.prev_hash.as_str(),
        actor: entry.actor.as_str(),
        actor_type: entry.actor_type.as_str(),
        actor_id: entry.actor_id.as_str(),
        actor_label: entry.actor_label.as_str(),
        action: entry.action.as_str(),
        resource_type: entry.resource_type.as_str(),
        resource_id: entry.resource_id.as_str(),
        payload: &entry.payload,
    })?;
    let digest = Sha256::digest(body);
    Ok(hex::encode(digest))
}

pub fn normalize_actor_type(raw: &str) -> String {
    match raw.trim().to_lowercase().as_str() {
        "party" => "party".to_string(),
        "service" => "service".to_string(),
        "system" => "system".to_string(),
        _ => "user".to_string(),
    }
}

pub fn actor_label(actor: &ActorRef) -> String {
    if !actor.label.trim().is_empty() {
        return actor.label.trim().to_string();
    }
    actor.legacy.trim().to_string()
}

pub fn sanitize_payload(input: &BTreeMap<String, Value>) -> BTreeMap<String, Value> {
    trim_json_map(input)
}

pub fn export_csv(entries: &[AuditEntry]) -> Result<String, AuditExportError> {
    let mut writer = Writer::from_writer(Vec::new());
    writer.write_record([
        "id",
        "tenant_id",
        "actor",
        "actor_type",
        "actor_id",
        "actor_label",
        "action",
        "resource_type",
        "resource_id",
        "prev_hash",
        "hash",
        "created_at",
        "payload",
    ])?;

    for entry in entries {
        writer.write_record([
            entry.id.as_str(),
            entry.tenant_id.as_str(),
            entry.actor.as_str(),
            entry.actor_type.as_str(),
            entry.actor_id.as_str(),
            entry.actor_label.as_str(),
            entry.action.as_str(),
            entry.resource_type.as_str(),
            entry.resource_id.as_str(),
            entry.prev_hash.as_str(),
            entry.hash.as_str(),
            entry.created_at.to_rfc3339().as_str(),
            serde_json::to_string(&entry.payload)?.as_str(),
        ])?;
    }

    let bytes = writer
        .into_inner()
        .map_err(|error| AuditExportError::Csv(error.into_error().into()))?;
    Ok(String::from_utf8(bytes)?)
}

pub fn export_jsonl(entries: &[AuditEntry]) -> Result<String, AuditExportError> {
    let mut lines = Vec::with_capacity(entries.len());
    for entry in entries {
        lines.push(serde_json::to_string(entry)?);
    }
    Ok(lines.join("\n"))
}
