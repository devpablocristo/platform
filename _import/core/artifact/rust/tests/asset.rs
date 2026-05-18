use std::collections::BTreeMap;

use chrono::{TimeZone, Utc};

use artifact::{build_filename, slug, Asset, Format};

#[test]
fn new_normalizes_asset() {
    let metadata = BTreeMap::from([("tenant".to_string(), "acme".to_string())]);
    let item = Asset::new("sales_export", Format::Csv, b"body", &metadata);

    assert_eq!(item.name, "sales_export.csv");
    assert_eq!(item.content_type, "text/csv; charset=utf-8");
    assert_eq!(item.size(), 4);
    assert_eq!(
        item.metadata.get("tenant").map(String::as_str),
        Some("acme")
    );
}

#[test]
fn build_filename_matches_go_behavior() {
    let got = build_filename(
        &["Sales Report", "ACME/Prod"],
        Format::Pdf,
        Some(Utc.with_ymd_and_hms(2026, 3, 20, 0, 0, 0).single().unwrap()),
    );

    assert_eq!(got, "sales_report_acme_prod_2026-03-20.pdf");
    assert_eq!(slug("  Sales/Report  "), "sales_report");
}
