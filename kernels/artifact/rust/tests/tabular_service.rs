use std::collections::BTreeMap;

use artifact::{
    CalamineXlsxCodec, CsvError, Sheet, TabularLimits, TabularService, TabularServiceError,
    TabularValidationError,
};

#[test]
fn csv_includes_bom_and_headers() {
    let service = TabularService::new(CalamineXlsxCodec);
    let content = service
        .csv(&Sheet {
            headers: vec!["name".into(), "email".into()],
            rows: vec![vec!["Juan".into(), "juan@example.com".into()]],
        })
        .unwrap();

    assert!(content.starts_with(&[0xEF, 0xBB, 0xBF]));
    let text = String::from_utf8(content[3..].to_vec()).unwrap();
    assert!(text.contains("name,email"));
}

#[test]
fn xlsx_produces_readable_workbook() {
    let service = TabularService::new(CalamineXlsxCodec);
    let sheet = Sheet {
        headers: vec!["name".into(), "email".into()],
        rows: vec![vec!["Juan".into(), "juan@example.com".into()]],
    };

    let content = service.xlsx(&sheet).unwrap();
    let parsed = service.parse_xlsx(&content, None).unwrap();

    assert_eq!(parsed.headers[0], "name");
    assert_eq!(parsed.rows[0][1], "juan@example.com");
}

#[test]
fn csv_asset_has_expected_metadata() {
    let service = TabularService::new(CalamineXlsxCodec);
    let item = service
        .csv_asset(
            "sales",
            &Sheet {
                headers: vec!["name".into()],
                rows: vec![vec!["acme".into()]],
            },
            &BTreeMap::from([("tenant".into(), "acme".into())]),
        )
        .unwrap();

    assert_eq!(item.name, "sales.csv");
    assert_eq!(item.content_type, "text/csv; charset=utf-8");
    assert!(String::from_utf8(item.body).unwrap().contains("acme"));
}

#[test]
fn parse_csv_matches_go_behavior() {
    let service = TabularService::new(CalamineXlsxCodec);
    let sheet = service
        .parse_csv(b"\xEF\xBB\xBFname,email\nJuan,juan@example.com\n")
        .unwrap();

    assert_eq!(sheet.headers[0], "name");
    assert_eq!(sheet.rows[0][1], "juan@example.com");
}

#[test]
fn csv_rejects_sheet_over_row_limit() {
    let service = TabularService::with_limits(
        CalamineXlsxCodec,
        TabularLimits {
            max_rows: 1,
            ..TabularLimits::default()
        },
    );

    let error = service
        .csv(&Sheet {
            headers: vec!["name".into()],
            rows: vec![vec!["acme".into()], vec!["globex".into()]],
        })
        .unwrap_err();

    assert!(matches!(
        error,
        TabularServiceError::Csv(CsvError::Validation(TabularValidationError::TooManyRows {
            limit: 1,
            actual: 2
        }))
    ));
}

#[test]
fn parse_csv_rejects_oversized_body() {
    let service = TabularService::with_limits(
        CalamineXlsxCodec,
        TabularLimits {
            max_body_bytes: 8,
            ..TabularLimits::default()
        },
    );

    let error = service
        .parse_csv(b"\xEF\xBB\xBFname,email\nJuan,juan@example.com\n")
        .unwrap_err();

    assert!(matches!(
        error,
        TabularServiceError::Csv(CsvError::Validation(TabularValidationError::BodyTooLarge {
            limit: 8,
            ..
        }))
    ));
}

#[test]
fn parse_xlsx_rejects_cell_over_limit() {
    let default_service = TabularService::new(CalamineXlsxCodec);
    let workbook = default_service
        .xlsx(&Sheet {
            headers: vec!["name".into()],
            rows: vec![vec!["abcdefghijklmnopqrstuvwxyz".into()]],
        })
        .unwrap();

    let constrained = TabularService::with_limits(
        CalamineXlsxCodec,
        TabularLimits {
            max_cell_bytes: 8,
            ..TabularLimits::default()
        },
    );
    let error = constrained.parse_xlsx(&workbook, None).unwrap_err();

    assert!(matches!(
        error,
        TabularServiceError::Codec(artifact::TabularCodecError::Validation(
            TabularValidationError::CellTooLarge {
                row: 2,
                column: 1,
                limit: 8,
                ..
            }
        ))
    ));
}
