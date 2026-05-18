use std::collections::BTreeMap;

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum Format {
    Csv,
    Xlsx,
    Pdf,
    Json,
    Txt,
    Png,
    Jpg,
}

impl Format {
    pub fn as_str(self) -> &'static str {
        match self {
            Self::Csv => "csv",
            Self::Xlsx => "xlsx",
            Self::Pdf => "pdf",
            Self::Json => "json",
            Self::Txt => "txt",
            Self::Png => "png",
            Self::Jpg => "jpg",
        }
    }
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct Asset {
    pub name: String,
    pub format: Format,
    pub content_type: String,
    pub body: Vec<u8>,
    #[serde(skip_serializing_if = "BTreeMap::is_empty", default)]
    pub metadata: BTreeMap<String, String>,
}

impl Asset {
    pub fn new(
        name: &str,
        format: Format,
        body: &[u8],
        metadata: &BTreeMap<String, String>,
    ) -> Self {
        Self {
            name: normalize_filename(name, format),
            format,
            content_type: content_type(format).to_string(),
            body: body.to_vec(),
            metadata: sanitize_metadata(metadata),
        }
    }

    pub fn size(&self) -> usize {
        self.body.len()
    }
}

pub fn build_filename(parts: &[&str], format: Format, now: Option<DateTime<Utc>>) -> String {
    let mut clean = parts
        .iter()
        .filter_map(|part| {
            let value = slug(part);
            if value.is_empty() {
                return None;
            }
            Some(value)
        })
        .collect::<Vec<_>>();

    if let Some(now) = now {
        clean.push(now.format("%Y-%m-%d").to_string());
    }
    if clean.is_empty() {
        clean.push("artifact".to_string());
    }

    normalize_filename(clean.join("_").as_str(), format)
}

pub fn normalize_filename(name: &str, format: Format) -> String {
    let trimmed = name.trim();
    let base = if trimmed.is_empty() {
        "artifact"
    } else {
        trimmed
    };
    let ext = extension(format);
    if ext.is_empty() {
        return base.to_string();
    }

    let suffix = format!(".{ext}");
    if base.to_lowercase().ends_with(suffix.as_str()) {
        return base.to_string();
    }
    format!("{base}{suffix}")
}

pub fn extension(format: Format) -> &'static str {
    format.as_str()
}

pub fn content_type(format: Format) -> &'static str {
    match format {
        Format::Csv => "text/csv; charset=utf-8",
        Format::Xlsx => "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
        Format::Pdf => "application/pdf",
        Format::Json => "application/json",
        Format::Txt => "text/plain; charset=utf-8",
        Format::Png => "image/png",
        Format::Jpg => "image/jpeg",
    }
}

pub fn slug(value: &str) -> String {
    let mut out = String::new();
    let mut last_sep = false;

    for ch in value.trim().to_lowercase().chars() {
        if ch.is_alphanumeric() {
            out.push(ch);
            last_sep = false;
            continue;
        }

        if (ch == '-' || ch == '_' || ch.is_whitespace() || ch == '/' || ch == '.')
            && !last_sep
            && !out.is_empty()
        {
            out.push('_');
            last_sep = true;
        }
    }

    out.trim_matches('_').to_string()
}

fn sanitize_metadata(input: &BTreeMap<String, String>) -> BTreeMap<String, String> {
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
