use chrono::{TimeZone, Utc};

use governance::{
    domain::approval::new_approval, ApprovalRequirement, ApprovalStatus, Decision, Evaluation,
    EvidenceService, HmacSha256Signer, Request, RiskAssessment, RiskLevel, TimelineEvent,
};

#[test]
fn build_signs_pack() {
    let signer = HmacSha256Signer::new("secret", "kid-1").unwrap();
    let service = EvidenceService::new(Some(signer));

    let now = Utc
        .with_ymd_and_hms(2026, 3, 20, 12, 0, 0)
        .single()
        .unwrap();

    let pack = service
        .build(
            Request {
                id: "req-1".into(),
                action: "delete".into(),
                ..Request::default()
            },
            Evaluation {
                request_id: "req-1".into(),
                evaluated_at: now,
                decision: Decision::RequireApproval,
                decision_reason: "policy".into(),
                policy_id: String::new(),
                policy_name: String::new(),
                shadow_policies: Vec::new(),
                risk_tier: RiskLevel::High,
                risk: RiskAssessment {
                    factors: Vec::new(),
                    raw_score: 0.0,
                    amplification: 1.0,
                    final_score: 0.0,
                    level: RiskLevel::High,
                    recommended: Decision::RequireApproval,
                },
                approval: ApprovalRequirement {
                    required: true,
                    break_glass: false,
                    required_approvals: 1,
                    expires_at: Some(now),
                },
            },
            Some(new_approval(
                "req-1",
                &ApprovalRequirement {
                    required: true,
                    break_glass: false,
                    required_approvals: 1,
                    expires_at: Some(now),
                },
                now,
            )),
            vec![TimelineEvent {
                event: "received".into(),
                actor: "bot-1".into(),
                at: now,
                summary: "request received".into(),
                data: Default::default(),
            }],
            now,
        )
        .unwrap();

    assert!(pack.signature.is_some());
    assert_eq!(pack.approval.unwrap().status, ApprovalStatus::Pending);
}
