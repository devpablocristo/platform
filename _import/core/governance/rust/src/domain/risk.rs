use std::cmp::Ordering;

use chrono::{DateTime, Timelike, Utc};

use super::model::{Decision, Request, RiskAssessment, RiskFactor, RiskLevel};

#[derive(Debug, Clone, PartialEq)]
pub struct RiskConfig {
    pub thresholds: Thresholds,
    pub high_actions: Vec<String>,
    pub medium_actions: Vec<String>,
    pub business_hours: BusinessHours,
    pub frequency_thresholds: FrequencyThresholds,
    pub actor_thresholds: ActorThresholds,
    pub success_rate_thresholds: SuccessRateThresholds,
    pub amplifications: Vec<AmplificationRule>,
    pub sensitive_systems: Vec<String>,
}

#[derive(Debug, Clone, PartialEq)]
pub struct Thresholds {
    pub allow: f64,
    pub enhanced_log: f64,
    pub require_approval: f64,
    pub deny: f64,
    pub max_amplification: f64,
}

#[derive(Debug, Clone, PartialEq)]
pub struct BusinessHours {
    pub start: u32,
    pub end: u32,
}

#[derive(Debug, Clone, PartialEq)]
pub struct FrequencyThresholds {
    pub warning: u32,
    pub critical: u32,
}

#[derive(Debug, Clone, PartialEq)]
pub struct ActorThresholds {
    pub unknown: u32,
    pub new: u32,
}

#[derive(Debug, Clone, PartialEq)]
pub struct SuccessRateThresholds {
    pub low: f64,
    pub moderate: f64,
    pub excellent: f64,
}

#[derive(Debug, Clone, PartialEq)]
pub struct AmplificationRule {
    pub factors: Vec<String>,
    pub multiplier: f64,
    pub reason: String,
}

#[derive(Debug, Clone, PartialEq)]
pub struct RiskHistory {
    pub actor_history: u32,
    pub recent_frequency: u32,
    pub success_rate: Option<f64>,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum DecisionRecommendation {
    FromPolicy(Decision),
    Default(Decision),
}

impl Default for RiskConfig {
    fn default() -> Self {
        Self {
            thresholds: Thresholds {
                allow: 0.5,
                enhanced_log: 1.0,
                require_approval: 1.5,
                deny: 2.0,
                max_amplification: 3.0,
            },
            high_actions: vec![
                "alert.silence".into(),
                "runbook.execute".into(),
                "delete".into(),
            ],
            medium_actions: vec![
                "incident.resolve".into(),
                "config.update".into(),
                "deploy.trigger".into(),
            ],
            business_hours: BusinessHours { start: 9, end: 18 },
            frequency_thresholds: FrequencyThresholds {
                warning: 10,
                critical: 20,
            },
            actor_thresholds: ActorThresholds {
                unknown: 0,
                new: 10,
            },
            success_rate_thresholds: SuccessRateThresholds {
                low: 0.5,
                moderate: 0.8,
                excellent: 0.95,
            },
            amplifications: vec![
                AmplificationRule {
                    factors: vec!["off_hours".into(), "actor_unknown".into()],
                    multiplier: 1.8,
                    reason: "off-hours + unknown actor".into(),
                },
                AmplificationRule {
                    factors: vec!["action_type".into(), "frequency_anomaly".into()],
                    multiplier: 1.5,
                    reason: "risky action + frequency anomaly".into(),
                },
                AmplificationRule {
                    factors: vec!["actor_unknown".into(), "target_sensitivity".into()],
                    multiplier: 1.6,
                    reason: "unknown actor + sensitive target".into(),
                },
                AmplificationRule {
                    factors: vec![
                        "off_hours".into(),
                        "actor_unknown".into(),
                        "frequency_anomaly".into(),
                    ],
                    multiplier: 2.5,
                    reason: "full cascade: off-hours + unknown + frequency".into(),
                },
                AmplificationRule {
                    factors: vec![
                        "action_type".into(),
                        "off_hours".into(),
                        "target_sensitivity".into(),
                    ],
                    multiplier: 2.0,
                    reason: "risky action + off-hours + sensitive target".into(),
                },
            ],
            sensitive_systems: vec!["production".into(), "prod".into()],
        }
    }
}

pub fn evaluate_risk(
    request: &Request,
    history: &RiskHistory,
    config: &RiskConfig,
    policy_risk_override: Option<&RiskLevel>,
    now: DateTime<Utc>,
) -> RiskAssessment {
    let factors = evaluate_factors(request, history, config, now);
    let raw_score = sum_factors(&factors);
    let amplification = calculate_amplification(&factors, config);
    let mut final_score = raw_score * amplification;

    if let Some(override_level) = policy_risk_override {
        final_score = apply_policy_override(override_level, final_score, config);
    }

    RiskAssessment {
        factors,
        raw_score,
        amplification,
        final_score,
        level: score_to_level(final_score, config),
        recommended: score_to_decision(final_score, config),
    }
}

pub fn tier_for_action(
    action: &str,
    policy_risk_override: Option<&RiskLevel>,
    config: &RiskConfig,
) -> RiskLevel {
    if let Some(level) = policy_risk_override {
        return level.clone();
    }

    let action = action.trim();
    if contains_trimmed(&config.high_actions, action) {
        return RiskLevel::High;
    }
    if contains_trimmed(&config.medium_actions, action) {
        return RiskLevel::Medium;
    }
    RiskLevel::Low
}

pub fn decide_from_policy(effect: &Decision, tier: &RiskLevel) -> DecisionRecommendation {
    match effect {
        Decision::Deny => DecisionRecommendation::FromPolicy(Decision::Deny),
        Decision::RequireApproval => DecisionRecommendation::FromPolicy(Decision::RequireApproval),
        Decision::Allow if matches!(tier, RiskLevel::High) => {
            DecisionRecommendation::FromPolicy(Decision::RequireApproval)
        }
        Decision::Allow => DecisionRecommendation::FromPolicy(Decision::Allow),
    }
}

pub fn default_decision(tier: &RiskLevel) -> DecisionRecommendation {
    match tier {
        RiskLevel::High => DecisionRecommendation::Default(Decision::RequireApproval),
        RiskLevel::Low | RiskLevel::Medium => DecisionRecommendation::Default(Decision::Allow),
    }
}

fn evaluate_factors(
    request: &Request,
    history: &RiskHistory,
    config: &RiskConfig,
    now: DateTime<Utc>,
) -> Vec<RiskFactor> {
    vec![
        action_factor(request.action.as_str(), config),
        off_hours_factor(now, config),
        actor_history_factor(history.actor_history, config),
        frequency_factor(history.recent_frequency, config),
        success_rate_factor(history.success_rate, config),
        target_factor(request.target.system.as_str(), config),
    ]
}

fn action_factor(action: &str, config: &RiskConfig) -> RiskFactor {
    let action = action.trim();
    if contains_trimmed(&config.high_actions, action) {
        return RiskFactor {
            name: "action_type".into(),
            score: 0.4,
            active: true,
            reason: format!("{action} is high-risk action"),
        };
    }
    if contains_trimmed(&config.medium_actions, action) {
        return RiskFactor {
            name: "action_type".into(),
            score: 0.2,
            active: true,
            reason: format!("{action} is medium-risk action"),
        };
    }
    RiskFactor {
        name: "action_type".into(),
        score: 0.1,
        active: false,
        reason: format!("{action} is low-risk action"),
    }
}

fn off_hours_factor(now: DateTime<Utc>, config: &RiskConfig) -> RiskFactor {
    let hour = now.hour();
    if hour < config.business_hours.start || hour >= config.business_hours.end {
        return RiskFactor {
            name: "off_hours".into(),
            score: 0.2,
            active: true,
            reason: "request at off-hours".into(),
        };
    }
    RiskFactor {
        name: "off_hours".into(),
        score: 0.0,
        active: false,
        reason: String::new(),
    }
}

fn actor_history_factor(actor_history: u32, config: &RiskConfig) -> RiskFactor {
    match actor_history.cmp(&config.actor_thresholds.unknown) {
        Ordering::Less | Ordering::Equal => RiskFactor {
            name: "actor_unknown".into(),
            score: 0.3,
            active: true,
            reason: "unknown actor, no previous requests".into(),
        },
        Ordering::Greater if actor_history < config.actor_thresholds.new => RiskFactor {
            name: "actor_unknown".into(),
            score: 0.15,
            active: true,
            reason: "new actor with limited history".into(),
        },
        Ordering::Greater => RiskFactor {
            name: "actor_unknown".into(),
            score: 0.0,
            active: false,
            reason: String::new(),
        },
    }
}

fn frequency_factor(recent_frequency: u32, config: &RiskConfig) -> RiskFactor {
    if recent_frequency > config.frequency_thresholds.critical {
        return RiskFactor {
            name: "frequency_anomaly".into(),
            score: 0.3,
            active: true,
            reason: "frequency above critical threshold".into(),
        };
    }
    if recent_frequency > config.frequency_thresholds.warning {
        return RiskFactor {
            name: "frequency_anomaly".into(),
            score: 0.15,
            active: true,
            reason: "frequency above warning threshold".into(),
        };
    }
    RiskFactor {
        name: "frequency_anomaly".into(),
        score: 0.0,
        active: false,
        reason: String::new(),
    }
}

fn success_rate_factor(success_rate: Option<f64>, config: &RiskConfig) -> RiskFactor {
    match success_rate {
        None => RiskFactor {
            name: "execution_history".into(),
            score: 0.0,
            active: false,
            reason: String::new(),
        },
        Some(value) if value < config.success_rate_thresholds.low => RiskFactor {
            name: "execution_history".into(),
            score: 0.3,
            active: true,
            reason: "low historical success rate".into(),
        },
        Some(value) if value < config.success_rate_thresholds.moderate => RiskFactor {
            name: "execution_history".into(),
            score: 0.1,
            active: true,
            reason: "moderate historical success rate".into(),
        },
        Some(value) if value >= config.success_rate_thresholds.excellent => RiskFactor {
            name: "execution_history".into(),
            score: -0.15,
            active: false,
            reason: "excellent historical success rate".into(),
        },
        Some(_) => RiskFactor {
            name: "execution_history".into(),
            score: 0.0,
            active: false,
            reason: String::new(),
        },
    }
}

fn target_factor(system: &str, config: &RiskConfig) -> RiskFactor {
    let normalized = system.trim().to_lowercase();
    if contains_trimmed(&config.sensitive_systems, normalized.as_str()) {
        return RiskFactor {
            name: "target_sensitivity".into(),
            score: 0.3,
            active: true,
            reason: "target is sensitive system".into(),
        };
    }
    RiskFactor {
        name: "target_sensitivity".into(),
        score: 0.0,
        active: false,
        reason: String::new(),
    }
}

fn calculate_amplification(factors: &[RiskFactor], config: &RiskConfig) -> f64 {
    let mut active = Vec::new();
    for factor in factors {
        if factor.active {
            active.push(factor.name.as_str());
        }
    }

    let mut amplification = 1.0_f64;
    for rule in &config.amplifications {
        if rule
            .factors
            .iter()
            .all(|candidate| active.iter().any(|name| name == candidate))
        {
            amplification = amplification.max(rule.multiplier);
        }
    }
    if active.len() >= 4 {
        amplification = amplification.max(2.5);
    }

    amplification.min(config.thresholds.max_amplification)
}

fn apply_policy_override(
    override_level: &RiskLevel,
    current_score: f64,
    config: &RiskConfig,
) -> f64 {
    match override_level {
        RiskLevel::High if current_score < config.thresholds.require_approval => {
            config.thresholds.require_approval
        }
        RiskLevel::Medium if current_score < config.thresholds.enhanced_log => {
            config.thresholds.enhanced_log
        }
        RiskLevel::Low if current_score > config.thresholds.allow => config.thresholds.allow * 0.9,
        _ => current_score,
    }
}

fn score_to_level(score: f64, config: &RiskConfig) -> RiskLevel {
    if score >= config.thresholds.require_approval {
        return RiskLevel::High;
    }
    if score >= config.thresholds.enhanced_log {
        return RiskLevel::Medium;
    }
    RiskLevel::Low
}

fn score_to_decision(score: f64, config: &RiskConfig) -> Decision {
    if score >= config.thresholds.deny {
        return Decision::Deny;
    }
    if score >= config.thresholds.require_approval {
        return Decision::RequireApproval;
    }
    Decision::Allow
}

fn sum_factors(factors: &[RiskFactor]) -> f64 {
    let total: f64 = factors.iter().map(|factor| factor.score).sum();
    total.max(0.0)
}

fn contains_trimmed(values: &[String], expected: &str) -> bool {
    values
        .iter()
        .any(|candidate| candidate.trim() == expected.trim())
}
