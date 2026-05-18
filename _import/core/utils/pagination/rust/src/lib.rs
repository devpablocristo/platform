//! Paginación cursor-based y offset-based con generics reales.

use serde::{Deserialize, Serialize};

/// Configuración de paginación.
#[derive(Debug, Clone)]
pub struct Config {
    pub default_limit: usize,
    pub max_limit: usize,
}

impl Default for Config {
    fn default() -> Self {
        Self {
            default_limit: 20,
            max_limit: 100,
        }
    }
}

/// Parámetros de paginación cursor-based.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CursorParams {
    pub limit: usize,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub cursor: Option<String>,
}

/// Página de resultados cursor-based.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CursorPage<T> {
    pub items: Vec<T>,
    pub has_more: bool,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub next_cursor: Option<String>,
}

/// Parámetros de paginación offset-based.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OffsetParams {
    pub page: usize,
    pub per_page: usize,
}

/// Página de resultados offset-based.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OffsetPage<T> {
    pub items: Vec<T>,
    pub page: usize,
    pub per_page: usize,
    pub total: u64,
}

/// Normaliza limit con defaults y techo.
pub fn normalize_limit(limit: usize, config: &Config) -> usize {
    if limit == 0 {
        return config.default_limit;
    }
    limit.min(config.max_limit)
}

/// Parsea CursorParams desde strings HTTP con normalización.
pub fn parse_cursor_params(raw_limit: Option<&str>, raw_cursor: Option<&str>, config: &Config) -> CursorParams {
    let limit = raw_limit
        .and_then(|s| s.trim().parse::<usize>().ok())
        .map(|n| normalize_limit(n, config))
        .unwrap_or(config.default_limit);

    let cursor = raw_cursor
        .map(|s| s.trim().to_owned())
        .filter(|s| !s.is_empty());

    CursorParams { limit, cursor }
}

/// Construye una CursorPage copiando items.
pub fn build_cursor_page<T: Clone>(items: Vec<T>, has_more: bool, next_cursor: Option<String>) -> CursorPage<T> {
    CursorPage {
        items,
        has_more,
        next_cursor,
    }
}

/// Parsea OffsetParams desde strings HTTP.
pub fn parse_offset_params(raw_page: Option<&str>, raw_per_page: Option<&str>, config: &Config) -> OffsetParams {
    let page = raw_page
        .and_then(|s| s.trim().parse::<usize>().ok())
        .filter(|&n| n >= 1)
        .unwrap_or(1);

    let per_page = raw_per_page
        .and_then(|s| s.trim().parse::<usize>().ok())
        .map(|n| normalize_limit(n, config))
        .unwrap_or(config.default_limit);

    OffsetParams { page, per_page }
}

impl OffsetParams {
    /// Offset SQL para LIMIT/OFFSET.
    pub fn offset(&self) -> u64 {
        ((self.page.saturating_sub(1)) * self.per_page) as u64
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn normalize_limit_applies_max() {
        let config = Config { default_limit: 20, max_limit: 50 };
        assert_eq!(normalize_limit(100, &config), 50);
        assert_eq!(normalize_limit(0, &config), 20);
        assert_eq!(normalize_limit(30, &config), 30);
    }

    #[test]
    fn cursor_params_defaults() {
        let params = parse_cursor_params(None, None, &Config::default());
        assert_eq!(params.limit, 20);
        assert!(params.cursor.is_none());
    }

    #[test]
    fn offset_params_calculates_offset() {
        let params = OffsetParams { page: 3, per_page: 20 };
        assert_eq!(params.offset(), 40);
    }

    #[test]
    fn build_page_with_items() {
        let page = build_cursor_page(vec![1, 2, 3], true, Some("abc".into()));
        assert_eq!(page.items.len(), 3);
        assert!(page.has_more);
        assert_eq!(page.next_cursor, Some("abc".into()));
    }
}
