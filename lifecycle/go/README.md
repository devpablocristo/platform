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
