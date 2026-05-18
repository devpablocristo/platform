use std::io::Cursor;

use calamine::{open_workbook_auto_from_rs, Data, Reader};
use rust_xlsxwriter::{Format, Workbook};

use crate::application::ports::{TabularCodec, TabularCodecError};
use crate::domain::tabular::{Sheet, TabularLimits, TabularValidationError};

#[derive(Debug, Default, Clone, Copy)]
pub struct CalamineXlsxCodec;

impl TabularCodec for CalamineXlsxCodec {
    fn encode_xlsx(
        &self,
        sheet: &Sheet,
        limits: &TabularLimits,
    ) -> Result<Vec<u8>, TabularCodecError> {
        sheet.validate(limits)?;
        let mut workbook = Workbook::new();
        let worksheet = workbook.add_worksheet();
        let header_format = Format::new().set_bold();

        for (col, header) in sheet.headers.iter().enumerate() {
            let col = col as u16;
            worksheet
                .write_string_with_format(0, col, header.as_str(), &header_format)
                .map_err(|error| TabularCodecError::Operation(error.to_string()))?;
            worksheet
                .set_column_width(col, 18.0)
                .map_err(|error| TabularCodecError::Operation(error.to_string()))?;
        }

        for (row_idx, row) in sheet.rows.iter().enumerate() {
            let row_idx = (row_idx + 1) as u32;
            for (col_idx, value) in row.iter().enumerate() {
                worksheet
                    .write_string(row_idx, col_idx as u16, value.as_str())
                    .map_err(|error| TabularCodecError::Operation(error.to_string()))?;
            }
        }

        workbook
            .save_to_buffer()
            .map_err(|error| TabularCodecError::Operation(error.to_string()))
    }

    fn decode_xlsx(
        &self,
        body: &[u8],
        sheet_name: Option<&str>,
        limits: &TabularLimits,
    ) -> Result<Sheet, TabularCodecError> {
        if body.len() > limits.max_body_bytes {
            return Err(TabularValidationError::BodyTooLarge {
                limit: limits.max_body_bytes,
                actual: body.len(),
            }
            .into());
        }

        let mut workbook = open_workbook_auto_from_rs(Cursor::new(body))
            .map_err(|error| TabularCodecError::Operation(error.to_string()))?;

        let selected = match sheet_name.map(str::trim) {
            Some(name) if !name.is_empty() => name.to_string(),
            _ => workbook.sheet_names().first().cloned().ok_or_else(|| {
                TabularCodecError::Operation("xlsx workbook has no sheets".into())
            })?,
        };

        let range = workbook
            .worksheet_range(selected.as_str())
            .map_err(|error| TabularCodecError::Operation(error.to_string()))?;

        let mut rows = range.rows();
        let Some(headers) = rows.next() else {
            return Ok(Sheet::default());
        };
        let headers = headers.iter().map(cell_to_string).collect::<Vec<_>>();
        let mut data_rows = Vec::new();

        for (index, row) in rows.enumerate() {
            if data_rows.len() >= limits.max_rows {
                return Err(TabularValidationError::TooManyRows {
                    limit: limits.max_rows,
                    actual: data_rows.len() + 1,
                }
                .into());
            }
            let values = row.iter().map(cell_to_string).collect::<Vec<_>>();
            validate_row(values.as_slice(), index + 2, limits)?;
            data_rows.push(values);
        }

        let sheet = Sheet {
            headers,
            rows: data_rows,
        };
        sheet.validate(limits)?;

        Ok(sheet)
    }
}

fn cell_to_string(cell: &Data) -> String {
    match cell {
        Data::Empty => String::new(),
        Data::String(value) | Data::DateTimeIso(value) | Data::DurationIso(value) => value.clone(),
        Data::Int(value) => value.to_string(),
        Data::Float(value) => value.to_string(),
        Data::Bool(value) => value.to_string(),
        Data::DateTime(value) => value.to_string(),
        Data::Error(value) => format!("{value:?}"),
    }
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
