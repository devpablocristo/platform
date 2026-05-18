use chrono::{Duration, Utc};

use governance::domain::approval::{new_approval, requirement_for};
use governance::{
    approve, reject, ApprovalConfig, ApprovalRequirement, ApprovalStatus, Decision, Request,
    RiskLevel,
};

#[test]
fn requirement_enables_break_glass() {
    let config = ApprovalConfig::default();
    let requirement = requirement_for(
        &Request {
            action: "delete".into(),
            ..Request::default()
        },
        &Decision::RequireApproval,
        &RiskLevel::High,
        &config,
        Utc::now(),
    );

    assert!(requirement.required);
    assert!(requirement.break_glass);
    assert_eq!(requirement.required_approvals, 2);
}

#[test]
fn approve_break_glass_requires_quorum() {
    let approval = new_approval(
        "req-1",
        &ApprovalRequirement {
            required: true,
            break_glass: true,
            required_approvals: 2,
            expires_at: Some(Utc::now() + Duration::hours(1)),
        },
        Utc::now(),
    );

    let first = approve(&approval, "alice", "ok", Utc::now()).unwrap();
    assert!(!first.finalized);
    assert_eq!(first.approval.status, ApprovalStatus::Pending);

    let second = approve(&first.approval, "bob", "ok", Utc::now()).unwrap();
    assert!(second.finalized);
    assert_eq!(second.approval.status, ApprovalStatus::Approved);
}

#[test]
fn reject_finalizes_immediately() {
    let approval = new_approval(
        "req-1",
        &ApprovalRequirement {
            required: true,
            break_glass: true,
            required_approvals: 2,
            expires_at: Some(Utc::now() + Duration::hours(1)),
        },
        Utc::now(),
    );

    let rejected = reject(&approval, "alice", "no", Utc::now()).unwrap();
    assert_eq!(rejected.status, ApprovalStatus::Rejected);
}
