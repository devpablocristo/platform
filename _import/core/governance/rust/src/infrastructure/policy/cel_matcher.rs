use std::collections::HashMap;
use std::sync::{Arc, Mutex};

use cel::{Context, Program, Value as CelValue};
use chrono::{DateTime, Datelike, Timelike, Utc};
use serde::Serialize;

use crate::application::ports::{PolicyEvaluationError, PolicyMatcher};
use crate::domain::model::{Policy, Request};

#[derive(Debug, Default)]
pub struct CelPolicyMatcher {
    programs: Mutex<HashMap<String, Arc<Program>>>,
}

impl CelPolicyMatcher {
    pub fn new() -> Self {
        Self::default()
    }

    fn program(&self, expression: &str) -> Result<Arc<Program>, PolicyEvaluationError> {
        let expression = expression.trim();

        if let Some(program) = self
            .programs
            .lock()
            .map_err(|_| {
                PolicyEvaluationError::EvaluationFailed("policy program cache poisoned".to_string())
            })?
            .get(expression)
            .cloned()
        {
            return Ok(program);
        }

        let program = Program::compile(expression)
            .map(Arc::new)
            .map_err(|error| PolicyEvaluationError::UnsupportedExpression(error.to_string()))?;

        let mut programs = self.programs.lock().map_err(|_| {
            PolicyEvaluationError::EvaluationFailed("policy program cache poisoned".to_string())
        })?;
        programs.insert(expression.to_string(), Arc::clone(&program));
        Ok(program)
    }
}

impl PolicyMatcher for CelPolicyMatcher {
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

        if policy.expression.trim().is_empty() {
            return Ok(true);
        }

        let program = self.program(policy.expression.as_str())?;

        let mut context = Context::default();
        context
            .add_variable("request", request_to_input(request))
            .map_err(|error| PolicyEvaluationError::EvaluationFailed(error.to_string()))?;
        context
            .add_variable(
                "time",
                TimeInput {
                    hour: now.hour(),
                    day_of_week: now.weekday().num_days_from_sunday(),
                },
            )
            .map_err(|error| PolicyEvaluationError::EvaluationFailed(error.to_string()))?;

        let result = program.execute(&context).map_err(|error| {
            PolicyEvaluationError::EvaluationFailed(format!("eval policy: {error}"))
        })?;

        match result {
            CelValue::Bool(value) => Ok(value),
            value => Err(PolicyEvaluationError::EvaluationFailed(format!(
                "policy must return bool, got {}",
                value.type_of()
            ))),
        }
    }
}

#[derive(Debug, Clone, Serialize)]
struct TimeInput {
    hour: u32,
    day_of_week: u32,
}

#[derive(Debug, Clone, Serialize)]
struct RequestInput<'a> {
    id: &'a str,
    subject: SubjectInput<'a>,
    action: &'a str,
    target: TargetInput<'a>,
    params: &'a std::collections::BTreeMap<String, serde_json::Value>,
    metadata: &'a std::collections::BTreeMap<String, serde_json::Value>,
    reason: &'a str,
    context: &'a str,
    created_at: String,
}

#[derive(Debug, Clone, Serialize)]
struct SubjectInput<'a> {
    #[serde(rename = "type")]
    subject_type: &'a crate::domain::model::RequesterType,
    id: &'a str,
    name: &'a str,
}

#[derive(Debug, Clone, Serialize)]
struct TargetInput<'a> {
    system: &'a str,
    resource: &'a str,
}

fn request_to_input(request: &Request) -> RequestInput<'_> {
    RequestInput {
        id: request.id.as_str(),
        subject: SubjectInput {
            subject_type: &request.subject.subject_type,
            id: request.subject.id.as_str(),
            name: request.subject.name.as_str(),
        },
        action: request.action.as_str(),
        target: TargetInput {
            system: request.target.system.as_str(),
            resource: request.target.resource.as_str(),
        },
        params: &request.params,
        metadata: &request.metadata,
        reason: request.reason.as_str(),
        context: request.context.as_str(),
        created_at: request.created_at.to_rfc3339(),
    }
}
