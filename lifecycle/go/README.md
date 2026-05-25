# platform/lifecycle/go

Canonical CRUDAR lifecycle primitives for Go services.

This module owns the shared mechanics for archive, restore, hard delete, bulk
archive, retention policy hooks and audit records. Tenants are represented as
opaque `string` values so products can use UUIDs, slugs, or local IDs such as
`argos-local-org` without adapters.

Consumers provide the resource vocabulary and policies. The package does not
know product entities.
