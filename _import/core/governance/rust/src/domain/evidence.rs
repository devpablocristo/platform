use std::collections::BTreeMap;

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use serde_json::Value;

use super::model::{ApprovalRecord, Evaluation, Request};

pub const EVIDENCE_PACK_VERSION: &str = "1.0";

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct TimelineEvent {
    pub event: String,
    pub actor: String,
    pub at: DateTime<Utc>,
    pub summary: String,
    #[serde(skip_serializing_if = "BTreeMap::is_empty", default)]
    pub data: BTreeMap<String, Value>,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct EvidencePack {
    pub version: String,
    pub generated_at: DateTime<Utc>,
    pub request: Request,
    pub evaluation: Evaluation,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub approval: Option<ApprovalRecord>,
    #[serde(skip_serializing_if = "Vec::is_empty", default)]
    pub timeline: Vec<TimelineEvent>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub signature: Option<String>,
}
