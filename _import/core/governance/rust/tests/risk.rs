use chrono::{TimeZone, Utc};

use governance::{
    evaluate_risk, tier_for_action, Decision, Request, RiskConfig, RiskHistory, RiskLevel, Target,
};

#[test]
fn tier_uses_action_defaults() {
    let config = RiskConfig::default();

    assert_eq!(tier_for_action("delete", None, &config), RiskLevel::High);
    assert_eq!(
        tier_for_action("deploy.trigger", None, &config),
        RiskLevel::Medium
    );
    assert_eq!(tier_for_action("read", None, &config), RiskLevel::Low);
}

#[test]
fn evaluate_high_risk_cascade() {
    let config = RiskConfig::default();
    let request = Request {
        action: "delete".into(),
        target: Target {
            system: "prod".into(),
            resource: String::new(),
        },
        ..Request::default()
    };

    let assessment = evaluate_risk(
        &request,
        &RiskHistory {
            actor_history: 0,
            recent_frequency: 25,
            success_rate: Some(0.4),
        },
        &config,
        None,
        Utc.with_ymd_and_hms(2026, 3, 20, 22, 0, 0)
            .single()
            .unwrap(),
    );

    assert_eq!(assessment.level, RiskLevel::High);
    assert_eq!(assessment.recommended, Decision::Deny);
    assert!(assessment.amplification > 1.0);
}
