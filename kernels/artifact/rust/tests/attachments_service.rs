use chrono::{TimeZone, Utc};

use artifact::{AttachmentsService, Clock};

#[derive(Debug, Clone, Copy)]
struct FixedClock;

impl Clock for FixedClock {
    fn now(&self) -> chrono::DateTime<Utc> {
        Utc.with_ymd_and_hms(2026, 3, 20, 12, 0, 0)
            .single()
            .unwrap()
    }
}

#[test]
fn build_storage_key_sanitizes_segments() {
    let service = AttachmentsService::new(FixedClock);
    let key = service
        .build_storage_key("acme", "Invoice", "inv_1", "../factura.pdf")
        .unwrap();

    assert!(key.contains("acme/invoice/inv_1/"));
    assert!(!key.contains(".."));
}

#[test]
fn build_download_link_uses_default_ttl() {
    let service = AttachmentsService::new(FixedClock);
    let link = service
        .build_download_link("https://api.example.com/v1", "att_1", None)
        .unwrap();

    assert!(link.url.contains("/attachments/att_1/download"));
    assert_eq!(
        link.expires_at,
        Utc.with_ymd_and_hms(2026, 3, 20, 12, 15, 0)
            .single()
            .unwrap()
    );
}
