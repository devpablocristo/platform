use std::collections::BTreeMap;

use chrono::{TimeZone, Utc};
use serde_json::json;

use governance::{
    CelPolicyMatcher, Policy, PolicyMatcher, Request, RequesterType, Subject, Target,
};

#[test]
fn matches_nested_fields_and_numeric_comparison() {
    let matcher = CelPolicyMatcher::new();
    let now = Utc
        .with_ymd_and_hms(2026, 3, 20, 22, 0, 0)
        .single()
        .unwrap();

    let matches = matcher
        .matches(
            &Request {
                subject: Subject {
                    subject_type: RequesterType::Agent,
                    id: "bot-1".into(),
                    name: String::new(),
                },
                action: "delete".into(),
                target: Target {
                    system: "prod".into(),
                    resource: String::new(),
                },
                params: BTreeMap::from([("amount".into(), json!(1200))]),
                created_at: now,
                ..Request::default()
            },
            &Policy {
                id: "policy-1".into(),
                name: "delete high amount".into(),
                expression: "request.action == \"delete\" && request.target.system == \"prod\" && request.params.amount > 1000".into(),
                effect: governance::Decision::Allow,
                risk_override: None,
                priority: 1,
                mode: governance::PolicyMode::Enforce,
                enabled: true,
                action_filter: String::new(),
                system_filter: String::new(),
            },
            now,
        )
        .unwrap();

    assert!(matches);
}

#[test]
fn rejects_invalid_expression() {
    let matcher = CelPolicyMatcher::new();
    let error = matcher
        .matches(
            &Request::default(),
            &Policy {
                id: "policy-2".into(),
                name: "invalid".into(),
                expression: "request.action = \"delete\"".into(),
                effect: governance::Decision::Allow,
                risk_override: None,
                priority: 1,
                mode: governance::PolicyMode::Enforce,
                enabled: true,
                action_filter: String::new(),
                system_filter: String::new(),
            },
            Utc::now(),
        )
        .unwrap_err();

    assert!(
        matches!(
            error,
            governance::PolicyEvaluationError::UnsupportedExpression(_)
        ),
        "unexpected error: {error}"
    );
}

#[test]
fn applies_static_filters_before_expression() {
    let matcher = CelPolicyMatcher::new();
    let matched = matcher
        .matches(
            &Request {
                action: "deploy".into(),
                target: Target {
                    system: "prod".into(),
                    resource: String::new(),
                },
                ..Request::default()
            },
            &Policy {
                id: "policy-3".into(),
                name: "delete only".into(),
                expression: "true".into(),
                effect: governance::Decision::Allow,
                risk_override: None,
                priority: 1,
                mode: governance::PolicyMode::Enforce,
                enabled: true,
                action_filter: "delete".into(),
                system_filter: "prod".into(),
            },
            Utc::now(),
        )
        .unwrap();

    assert!(!matched);
}
