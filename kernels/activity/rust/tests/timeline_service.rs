use std::collections::BTreeMap;

use chrono::{TimeZone, Utc};
use serde_json::json;
use tempfile::tempdir;

use activity::{
    Clock, FileSystemTimelineRepository, IdGenerator, InMemoryTimelineRepository,
    TimelineRecordInput, TimelineService,
};

#[derive(Debug, Clone, Copy)]
struct FixedClock;

impl Clock for FixedClock {
    fn now(&self) -> chrono::DateTime<Utc> {
        Utc.with_ymd_and_hms(2026, 3, 20, 12, 0, 0)
            .single()
            .unwrap()
    }
}

#[derive(Debug, Clone, Copy)]
struct StaticIds;

impl IdGenerator for StaticIds {
    fn new_id(&self) -> String {
        "timeline-id".to_string()
    }
}

#[test]
fn record_normalizes_entry() {
    let repo = InMemoryTimelineRepository::default();
    let service = TimelineService::new(repo, StaticIds, FixedClock);

    let entry = service
        .record(TimelineRecordInput {
            id: String::new(),
            tenant_id: "acme".into(),
            entity_type: "invoice".into(),
            entity_id: "inv_1".into(),
            event_type: "invoice.sent".into(),
            title: " Invoice sent ".into(),
            description: " queued ".into(),
            actor: " pablo ".into(),
            metadata: BTreeMap::from([
                ("status".into(), json!("sent")),
                ("".into(), json!("ignored")),
            ]),
            created_at: None,
        })
        .unwrap();

    assert_eq!(entry.id, "timeline-id");
    assert_eq!(entry.title, "Invoice sent");
    assert_eq!(entry.description, "queued");
    assert_eq!(entry.actor, "pablo");
    assert_eq!(entry.metadata.len(), 1);
}

#[test]
fn list_filters_by_entity() {
    let repo = InMemoryTimelineRepository::default();
    let service = TimelineService::new(repo, StaticIds, FixedClock);

    service
        .record(TimelineRecordInput {
            id: "evt_1".into(),
            tenant_id: "acme".into(),
            entity_type: "invoice".into(),
            entity_id: "inv_1".into(),
            event_type: "invoice.sent".into(),
            title: "Invoice sent".into(),
            description: String::new(),
            actor: String::new(),
            metadata: BTreeMap::new(),
            created_at: Some(
                Utc.with_ymd_and_hms(2026, 3, 20, 12, 0, 0)
                    .single()
                    .unwrap(),
            ),
        })
        .unwrap();
    service
        .record(TimelineRecordInput {
            id: "evt_2".into(),
            tenant_id: "acme".into(),
            entity_type: "invoice".into(),
            entity_id: "inv_2".into(),
            event_type: "invoice.sent".into(),
            title: "Invoice sent".into(),
            description: String::new(),
            actor: String::new(),
            metadata: BTreeMap::new(),
            created_at: None,
        })
        .unwrap();

    let items = service.list("acme", "invoice", "inv_1", 10).unwrap();

    assert_eq!(items.len(), 1);
    assert_eq!(items[0].id, "evt_1");
}

#[test]
fn file_system_repo_persists_timeline_entries() {
    let dir = tempdir().unwrap();
    let service = TimelineService::new(
        FileSystemTimelineRepository::new(dir.path()),
        StaticIds,
        FixedClock,
    );

    service
        .record(TimelineRecordInput {
            id: "evt_1".into(),
            tenant_id: "acme".into(),
            entity_type: "invoice".into(),
            entity_id: "inv_1".into(),
            event_type: "invoice.sent".into(),
            title: "Invoice sent".into(),
            description: String::new(),
            actor: String::new(),
            metadata: BTreeMap::new(),
            created_at: None,
        })
        .unwrap();

    let reloaded = TimelineService::new(
        FileSystemTimelineRepository::new(dir.path()),
        StaticIds,
        FixedClock,
    );
    let items = reloaded.list("acme", "invoice", "inv_1", 10).unwrap();

    assert_eq!(items.len(), 1);
    assert_eq!(items[0].event_type, "invoice.sent");
}
