use chrono::{DateTime, Datelike, Timelike, Utc};

use crate::application::ports::{PolicyEvaluationError, PolicyMatcher};
use crate::domain::model::{Policy, Request};

#[derive(Debug, Default, Clone, Copy)]
pub struct DeterministicPolicyMatcher;

impl PolicyMatcher for DeterministicPolicyMatcher {
    fn matches(
        &self,
        request: &Request,
        policy: &Policy,
        now: DateTime<Utc>,
    ) -> Result<bool, PolicyEvaluationError> {
        if !policy.action_filter.trim().is_empty()
            && policy.action_filter.trim() != request.action.trim()
        {
            return Ok(false);
        }

        if !policy.system_filter.trim().is_empty()
            && policy.system_filter.trim() != request.target.system.trim()
        {
            return Ok(false);
        }

        evaluate_expression(policy.expression.as_str(), request, now)
    }
}

fn evaluate_expression(
    expression: &str,
    request: &Request,
    now: DateTime<Utc>,
) -> Result<bool, PolicyEvaluationError> {
    let expression = expression.trim();
    if expression.is_empty() || expression == "true" {
        return Ok(true);
    }
    if expression == "false" {
        return Ok(false);
    }

    let (left, right) = expression
        .split_once("==")
        .ok_or_else(|| PolicyEvaluationError::UnsupportedExpression(expression.to_string()))?;
    let left = left.trim();
    let right = unquote(right.trim())?;

    let actual = match left {
        "request.id" => request.id.as_str(),
        "request.action" => request.action.as_str(),
        "request.target.system" => request.target.system.as_str(),
        "request.target.resource" => request.target.resource.as_str(),
        "request.subject.id" => request.subject.id.as_str(),
        "request.subject.name" => request.subject.name.as_str(),
        "request.subject.type" => match request.subject.subject_type {
            crate::domain::model::RequesterType::Agent => "agent",
            crate::domain::model::RequesterType::Service => "service",
            crate::domain::model::RequesterType::Human => "human",
        },
        "request.reason" => request.reason.as_str(),
        "request.context" => request.context.as_str(),
        "time.hour" => {
            return Ok(now.hour().to_string() == right);
        }
        "time.day_of_week" => {
            return Ok(now.weekday().num_days_from_sunday().to_string() == right);
        }
        _ => {
            return Err(PolicyEvaluationError::UnsupportedExpression(
                expression.to_string(),
            ))
        }
    };

    Ok(actual == right)
}

fn unquote(value: &str) -> Result<String, PolicyEvaluationError> {
    if value.len() >= 2
        && ((value.starts_with('"') && value.ends_with('"'))
            || (value.starts_with('\'') && value.ends_with('\'')))
    {
        return Ok(value[1..value.len() - 1].to_string());
    }
    Err(PolicyEvaluationError::UnsupportedExpression(
        value.to_string(),
    ))
}
