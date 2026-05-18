//! Autenticación inbound: credencial → Principal.
//! JWT (JWKS) + API key. Traits async para inyección.

use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use subtle::ConstantTimeEq;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Principal {
    pub org_id: String,
    pub actor: String,
    pub role: String,
    pub scopes: Vec<String>,
    pub auth_method: String,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub claims: Option<serde_json::Value>,
}

/// Credencial extraída de un request HTTP.
#[derive(Debug, Clone)]
pub enum Credential {
    Bearer(String),
    ApiKey(String),
}

impl Credential {
    pub fn kind(&self) -> &'static str {
        match self {
            Self::Bearer(_) => "bearer",
            Self::ApiKey(_) => "api_key",
        }
    }
}

/// Trait para verificar credenciales.
#[async_trait::async_trait]
pub trait Authenticator: Send + Sync {
    async fn authenticate(&self, cred: &Credential) -> Result<Principal, AuthnError>;
}

#[derive(Debug, thiserror::Error)]
pub enum AuthnError {
    #[error("no valid credential")]
    NoCredential,
    #[error("wrong credential kind")]
    WrongKind,
    #[error("invalid token: {0}")]
    InvalidToken(String),
    #[error("invalid api key")]
    InvalidApiKey,
    #[error("provider error: {0}")]
    Provider(String),
}

// --- Extractors ---

/// Extrae Bearer token del header Authorization.
pub fn extract_bearer(auth_header: &str) -> Option<String> {
    let trimmed = auth_header.trim();
    if trimmed.len() > 7 && trimmed[..7].eq_ignore_ascii_case("bearer ") {
        let token = trimmed[7..].trim();
        if !token.is_empty() {
            return Some(token.to_owned());
        }
    }
    None
}

/// Extrae API key de X-API-Key header.
pub fn extract_api_key(x_api_key: &str) -> Option<String> {
    let trimmed = x_api_key.trim();
    if trimmed.is_empty() {
        None
    } else {
        Some(trimmed.to_owned())
    }
}

/// Intenta autenticación JWT primero, luego API key.
pub async fn try_inbound(
    jwt_auth: Option<&dyn Authenticator>,
    api_key_auth: Option<&dyn Authenticator>,
    authorization: &str,
    x_api_key: &str,
) -> Result<(Principal, &'static str), AuthnError> {
    if let Some(auth) = jwt_auth {
        if let Some(token) = extract_bearer(authorization) {
            match auth.authenticate(&Credential::Bearer(token)).await {
                Ok(p) => return Ok((p, "jwt")),
                Err(e) => {
                    if api_key_auth.is_none() {
                        return Err(e);
                    }
                }
            }
        }
    }

    if let Some(auth) = api_key_auth {
        if let Some(key) = extract_api_key(x_api_key) {
            let p = auth.authenticate(&Credential::ApiKey(key)).await?;
            return Ok((p, "api_key"));
        }
    }

    Err(AuthnError::NoCredential)
}

// --- API Key authenticator con SHA-256 ---

/// Entry de API key: nombre + hash SHA-256.
#[derive(Debug, Clone)]
pub struct ApiKeyEntry {
    pub name: String,
    pub hash: [u8; 32],
}

/// Authenticator de API keys por SHA-256 constant-time.
pub struct ApiKeyAuthenticator {
    entries: Vec<ApiKeyEntry>,
}

impl ApiKeyAuthenticator {
    /// Parsea config "name=secret,name2=secret2".
    pub fn from_config(raw: &str) -> Result<Self, AuthnError> {
        let mut entries = Vec::new();
        for pair in raw.split(',') {
            let pair = pair.trim();
            if pair.is_empty() {
                continue;
            }
            let (name, secret) = pair
                .split_once('=')
                .ok_or_else(|| AuthnError::Provider("expected name=secret".into()))?;
            let name = name.trim();
            let secret = secret.trim();
            if name.is_empty() || secret.is_empty() {
                return Err(AuthnError::Provider("empty name or secret".into()));
            }
            let hash: [u8; 32] = Sha256::digest(secret.as_bytes()).into();
            entries.push(ApiKeyEntry { name: name.to_owned(), hash });
        }
        if entries.is_empty() {
            return Err(AuthnError::Provider("no api keys configured".into()));
        }
        Ok(Self { entries })
    }
}

#[async_trait::async_trait]
impl Authenticator for ApiKeyAuthenticator {
    async fn authenticate(&self, cred: &Credential) -> Result<Principal, AuthnError> {
        let key = match cred {
            Credential::ApiKey(k) => k,
            _ => return Err(AuthnError::WrongKind),
        };
        let sum: [u8; 32] = Sha256::digest(key.as_bytes()).into();
        for entry in &self.entries {
            if sum.ct_eq(&entry.hash).into() {
                return Ok(Principal {
                    org_id: String::new(),
                    actor: format!("api_key:{}", entry.name),
                    role: "service".into(),
                    scopes: vec![],
                    auth_method: "api_key".into(),
                    claims: None,
                });
            }
        }
        Err(AuthnError::InvalidApiKey)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn extract_bearer_works() {
        assert_eq!(extract_bearer("Bearer abc123"), Some("abc123".into()));
        assert_eq!(extract_bearer("bearer  xyz"), Some("xyz".into()));
        assert_eq!(extract_bearer("Basic abc"), None);
        assert_eq!(extract_bearer(""), None);
    }

    #[test]
    fn extract_api_key_works() {
        assert_eq!(extract_api_key("my-key"), Some("my-key".into()));
        assert_eq!(extract_api_key("  "), None);
    }

    #[tokio::test]
    async fn api_key_authenticator_validates() {
        let auth = ApiKeyAuthenticator::from_config("admin=secret123").unwrap();
        let result = auth.authenticate(&Credential::ApiKey("secret123".into())).await;
        assert!(result.is_ok());
        let p = result.unwrap();
        assert_eq!(p.actor, "api_key:admin");
        assert_eq!(p.auth_method, "api_key");
    }

    #[tokio::test]
    async fn api_key_authenticator_rejects_invalid() {
        let auth = ApiKeyAuthenticator::from_config("admin=secret123").unwrap();
        let result = auth.authenticate(&Credential::ApiKey("wrong".into())).await;
        assert!(matches!(result, Err(AuthnError::InvalidApiKey)));
    }

    #[tokio::test]
    async fn try_inbound_falls_back_to_api_key() {
        let auth = ApiKeyAuthenticator::from_config("svc=key1").unwrap();
        let (p, method) = try_inbound(None, Some(&auth), "", "key1").await.unwrap();
        assert_eq!(method, "api_key");
        assert_eq!(p.actor, "api_key:svc");
    }
}
