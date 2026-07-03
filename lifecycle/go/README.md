# platform/lifecycle/go

Canonical lifecycle primitives for Go services.

The model is intentionally explicit:

- `active`: participates in normal product workflows.
- `archived`: retained, reversible, and excluded from active workflows by default.
- `trashed`: reversible delete state, usually with a retention window.
- `purged`: irreversible deletion.

Archive is not deletion. Trash is reversible deletion. Purge is irreversible.

Tenants, actors and resource types are opaque strings so products can use UUIDs,
slugs or local IDs without adapters. Consumers provide repositories and policies;
this module provides the lifecycle mechanics, audit records and policy checks.

## Generic audit

The public audit shape is `lifecycle.AuditEvent` and the persistence port is
`lifecycle.AuditPort`.

The event is intentionally generic and can also be used by consumers for
non-archive lifecycle actions such as create, update or status changes:

- `ActionCreate`
- `ActionUpdate`
- `ActionStatusChanged`
- `ActionArchive`
- `ActionRestore`
- `ActionHardDelete`

Compatibility names remain available:

- `LifecycleAudit` is an alias of `AuditEvent`.
- `ArchiveAudit` is an alias of `AuditEvent`.
