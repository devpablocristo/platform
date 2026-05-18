//! Validadores puros sin dependencias de framework.
//! Cada función retorna `Result<(), ValidationError>`.

use std::fmt;

/// Error de validación con campo y mensaje.
#[derive(Debug, Clone)]
pub struct ValidationError {
    pub field: String,
    pub message: String,
}

impl fmt::Display for ValidationError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{}: {}", self.field, self.message)
    }
}

impl std::error::Error for ValidationError {}

/// Crea un ValidationError.
pub fn err(field: impl Into<String>, message: impl Into<String>) -> ValidationError {
    ValidationError { field: field.into(), message: message.into() }
}

/// Combina múltiples errores en uno solo ("; " separados).
pub fn join_errors(errors: &[ValidationError]) -> Option<ValidationError> {
    let non_empty: Vec<_> = errors.iter().filter(|e| !e.message.is_empty()).collect();
    if non_empty.is_empty() {
        return None;
    }
    let combined: Vec<String> = non_empty.iter().map(|e| format!("{}: {}", e.field, e.message)).collect();
    Some(ValidationError {
        field: "validation".into(),
        message: combined.join("; "),
    })
}

// --- Strings ---

pub fn required_string(field: &str, value: &str) -> Result<(), ValidationError> {
    if value.trim().is_empty() {
        return Err(err(field, "cannot be empty"));
    }
    Ok(())
}

pub fn string_len(field: &str, value: &str, min: usize, max: usize) -> Result<(), ValidationError> {
    let len = value.trim().len();
    if len < min {
        return Err(err(field, format!("must be at least {min} characters long")));
    }
    if len > max {
        return Err(err(field, format!("must be at most {max} characters long")));
    }
    Ok(())
}

pub fn email(field: &str, value: &str) -> Result<(), ValidationError> {
    if value.is_empty() {
        return Err(err(field, "cannot be empty"));
    }
    if value.contains(' ') {
        return Err(err(field, "cannot contain spaces"));
    }
    if value.matches('@').count() != 1 {
        return Err(err(field, "must contain exactly one @ symbol"));
    }
    let parts: Vec<&str> = value.split('@').collect();
    if parts.len() != 2 || parts[0].is_empty() || parts[1].is_empty() || !parts[1].contains('.') {
        return Err(err(field, "invalid email format"));
    }
    Ok(())
}

pub fn url(field: &str, value: &str) -> Result<(), ValidationError> {
    if value.is_empty() {
        return Err(err(field, "cannot be empty"));
    }
    if !value.starts_with("http://") && !value.starts_with("https://") {
        return Err(err(field, "scheme must be http or https"));
    }
    Ok(())
}

pub fn numeric(field: &str, value: &str) -> Result<(), ValidationError> {
    if value.trim().is_empty() {
        return Err(err(field, "cannot be empty"));
    }
    if !value.trim().chars().all(|c| c.is_ascii_digit()) {
        return Err(err(field, "must contain only digits"));
    }
    Ok(())
}

// --- Numbers ---

pub fn int_range(field: &str, n: i64, min: i64, max: i64) -> Result<(), ValidationError> {
    if n < min || n > max {
        return Err(err(field, format!("must be between {min} and {max}")));
    }
    Ok(())
}

pub fn float_range(field: &str, n: f64, min: f64, max: f64) -> Result<(), ValidationError> {
    if n < min || n > max {
        return Err(err(field, format!("must be between {min} and {max}")));
    }
    Ok(())
}

pub fn non_negative(field: &str, n: i64) -> Result<(), ValidationError> {
    if n < 0 {
        return Err(err(field, "must be non-negative"));
    }
    Ok(())
}

// --- IDs ---

pub fn uuid_v4(field: &str, value: &str) -> Result<(), ValidationError> {
    if value.is_empty() {
        return Err(err(field, "cannot be empty"));
    }
    match uuid::Uuid::parse_str(value) {
        Ok(id) if id.get_version_num() == 4 => Ok(()),
        _ => Err(err(field, "invalid UUID v4 format")),
    }
}

pub fn uuid_any(field: &str, value: &str) -> Result<(), ValidationError> {
    if value.is_empty() {
        return Err(err(field, "cannot be empty"));
    }
    uuid::Uuid::parse_str(value)
        .map(|_| ())
        .map_err(|_| err(field, "invalid UUID format"))
}

// --- Time ---

pub fn iso_date(field: &str, value: &str) -> Result<chrono::NaiveDate, ValidationError> {
    if value.is_empty() {
        return Err(err(field, "cannot be empty"));
    }
    chrono::NaiveDate::parse_from_str(value, "%Y-%m-%d")
        .map_err(|_| err(field, "invalid ISO date format (YYYY-MM-DD)"))
}

pub fn iso_timestamp(field: &str, value: &str) -> Result<chrono::DateTime<chrono::Utc>, ValidationError> {
    if value.is_empty() {
        return Err(err(field, "cannot be empty"));
    }
    value
        .parse::<chrono::DateTime<chrono::Utc>>()
        .map_err(|_| err(field, "invalid ISO timestamp format"))
}

pub fn not_future(field: &str, value: &str) -> Result<(), ValidationError> {
    let t = iso_timestamp(field, value)?;
    if t > chrono::Utc::now() {
        return Err(err(field, "cannot be in the future"));
    }
    Ok(())
}

// --- Enums ---

pub fn enum_string(field: &str, value: &str, allowed: &[&str]) -> Result<(), ValidationError> {
    if allowed.is_empty() {
        return Err(err(field, "no allowed values specified"));
    }
    if allowed.contains(&value) {
        return Ok(());
    }
    Err(err(field, "value not in allowed list"))
}

// --- Money ---

pub fn currency_iso4217(field: &str, code: &str) -> Result<(), ValidationError> {
    if code.is_empty() {
        return Err(err(field, "cannot be empty"));
    }
    if code.len() != 3 {
        return Err(err(field, "must be exactly 3 characters long"));
    }
    if !code.chars().all(|c| c.is_ascii_uppercase()) {
        return Err(err(field, "must be uppercase letters only"));
    }
    Ok(())
}

pub fn monetary_cents(field: &str, cents: i64) -> Result<(), ValidationError> {
    non_negative(field, cents)
}

// --- Collections ---

pub fn slice_not_empty<T>(field: &str, items: &[T]) -> Result<(), ValidationError> {
    if items.is_empty() {
        return Err(err(field, "cannot be empty"));
    }
    Ok(())
}

pub fn slice_len(field: &str, len: usize, min: usize, max: usize) -> Result<(), ValidationError> {
    if len < min {
        return Err(err(field, format!("must have at least {min} items")));
    }
    if len > max {
        return Err(err(field, format!("must have at most {max} items")));
    }
    Ok(())
}

pub fn unique_strings(field: &str, items: &[&str], case_insensitive: bool) -> Result<(), ValidationError> {
    let mut seen = std::collections::HashSet::new();
    for &item in items {
        let key = if case_insensitive { item.to_lowercase() } else { item.to_owned() };
        if !seen.insert(key) {
            return Err(err(field, "contains duplicate values"));
        }
    }
    Ok(())
}

// --- Pagination ---

pub fn pagination(page: usize, limit: usize, max_limit: usize) -> Result<(), ValidationError> {
    if page < 1 {
        return Err(err("page", "must be at least 1"));
    }
    if limit < 1 {
        return Err(err("limit", "must be at least 1"));
    }
    if limit > max_limit {
        return Err(err("limit", "exceeds maximum allowed value"));
    }
    Ok(())
}

pub fn sort_dir(field: &str, dir: &str) -> Result<(), ValidationError> {
    match dir.to_lowercase().as_str() {
        "asc" | "desc" => Ok(()),
        _ => Err(err(field, "must be 'asc' or 'desc'")),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn required_string_rejects_empty() {
        assert!(required_string("name", "").is_err());
        assert!(required_string("name", "   ").is_err());
        assert!(required_string("name", "hello").is_ok());
    }

    #[test]
    fn email_validates() {
        assert!(email("email", "user@example.com").is_ok());
        assert!(email("email", "invalid").is_err());
        assert!(email("email", "a@b@c").is_err());
    }

    #[test]
    fn uuid_v4_validates() {
        assert!(uuid_v4("id", "550e8400-e29b-41d4-a716-446655440000").is_ok());
        assert!(uuid_v4("id", "not-a-uuid").is_err());
    }

    #[test]
    fn int_range_validates() {
        assert!(int_range("age", 25, 0, 150).is_ok());
        assert!(int_range("age", -1, 0, 150).is_err());
    }

    #[test]
    fn currency_validates() {
        assert!(currency_iso4217("currency", "USD").is_ok());
        assert!(currency_iso4217("currency", "usd").is_err());
        assert!(currency_iso4217("currency", "US").is_err());
    }

    #[test]
    fn sort_dir_validates() {
        assert!(sort_dir("sort", "asc").is_ok());
        assert!(sort_dir("sort", "DESC").is_ok());
        assert!(sort_dir("sort", "random").is_err());
    }
}
