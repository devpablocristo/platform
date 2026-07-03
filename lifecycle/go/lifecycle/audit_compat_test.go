package lifecycle

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

type compatAuditPort struct {
	entries []AuditEvent
}

func (p *compatAuditPort) Append(_ context.Context, event AuditEvent) error {
	p.entries = append(p.entries, event)
	return nil
}

func TestAuditEventCompatibilityAliases(t *testing.T) {
	var _ AuditPort = (*compatAuditPort)(nil)

	id := uuid.New()
	event := AuditEvent{
		ID:           id,
		TenantID:     "tenant-a",
		ResourceType: "widget",
		ResourceID:   uuid.New(),
		Action:       ActionStatusChanged,
	}

	var lifecycleAudit LifecycleAudit = event
	var archiveAudit ArchiveAudit = lifecycleAudit
	if archiveAudit.ID != id || archiveAudit.Action != ActionStatusChanged {
		t.Fatalf("unexpected compatibility alias values: %+v", archiveAudit)
	}

	port := &compatAuditPort{}
	if err := port.Append(context.Background(), archiveAudit); err != nil {
		t.Fatal(err)
	}
	if len(port.entries) != 1 || port.entries[0].ID != id {
		t.Fatalf("expected appended audit event, got %+v", port.entries)
	}
}

func TestGenericAuditActions(t *testing.T) {
	for _, action := range []Action{
		ActionCreate,
		ActionUpdate,
		ActionStatusChanged,
		ActionArchive,
		ActionRestore,
		ActionHardDelete,
	} {
		if action == "" {
			t.Fatal("generic audit action must not be empty")
		}
	}
}
