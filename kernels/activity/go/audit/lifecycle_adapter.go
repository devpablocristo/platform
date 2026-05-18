package audit

import (
	"context"
	"fmt"

	domain "github.com/devpablocristo/platform/kernels/activity/go/audit/usecases/domain"
	kernel "github.com/devpablocristo/platform/kernels/activity/go/kernel/usecases/domain"
)

// LifecycleAdapter wraps *Usecases and provides an Append* method whose
// signature is shaped to satisfy lifecycle.AuditPort with a thin (~3 line)
// wire-time shim from the consumer.
//
// Direction matters: activity/go MUST NOT import lifecycle/go (that would be
// a circular dependency since lifecycle already depends on AuditPort). The
// consumer writes the wire-time wrapper:
//
//	type auditPortShim struct{ a *audit.LifecycleAdapter }
//	func (s auditPortShim) Append(ctx context.Context, e lifecycle.ArchiveAudit) error {
//	    var reason, batch *string
//	    if e.Reason != nil { reason = e.Reason }
//	    if e.BatchID != nil { s := e.BatchID.String(); batch = &s }
//	    return s.a.AppendArchiveAudit(
//	        ctx, e.TenantID.String(), e.ResourceType, e.ResourceID.String(),
//	        string(e.Action), e.Actor, reason, batch,
//	        map[string]any{"occurred_at": e.OccurredAt},
//	    )
//	}
//
// See platform/docs/integration/lifecycle-audit.md (todo, Ola B6) for a
// worked example.
type LifecycleAdapter struct {
	usecases *Usecases
}

// NewLifecycleAdapter creates an adapter over the canonical audit Usecases.
func NewLifecycleAdapter(usecases *Usecases) *LifecycleAdapter {
	return &LifecycleAdapter{usecases: usecases}
}

// AppendArchiveAudit converts a lifecycle-style entry into the canonical
// LogInput and writes it through the audit Usecases. The returned error is
// surfaced verbatim from the audit subsystem; consumers that prefer to drop
// audit failures silently (legacy pymes behavior) can wrap this call.
//
// resourceID is sent as a string for symmetry with the rest of the kernel —
// the consumer is responsible for stringifying its uuid.UUID at the wire.
func (a *LifecycleAdapter) AppendArchiveAudit(
	ctx context.Context,
	tenantID, resourceType, resourceID, action, actor string,
	reason *string,
	batchID *string,
	payload map[string]any,
) error {
	if a == nil || a.usecases == nil {
		return fmt.Errorf("audit lifecycle adapter not configured")
	}
	merged := map[string]any{}
	for k, v := range payload {
		merged[k] = v
	}
	if reason != nil {
		merged["reason"] = *reason
	}
	if batchID != nil {
		merged["batch_id"] = *batchID
	}

	_, err := a.usecases.Append(ctx, domain.LogInput{
		TenantID: tenantID,
		Actor: kernel.ActorRef{
			Legacy: actor,
			Type:   "user",
			Label:  actor,
		},
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Payload:      merged,
	})
	return err
}
