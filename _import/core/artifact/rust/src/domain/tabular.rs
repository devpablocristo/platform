use serde::{Deserialize, Serialize};
use thiserror::Error;

#[derive(Debug, Clone, PartialEq, Eq, Default, Serialize, Deserialize)]
pub struct Sheet {
    pub headers: Vec<String>,
    pub rows: Vec<Vec<String>>,
}

impl Sheet {
    pub fn validate(&self, limits: &TabularLimits) -> Result<(), TabularValidationError> {
        validate_row(self.headers.as_slice(), 1, limits)?;
        if self.rows.len() > limits.max_rows {
            return Err(TabularValidationError::TooManyRows {
                limit: limits.max_rows,
                actual: self.rows.len(),
            });
        }

        for (index, row) in self.rows.iter().enumerate() {
            validate_row(row.as_slice(), index + 2, limits)?;
        }

        Ok(())
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct TabularLimits {
    pub max_body_bytes: usize,
    pub max_rows: usize,
    pub max_columns: usize,
    pub max_cell_bytes: usize,
}

impl Default for TabularLimits {
    fn default() -> Self {
        Self {
            max_body_bytes: 10 * 1024 * 1024,
            max_rows: 50_000,
            max_columns: 256,
            max_cell_bytes: 64 * 1024,
        }
    }
}

#[derive(Debug, Error, Clone, PartialEq, Eq)]
pub enum TabularValidationError {
    #[error("tabular body exceeds byte limit {limit}: {actual}")]
    BodyTooLarge { limit: usize, actual: usize },
    #[error("tabular row count exceeds limit {limit}: {actual}")]
    TooManyRows { limit: usize, actual: usize },
    #[error("tabular column count exceeds limit {limit} at row {row}: {actual}")]
    TooManyColumns {
        row: usize,
        limit: usize,
        actual: usize,
    },
    #[error("tabular cell exceeds byte limit {limit} at row {row}, column {column}: {actual}")]
    CellTooLarge {
        row: usize,
        column: usize,
        limit: usize,
        actual: usize,
    },
}

#[derive(Debug, Error)]
pub enum CsvError {
    #[error(transparent)]
    Validation(#[from] TabularValidationError),
    #[error(transparent)]
    Csv(#[from] csv::Error),
    #[error(transparent)]
    Utf8(#[from] std::string::FromUtf8Error),
}

pub fn csv_bytes(headers: &[String], rows: &[Vec<String>]) -> Result<Vec<u8>, CsvError> {
    csv_bytes_with_limits(headers, rows, &TabularLimits::default())
}

pub fn csv_bytes_with_limits(
    headers: &[String],
    rows: &[Vec<String>],
    limits: &TabularLimits,
) -> Result<Vec<u8>, CsvError> {
    Sheet {
        headers: headers.to_vec(),
        rows: rows.to_vec(),
    }
    .validate(limits)?;

    let mut writer = csv::Writer::from_writer(Vec::new());
    writer.write_record(headers)?;
    for row in rows {
        writer.write_record(row)?;
    }

    let mut out = vec![0xEF, 0xBB, 0xBF];
    let bytes = writer
        .into_inner()
        .map_err(|error| CsvError::Csv(error.into_error().into()))?;
    out.extend(bytes);
    Ok(out)
}

pub fn parse_csv_bytes(body: &[u8]) -> Result<Sheet, CsvError> {
    parse_csv_bytes_with_limits(body, &TabularLimits::default())
}

pub fn parse_csv_bytes_with_limits(body: &[u8], limits: &TabularLimits) -> Result<Sheet, CsvError> {
    if body.len() > limits.max_body_bytes {
        return Err(TabularValidationError::BodyTooLarge {
            limit: limits.max_body_bytes,
            actual: body.len(),
        }
        .into());
    }

    let body = body.strip_prefix(&[0xEF, 0xBB, 0xBF]).unwrap_or(body);
    let mut reader = csv::ReaderBuilder::new()
        .has_headers(false)
        .from_reader(body);
    let mut records = reader.records();

    let Some(headers) = records.next() else {
        return Ok(Sheet::default());
    };
    let headers = headers?.iter().map(ToString::to_string).collect::<Vec<_>>();
    validate_row(headers.as_slice(), 1, limits)?;

    let mut rows = Vec::new();
    for (index, record) in records.enumerate() {
        let row_number = index + 2;
        if rows.len() >= limits.max_rows {
            return Err(TabularValidationError::TooManyRows {
                limit: limits.max_rows,
                actual: rows.len() + 1,
            }
            .into());
        }
        let row = record?.iter().map(ToString::to_string).collect::<Vec<_>>();
        validate_row(row.as_slice(), row_number, limits)?;
        rows.push(row);
    }

    Ok(Sheet { headers, rows })
}

fn validate_row(
    row: &[String],
    row_index: usize,
    limits: &TabularLimits,
) -> Result<(), TabularValidationError> {
    if row.len() > limits.max_columns {
        return Err(TabularValidationError::TooManyColumns {
            row: row_index,
            limit: limits.max_columns,
            actual: row.len(),
        });
    }

    for (index, value) in row.iter().enumerate() {
        let size = value.len();
        if size > limits.max_cell_bytes {
            return Err(TabularValidationError::CellTooLarge {
                row: row_index,
                column: index + 1,
                limit: limits.max_cell_bytes,
                actual: size,
            });
        }
    }

    Ok(())
}
