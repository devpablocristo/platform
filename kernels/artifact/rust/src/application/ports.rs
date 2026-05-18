use chrono::{DateTime, Utc};
use thiserror::Error;

use crate::domain::tabular::{Sheet, TabularLimits, TabularValidationError};

pub trait Clock {
    fn now(&self) -> DateTime<Utc>;
}

#[derive(Debug, Error)]
pub enum TabularCodecError {
    #[error(transparent)]
    Validation(#[from] TabularValidationError),
    #[error("{0}")]
    Operation(String),
}

pub trait TabularCodec {
    fn encode_xlsx(
        &self,
        sheet: &Sheet,
        limits: &TabularLimits,
    ) -> Result<Vec<u8>, TabularCodecError>;
    fn decode_xlsx(
        &self,
        body: &[u8],
        sheet_name: Option<&str>,
        limits: &TabularLimits,
    ) -> Result<Sheet, TabularCodecError>;
}
