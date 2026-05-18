use chrono::{DateTime, Utc};
use thiserror::Error;

use crate::domain::approval::{requirement_for, ApprovalConfig};
use crate::domain::model::{Decision, Evaluation, Policy, Request};
use crate::domain::risk::{
    decide_from_policy, default_decision, evaluate_risk, tier_for_action, RiskConfig, RiskHistory,
};

use super::ports::{PolicyEvaluationError, PolicyMatcher};

#[derive(Debug, Clone, PartialEq)]
pub struct DecisionInput {
    pub request: Request,
    pub policies: Vec<Policy>,
    pub history: RiskHistory,
    pub now: DateTime<Utc>,
}

#[derive(Debug, Error)]
pub enum DecisionEngineError {
    #[error("policy evaluation failed for {policy_name}: {source}")]
    PolicyEvaluation {
        policy_name: String,
        #[source]
        source: PolicyEvaluationError,
    },
}

pub struct DecisionEngine<M> {
    matcher: M,
    risk_config: RiskConfig,
    approval_config: ApprovalConfig,
}

impl<M> DecisionEngine<M>
where
    M: PolicyMatcher,
{
    pub fn new(matcher: M, risk_config: RiskConfig, approval_config: ApprovalConfig) -> Self {
        Self {
            matcher,
            risk_config,
            approval_config,
        }
    }

    pub fn evaluate(&self, input: DecisionInput) -> Result<Evaluation, DecisionEngineError> {
        let mut policies = input.policies;
        policies.sort_by(|left, right| {
            left.priority
                .cmp(&right.priority)
                .then_with(|| left.id.cmp(&right.id))
        });

        let mut evaluation = Evaluation {
            request_id: input.request.id.clone(),
            evaluated_at: input.now,
            decision: Decision::Allow,
            decision_reason: String::new(),
            policy_id: String::new(),
            policy_name: String::new(),
            shadow_policies: Vec::new(),
            risk_tier: tier_for_action(input.request.action.as_str(), None, &self.risk_config),
            risk: evaluate_risk(
                &input.request,
                &input.history,
                &self.risk_config,
                None,
                input.now,
            ),
            approval: Default::default(),
        };

        let mut selected: Option<Policy> = None;
        for policy in policies {
            if !policy.enabled {
                continue;
            }

            let matches = self
                .matcher
                .matches(&input.request, &policy, input.now)
                .map_err(|source| DecisionEngineError::PolicyEvaluation {
                    policy_name: policy.name.clone(),
                    source,
                })?;

            if !matches {
                continue;
            }

            if matches!(policy.mode, crate::domain::model::PolicyMode::Shadow) {
                evaluation.shadow_policies.push(policy.id.clone());
                continue;
            }

            selected = Some(policy);
            break;
        }

        let risk_override = selected
            .as_ref()
            .and_then(|policy| policy.risk_override.as_ref());
        evaluation.risk = evaluate_risk(
            &input.request,
            &input.history,
            &self.risk_config,
            risk_override,
            input.now,
        );
        evaluation.risk_tier = tier_for_action(
            input.request.action.as_str(),
            risk_override,
            &self.risk_config,
        );

        match selected {
            Some(policy) => {
                let decision = match decide_from_policy(&policy.effect, &evaluation.risk_tier) {
                    crate::domain::risk::DecisionRecommendation::FromPolicy(decision)
                    | crate::domain::risk::DecisionRecommendation::Default(decision) => decision,
                };
                evaluation.decision = decision;
                evaluation.decision_reason = format!("Policy '{}'", policy.name);
                evaluation.policy_id = policy.id;
                evaluation.policy_name = policy.name;
            }
            None => {
                let decision = match default_decision(&evaluation.risk_tier) {
                    crate::domain::risk::DecisionRecommendation::FromPolicy(decision)
                    | crate::domain::risk::DecisionRecommendation::Default(decision) => decision,
                };
                evaluation.decision = decision;
                let risk_label = match evaluation.risk_tier {
                    crate::domain::model::RiskLevel::Low => "low",
                    crate::domain::model::RiskLevel::Medium => "medium",
                    crate::domain::model::RiskLevel::High => "high",
                };
                evaluation.decision_reason =
                    format!("No policy matched; default for risk {risk_label}");
            }
        }

        evaluation.approval = requirement_for(
            &input.request,
            &evaluation.decision,
            &evaluation.risk_tier,
            &self.approval_config,
            input.now,
        );

        Ok(evaluation)
    }
}
