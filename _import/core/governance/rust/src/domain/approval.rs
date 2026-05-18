use chrono::{DateTime, Duration, Utc};
use thiserror::Error;

use super::model::{
    ApprovalAction, ApprovalDecisionRecord, ApprovalRecord, ApprovalRequirement, ApprovalStatus,
    Decision, Request, RiskLevel,
};

#[derive(Debug, Clone, PartialEq)]
pub struct ApprovalConfig {
    pub default_ttl: Duration,
    pub break_glass_default: u32,
    pub break_glass_rules: Vec<BreakGlassRule>,
}

#[derive(Debug, Clone, PartialEq)]
pub struct BreakGlassRule {
    pub actions: Vec<String>,
    pub risk_level: Option<RiskLevel>,
    pub required_approvals: u32,
}

#[derive(Debug, Clone, PartialEq)]
pub struct ApprovalOutcome {
    pub approval: ApprovalRecord,
    pub finalized: bool,
}

#[derive(Debug, Error, Clone, PartialEq, Eq)]
pub enum ApprovalError {
    #[error("approval is not pending")]
    NotPending,
    #[error("approver already decided")]
    ApproverAlreadyDecided,
}

impl Default for ApprovalConfig {
    fn default() -> Self {
        Self {
            default_ttl: Duration::hours(1),
            break_glass_default: 2,
            break_glass_rules: vec![
                BreakGlassRule {
                    actions: vec!["delete".into()],
                    risk_level: Some(RiskLevel::High),
                    required_approvals: 2,
                },
                BreakGlassRule {
                    actions: vec!["runbook.execute".into()],
                    risk_level: Some(RiskLevel::High),
                    required_approvals: 2,
                },
            ],
        }
    }
}

pub fn requirement_for(
    request: &Request,
    decision: &Decision,
    risk_tier: &RiskLevel,
    config: &ApprovalConfig,
    now: DateTime<Utc>,
) -> ApprovalRequirement {
    if !matches!(decision, Decision::RequireApproval) {
        return ApprovalRequirement::default();
    }

    let mut required_approvals = 1_u32;
    for rule in &config.break_glass_rules {
        if matches_rule(rule, request.action.as_str(), risk_tier) {
            required_approvals = rule.required_approvals.max(1);
            break;
        }
    }

    let break_glass = required_approvals > 1;
    let required_approvals = if break_glass {
        required_approvals.max(config.break_glass_default.max(2))
    } else {
        required_approvals.max(1)
    };

    ApprovalRequirement {
        required: true,
        break_glass,
        required_approvals,
        expires_at: Some(now + config.default_ttl),
    }
}

pub fn new_approval(
    request_id: impl Into<String>,
    requirement: &ApprovalRequirement,
    now: DateTime<Utc>,
) -> ApprovalRecord {
    ApprovalRecord {
        request_id: request_id.into(),
        status: ApprovalStatus::Pending,
        decided_by: String::new(),
        decision_note: String::new(),
        decided_at: None,
        expires_at: requirement.expires_at,
        created_at: now,
        break_glass: requirement.break_glass,
        required_approvals: requirement.required_approvals.max(1),
        decisions: Vec::new(),
    }
}

pub fn approve(
    approval: &ApprovalRecord,
    approver_id: impl Into<String>,
    note: impl Into<String>,
    now: DateTime<Utc>,
) -> Result<ApprovalOutcome, ApprovalError> {
    if !matches!(approval.status, ApprovalStatus::Pending) {
        return Err(ApprovalError::NotPending);
    }

    let approver_id = approver_id.into().trim().to_string();
    if has_approver(approval, approver_id.as_str()) {
        return Err(ApprovalError::ApproverAlreadyDecided);
    }

    let mut updated = approval.clone();
    updated.decisions.push(ApprovalDecisionRecord {
        approver_id: approver_id.clone(),
        action: ApprovalAction::Approve,
        note: note.into().trim().to_string(),
        decided_at: now,
    });

    let approved_count = updated
        .decisions
        .iter()
        .filter(|decision| matches!(decision.action, ApprovalAction::Approve))
        .count() as u32;

    if updated.break_glass && approved_count < updated.required_approvals {
        return Ok(ApprovalOutcome {
            approval: updated,
            finalized: false,
        });
    }

    updated.status = ApprovalStatus::Approved;
    updated.decided_by = approver_id;
    updated.decision_note = updated
        .decisions
        .last()
        .map(|decision| decision.note.clone())
        .unwrap_or_default();
    updated.decided_at = Some(now);

    Ok(ApprovalOutcome {
        approval: updated,
        finalized: true,
    })
}

pub fn reject(
    approval: &ApprovalRecord,
    approver_id: impl Into<String>,
    note: impl Into<String>,
    now: DateTime<Utc>,
) -> Result<ApprovalRecord, ApprovalError> {
    if !matches!(approval.status, ApprovalStatus::Pending) {
        return Err(ApprovalError::NotPending);
    }

    let approver_id = approver_id.into().trim().to_string();
    if approval.break_glass && has_approver(approval, approver_id.as_str()) {
        return Err(ApprovalError::ApproverAlreadyDecided);
    }

    let mut updated = approval.clone();
    updated.decisions.push(ApprovalDecisionRecord {
        approver_id: approver_id.clone(),
        action: ApprovalAction::Reject,
        note: note.into().trim().to_string(),
        decided_at: now,
    });
    updated.status = ApprovalStatus::Rejected;
    updated.decided_by = approver_id;
    updated.decision_note = updated
        .decisions
        .last()
        .map(|decision| decision.note.clone())
        .unwrap_or_default();
    updated.decided_at = Some(now);

    Ok(updated)
}

fn matches_rule(rule: &BreakGlassRule, action: &str, risk_tier: &RiskLevel) -> bool {
    let matches_action = rule.actions.is_empty()
        || rule
            .actions
            .iter()
            .any(|candidate| candidate.trim() == action.trim());
    if !matches_action {
        return false;
    }

    match &rule.risk_level {
        Some(level) => level == risk_tier,
        None => true,
    }
}

fn has_approver(approval: &ApprovalRecord, approver_id: &str) -> bool {
    approval
        .decisions
        .iter()
        .any(|decision| decision.approver_id.trim() == approver_id.trim())
}
