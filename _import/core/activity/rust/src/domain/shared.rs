use std::collections::BTreeMap;

use chrono::{DateTime, TimeZone, Utc};
use serde::{Deserialize, Serialize};
use serde_json::Value;

pub fn unix_epoch() -> DateTime<Utc> {
    Utc.timestamp_opt(0, 0)
        .single()
        .expect("unix epoch must be representable")
}

#[derive(Debug, Clone, PartialEq, Eq, Default, Serialize, Deserialize)]
pub struct ActorRef {
    #[serde(skip_serializing_if = "String::is_empty", default)]
    pub legacy: String,
    #[serde(skip_serializing_if = "String::is_empty", default)]
    pub actor_type: String,
    #[serde(skip_serializing_if = "String::is_empty", default)]
    pub id: String,
    #[serde(skip_serializing_if = "String::is_empty", default)]
    pub label: String,
}

pub fn first_non_empty(values: &[&str]) -> String {
    for value in values {
        let trimmed = value.trim();
        if !trimmed.is_empty() {
            return trimmed.to_string();
        }
    }
    String::new()
}

pub fn trim_json_map(input: &BTreeMap<String, Value>) -> BTreeMap<String, Value> {
    input
        .iter()
        .filter_map(|(key, value)| {
            let key = key.trim();
            if key.is_empty() {
                return None;
            }
            Some((key.to_string(), value.clone()))
        })
        .collect()
}
