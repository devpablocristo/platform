use std::collections::BTreeMap;

use chrono::{DateTime, TimeZone, Utc};
use serde::{Deserialize, Serialize};
use serde_json::Value;

fn unix_epoch() -> DateTime<Utc> {
    Utc.timestamp_opt(0, 0)
        .single()
        .expect("unix epoch must be representable")
}

#[derive(Debug, Clone, PartialEq, Eq, Default, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum RequesterType {
    Agent,
    Service,
    #[default]
    Human,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum RiskLevel {
    Low,
    Medium,
    High,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum Decision {
    Allow,
    Deny,
    RequireApproval,
}

#[derive(Debug, Clone, PartialEq, Eq, Default, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum PolicyMode {
    #[default]
    Enforce,
    Shadow,
}

#[derive(Debug, Clone, PartialEq, Eq, Default, Serialize, Deserialize)]
pub struct Subject {
    #[serde(rename = "type")]
    pub subject_type: RequesterType,
    pub id: String,
    #[serde(skip_serializing_if = "String::is_empty", default)]
    pub name: String,
}

#[derive(Debug, Clone, PartialEq, Eq, Default, Serialize, Deserialize)]
pub struct Target {
    #[serde(skip_serializing_if = "String::is_empty", default)]
    pub system: String,
    #[serde(skip_serializing_if = "String::is_empty", default)]
    pub resource: String,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct Request {
    #[serde(skip_serializing_if = "String::is_empty", default)]
    pub id: String,
    pub subject: Subject,
    pub action: String,
    pub target: Target,
    #[serde(skip_serializing_if = "BTreeMap::is_empty", default)]
    pub params: BTreeMap<String, Value>,
    #[serde(skip_serializing_if = "BTreeMap::is_empty", default)]
    pub metadata: BTreeMap<String, Value>,
    #[serde(skip_serializing_if = "String::is_empty", default)]
    pub reason: String,
    #[serde(skip_serializing_if = "String::is_empty", default)]
    pub context: String,
    pub created_at: DateTime<Utc>,
}

impl Default for Request {
    fn default() -> Self {
        Self {
            id: String::new(),
            subject: Subject::default(),
            action: String::new(),
            target: Target::default(),
            params: BTreeMap::new(),
            metadata: BTreeMap::new(),
            reason: String::new(),
            context: String::new(),
            created_at: unix_epoch(),
        }
    }
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct Policy {
    pub id: String,
    pub name: String,
    pub expression: String,
    pub effect: Decision,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub risk_override: Option<RiskLevel>,
    pub priority: i32,
    #[serde(default)]
    pub mode: PolicyMode,
    #[serde(default)]
    pub enabled: bool,
    #[serde(skip_serializing_if = "String::is_empty", default)]
    pub action_filter: String,
    #[serde(skip_serializing_if = "String::is_empty", default)]
    pub system_filter: String,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct RiskFactor {
    pub name: String,
    pub score: f64,
    pub active: bool,
    #[serde(skip_serializing_if = "String::is_empty", default)]
    pub reason: String,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct RiskAssessment {
    pub factors: Vec<RiskFactor>,
    pub raw_score: f64,
    pub amplification: f64,
    pub final_score: f64,
    pub level: RiskLevel,
    pub recommended: Decision,
}

#[derive(Debug, Clone, PartialEq, Default, Serialize, Deserialize)]
pub struct ApprovalRequirement {
    pub required: bool,
    pub break_glass: bool,
    pub required_approvals: u32,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub expires_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct Evaluation {
    #[serde(skip_serializing_if = "String::is_empty", default)]
    pub request_id: String,
    pub evaluated_at: DateTime<Utc>,
    pub decision: Decision,
    pub decision_reason: String,
    #[serde(skip_serializing_if = "String::is_empty", default)]
    pub policy_id: String,
    #[serde(skip_serializing_if = "String::is_empty", default)]
    pub policy_name: String,
    #[serde(skip_serializing_if = "Vec::is_empty", default)]
    pub shadow_policies: Vec<String>,
    pub risk_tier: RiskLevel,
    pub risk: RiskAssessment,
    pub approval: ApprovalRequirement,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum ApprovalStatus {
    Pending,
    Approved,
    Rejected,
    Expired,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum ApprovalAction {
    Approve,
    Reject,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct ApprovalDecisionRecord {
    pub approver_id: String,
    pub action: ApprovalAction,
    #[serde(skip_serializing_if = "String::is_empty", default)]
    pub note: String,
    pub decided_at: DateTime<Utc>,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct ApprovalRecord {
    pub request_id: String,
    pub status: ApprovalStatus,
    #[serde(skip_serializing_if = "String::is_empty", default)]
    pub decided_by: String,
    #[serde(skip_serializing_if = "String::is_empty", default)]
    pub decision_note: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub decided_at: Option<DateTime<Utc>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub expires_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
    pub break_glass: bool,
    pub required_approvals: u32,
    #[serde(skip_serializing_if = "Vec::is_empty", default)]
    pub decisions: Vec<ApprovalDecisionRecord>,
}
