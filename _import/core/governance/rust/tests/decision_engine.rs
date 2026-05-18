use chrono::{TimeZone, Utc};
use serde_json::json;

use governance::{
    ApprovalConfig, CelPolicyMatcher, Decision, DecisionEngine, DecisionInput,
    DeterministicPolicyMatcher, Policy, PolicyMode, Request, RequesterType, RiskConfig,
    RiskHistory, Subject, Target,
};

#[test]
fn evaluate_uses_first_matching_enforced_policy() {
    let engine = DecisionEngine::new(
        DeterministicPolicyMatcher,
        RiskConfig::default(),
        ApprovalConfig::default(),
    );
    let now = Utc
        .with_ymd_and_hms(2026, 3, 20, 10, 0, 0)
        .single()
        .unwrap();

    let evaluation = engine
        .evaluate(DecisionInput {
            request: Request {
                id: "req-1".into(),
                action: "delete".into(),
                target: Target {
                    system: "prod".into(),
                    resource: String::new(),
                },
                subject: Subject {
                    subject_type: RequesterType::Agent,
                    id: "bot-1".into(),
                    name: String::new(),
                },
                created_at: now,
                ..Request::default()
            },
            policies: vec![
                Policy {
                    id: "shadow-1".into(),
                    name: "shadow".into(),
                    expression: "true".into(),
                    effect: Decision::Deny,
                    risk_override: None,
                    priority: 1,
                    mode: PolicyMode::Shadow,
                    enabled: true,
                    action_filter: String::new(),
                    system_filter: String::new(),
                },
                Policy {
                    id: "enforce-1".into(),
                    name: "allow-delete".into(),
                    expression: "request.action == \"delete\"".into(),
                    effect: Decision::Allow,
                    risk_override: None,
                    priority: 2,
                    mode: PolicyMode::Enforce,
                    enabled: true,
                    action_filter: String::new(),
                    system_filter: String::new(),
                },
                Policy {
                    id: "enforce-2".into(),
                    name: "deny-all".into(),
                    expression: "true".into(),
                    effect: Decision::Deny,
                    risk_override: None,
                    priority: 3,
                    mode: PolicyMode::Enforce,
                    enabled: true,
                    action_filter: String::new(),
                    system_filter: String::new(),
                },
            ],
            history: RiskHistory {
                actor_history: 0,
                recent_frequency: 0,
                success_rate: None,
            },
            now,
        })
        .unwrap();

    assert_eq!(evaluation.policy_id, "enforce-1");
    assert_eq!(evaluation.decision, Decision::RequireApproval);
    assert_eq!(evaluation.shadow_policies, vec!["shadow-1".to_string()]);
    assert!(evaluation.approval.required);
}

#[test]
fn evaluate_falls_back_to_default_decision() {
    let engine = DecisionEngine::new(
        DeterministicPolicyMatcher,
        RiskConfig::default(),
        ApprovalConfig::default(),
    );
    let now = Utc
        .with_ymd_and_hms(2026, 3, 20, 11, 0, 0)
        .single()
        .unwrap();

    let evaluation = engine
        .evaluate(DecisionInput {
            request: Request {
                id: "req-2".into(),
                action: "read".into(),
                target: Target {
                    system: "staging".into(),
                    resource: String::new(),
                },
                subject: Subject {
                    subject_type: RequesterType::Human,
                    id: "user-1".into(),
                    name: String::new(),
                },
                created_at: now,
                ..Request::default()
            },
            policies: Vec::new(),
            history: RiskHistory {
                actor_history: 20,
                recent_frequency: 0,
                success_rate: Some(0.99),
            },
            now,
        })
        .unwrap();

    assert!(evaluation.policy_id.is_empty());
    assert_eq!(evaluation.decision, Decision::Allow);
}

#[test]
fn evaluate_supports_cel_policy_matching() {
    let engine = DecisionEngine::new(
        CelPolicyMatcher::new(),
        RiskConfig::default(),
        ApprovalConfig::default(),
    );
    let now = Utc
        .with_ymd_and_hms(2026, 3, 20, 22, 0, 0)
        .single()
        .unwrap();

    let evaluation = engine
        .evaluate(DecisionInput {
            request: Request {
                id: "req-3".into(),
                action: "delete".into(),
                target: Target {
                    system: "prod".into(),
                    resource: String::new(),
                },
                subject: Subject {
                    subject_type: RequesterType::Agent,
                    id: "bot-1".into(),
                    name: String::new(),
                },
                params: std::collections::BTreeMap::from([("amount".into(), json!(1200))]),
                created_at: now,
                ..Request::default()
            },
            policies: vec![Policy {
                id: "enforce-cel".into(),
                name: "delete-high-amount".into(),
                expression:
                    "request.action == \"delete\" && request.target.system == \"prod\" && request.params.amount > 1000".into(),
                effect: Decision::Deny,
                risk_override: None,
                priority: 1,
                mode: PolicyMode::Enforce,
                enabled: true,
                action_filter: String::new(),
                system_filter: String::new(),
            }],
            history: RiskHistory {
                actor_history: 0,
                recent_frequency: 0,
                success_rate: None,
            },
            now,
        })
        .unwrap();

    assert_eq!(evaluation.policy_id, "enforce-cel");
    assert_eq!(evaluation.decision, Decision::Deny);
}
