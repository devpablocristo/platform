use hmac::{Hmac, Mac};
use sha2::Sha256;

use crate::application::ports::{EvidenceSigner, EvidenceSigningError};
use crate::domain::evidence::EvidencePack;

type HmacSha256 = Hmac<Sha256>;

#[derive(Debug, Clone)]
pub struct HmacSha256Signer {
    key: Vec<u8>,
    key_id: String,
}

impl HmacSha256Signer {
    pub fn new(
        signing_key: impl Into<String>,
        key_id: impl Into<String>,
    ) -> Result<Self, EvidenceSigningError> {
        let signing_key = signing_key.into();
        let signing_key = signing_key.trim();
        if signing_key.is_empty() {
            return Err(EvidenceSigningError::SigningKeyRequired);
        }

        let key_id = {
            let value = key_id.into();
            let value = value.trim();
            if value.is_empty() {
                "default".to_string()
            } else {
                value.to_string()
            }
        };

        Ok(Self {
            key: signing_key.as_bytes().to_vec(),
            key_id,
        })
    }
}

impl EvidenceSigner for HmacSha256Signer {
    fn sign(&self, pack: &mut EvidencePack) -> Result<(), EvidenceSigningError> {
        pack.signature = None;
        let payload = serde_json::to_vec(pack)
            .map_err(|error| EvidenceSigningError::Serialization(error.to_string()))?;

        let mut mac = HmacSha256::new_from_slice(&self.key)
            .map_err(|error| EvidenceSigningError::Serialization(error.to_string()))?;
        mac.update(&payload);

        let digest = hex::encode(mac.finalize().into_bytes());
        pack.signature = Some(format!(
            "{}:{}:{}",
            self.key_id,
            pack.generated_at.to_rfc3339(),
            digest
        ));
        Ok(())
    }
}
