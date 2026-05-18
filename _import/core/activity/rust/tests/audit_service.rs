use std::collections::BTreeMap;

use chrono::{TimeZone, Utc};
use serde_json::json;
use tempfile::tempdir;

use activity::{
    build_hash, ActorRef, AuditEntry, AuditLogInput, AuditRepository, AuditService, Clock,
    FileSystemAuditRepository, IdGenerator, InMemoryAuditRepository, RepositoryError,
};

#[derive(Debug, Clone, Copy)]
struct FixedClock;

impl Clock for FixedClock {
    fn now(&self) -> chrono::DateTime<Utc> {
        Utc.with_ymd_and_hms(2026, 3, 20, 10, 0, 0)
            .single()
            .unwrap()
    }
}

#[derive(Debug, Clone, Copy)]
struct StaticIds;

impl IdGenerator for StaticIds {
    fn new_id(&self) -> String {
        "generated-id".to_string()
    }
}

#[test]
fn append_builds_hash_chain() {
    let repo = InMemoryAuditRepository::default();
    let service = AuditService::new(repo, StaticIds, FixedClock);

    let first = service
        .append(AuditLogInput {
            tenant_id: "acme".into(),
            actor: ActorRef {
                legacy: "pablo".into(),
                actor_type: "user".into(),
                id: String::new(),
                label: String::new(),
            },
            action: "invoice.created".into(),
            resource_type: "invoice".into(),
            resource_id: "inv_1".into(),
            payload: BTreeMap::from([("amount".into(), json!(10))]),
            id: String::new(),
            created_at: None,
        })
        .unwrap();

    let second = service
        .append(AuditLogInput {
            tenant_id: "acme".into(),
            actor: ActorRef::default(),
            action: "invoice.sent".into(),
            resource_type: "invoice".into(),
            resource_id: "inv_1".into(),
            payload: BTreeMap::new(),
            id: "evt_2".into(),
            created_at: None,
        })
        .unwrap();

    assert_eq!(first.id, "generated-id");
    assert_eq!(second.prev_hash, first.hash);
    assert!(!second.hash.is_empty());
}

#[test]
fn export_csv_and_jsonl_include_action() {
    let repo = InMemoryAuditRepository::default();
    let service = AuditService::new(repo, StaticIds, FixedClock);

    service
        .append(AuditLogInput {
            tenant_id: "acme".into(),
            actor: ActorRef::default(),
            action: "invoice.created".into(),
            resource_type: "invoice".into(),
            resource_id: "inv_1".into(),
            payload: BTreeMap::from([("amount".into(), json!(10))]),
            id: "evt_1".into(),
            created_at: Some(
                Utc.with_ymd_and_hms(2026, 3, 20, 10, 0, 0)
                    .single()
                    .unwrap(),
            ),
        })
        .unwrap();

    let csv = service.export_csv("acme", 10).unwrap();
    let jsonl = service.export_jsonl("acme", 10).unwrap();

    assert!(csv.contains("invoice.created"));
    assert!(jsonl.contains("\"action\":\"invoice.created\""));
}

#[test]
fn file_system_repo_persists_entries_across_instances() {
    let dir = tempdir().unwrap();
    let service = AuditService::new(
        FileSystemAuditRepository::new(dir.path()),
        StaticIds,
        FixedClock,
    );

    service
        .append(AuditLogInput {
            tenant_id: "acme".into(),
            actor: ActorRef::default(),
            action: "invoice.created".into(),
            resource_type: "invoice".into(),
            resource_id: "inv_1".into(),
            payload: BTreeMap::new(),
            id: "evt_1".into(),
            created_at: None,
        })
        .unwrap();

    let reloaded = AuditService::new(
        FileSystemAuditRepository::new(dir.path()),
        StaticIds,
        FixedClock,
    );
    let items = reloaded.list("acme", 10).unwrap();

    assert_eq!(items.len(), 1);
    assert_eq!(items[0].id, "evt_1");
    assert_eq!(items[0].action, "invoice.created");
}

#[test]
fn file_system_repo_rejects_stale_hash_chain() {
    let dir = tempdir().unwrap();
    let repo = FileSystemAuditRepository::new(dir.path());
    let service = AuditService::new(
        FileSystemAuditRepository::new(dir.path()),
        StaticIds,
        FixedClock,
    );

    let first = service
        .append(AuditLogInput {
            tenant_id: "acme".into(),
            actor: ActorRef::default(),
            action: "invoice.created".into(),
            resource_type: "invoice".into(),
            resource_id: "inv_1".into(),
            payload: BTreeMap::new(),
            id: "evt_1".into(),
            created_at: None,
        })
        .unwrap();

    let mut stale = AuditEntry {
        id: "evt_2".into(),
        tenant_id: "acme".into(),
        actor: String::new(),
        actor_type: "user".into(),
        actor_id: String::new(),
        actor_label: String::new(),
        action: "invoice.sent".into(),
        resource_type: "invoice".into(),
        resource_id: "inv_1".into(),
        payload: BTreeMap::new(),
        prev_hash: String::new(),
        hash: String::new(),
        created_at: first.created_at,
    };
    stale.hash = build_hash(&stale).unwrap();

    let error = repo.append(stale).unwrap_err();
    assert!(matches!(error, RepositoryError::Conflict(_)));
}
