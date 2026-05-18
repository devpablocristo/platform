//! PostgreSQL pool + migraciones via sqlx.

use sqlx::postgres::{PgPool, PgPoolOptions};
use std::time::Duration;

#[derive(Debug, thiserror::Error)]
pub enum DbError {
    #[error("connect: {0}")]
    Connect(#[from] sqlx::Error),
    #[error("migrate: {0}")]
    Migrate(#[from] sqlx::migrate::MigrateError),
    #[error("config: {0}")]
    Config(String),
}

/// Configuración del pool.
#[derive(Debug, Clone)]
pub struct PoolConfig {
    pub min_connections: u32,
    pub max_connections: u32,
    pub connect_timeout: Duration,
    pub idle_timeout: Duration,
    pub max_lifetime: Duration,
}

impl Default for PoolConfig {
    fn default() -> Self {
        Self {
            min_connections: 1,
            max_connections: 8,
            connect_timeout: Duration::from_secs(5),
            idle_timeout: Duration::from_secs(300),
            max_lifetime: Duration::from_secs(1800),
        }
    }
}

/// Abre un pool PostgreSQL con config default.
pub async fn open(database_url: &str) -> Result<PgPool, DbError> {
    open_with_config(database_url, PoolConfig::default()).await
}

/// Abre un pool PostgreSQL con config explícita y valida conectividad.
pub async fn open_with_config(database_url: &str, config: PoolConfig) -> Result<PgPool, DbError> {
    let url = database_url.trim();
    if url.is_empty() {
        return Err(DbError::Config("DATABASE_URL is required".into()));
    }

    let pool = PgPoolOptions::new()
        .min_connections(config.min_connections)
        .max_connections(config.max_connections)
        .acquire_timeout(config.connect_timeout)
        .idle_timeout(config.idle_timeout)
        .max_lifetime(config.max_lifetime)
        .connect(url)
        .await?;

    tracing::info!("postgres pool opened");
    Ok(pool)
}

/// Ejecuta migraciones embebidas desde un directorio.
pub async fn migrate(pool: &PgPool, migrator: sqlx::migrate::Migrator) -> Result<(), DbError> {
    migrator.run(pool).await?;
    tracing::info!("migrations completed");
    Ok(())
}

/// Ping de salud.
pub async fn ping(pool: &PgPool) -> Result<(), DbError> {
    sqlx::query("SELECT 1").execute(pool).await?;
    Ok(())
}

/// Cierra el pool.
pub async fn close(pool: PgPool) {
    pool.close().await;
    tracing::info!("postgres pool closed");
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn default_config_is_sane() {
        let config = PoolConfig::default();
        assert_eq!(config.min_connections, 1);
        assert_eq!(config.max_connections, 8);
        assert!(config.connect_timeout.as_secs() > 0);
    }

    #[tokio::test]
    async fn open_rejects_empty_url() {
        let result = open("").await;
        assert!(matches!(result, Err(DbError::Config(_))));
    }
}
