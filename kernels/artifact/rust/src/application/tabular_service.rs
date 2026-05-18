use std::collections::BTreeMap;

use thiserror::Error;

use crate::domain::asset::{Asset, Format};
use crate::domain::tabular::{
    csv_bytes_with_limits, parse_csv_bytes_with_limits, CsvError, Sheet, TabularLimits,
};

use super::ports::{TabularCodec, TabularCodecError};

#[derive(Debug, Error)]
pub enum TabularServiceError {
    #[error(transparent)]
    Csv(#[from] CsvError),
    #[error(transparent)]
    Codec(#[from] TabularCodecError),
}

pub struct TabularService<C> {
    codec: C,
    limits: TabularLimits,
}

impl<C> TabularService<C>
where
    C: TabularCodec,
{
    pub fn new(codec: C) -> Self {
        Self::with_limits(codec, TabularLimits::default())
    }

    pub fn with_limits(codec: C, limits: TabularLimits) -> Self {
        Self { codec, limits }
    }

    pub fn limits(&self) -> TabularLimits {
        self.limits
    }

    pub fn csv(&self, sheet: &Sheet) -> Result<Vec<u8>, TabularServiceError> {
        csv_bytes_with_limits(&sheet.headers, &sheet.rows, &self.limits).map_err(Into::into)
    }

    pub fn csv_asset(
        &self,
        name: &str,
        sheet: &Sheet,
        metadata: &BTreeMap<String, String>,
    ) -> Result<Asset, TabularServiceError> {
        let body = self.csv(sheet)?;
        Ok(Asset::new(name, Format::Csv, &body, metadata))
    }

    pub fn xlsx(&self, sheet: &Sheet) -> Result<Vec<u8>, TabularServiceError> {
        self.codec
            .encode_xlsx(sheet, &self.limits)
            .map_err(Into::into)
    }

    pub fn xlsx_asset(
        &self,
        name: &str,
        sheet: &Sheet,
        metadata: &BTreeMap<String, String>,
    ) -> Result<Asset, TabularServiceError> {
        let body = self.xlsx(sheet)?;
        Ok(Asset::new(name, Format::Xlsx, &body, metadata))
    }

    pub fn parse_csv(&self, body: &[u8]) -> Result<Sheet, TabularServiceError> {
        parse_csv_bytes_with_limits(body, &self.limits).map_err(Into::into)
    }

    pub fn parse_xlsx(
        &self,
        body: &[u8],
        sheet_name: Option<&str>,
    ) -> Result<Sheet, TabularServiceError> {
        self.codec
            .decode_xlsx(body, sheet_name, &self.limits)
            .map_err(Into::into)
    }
}
