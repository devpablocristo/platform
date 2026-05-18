# Capability Manifest v1

Canonical contract: `../contracts/capabilities/v1/capability-manifest.schema.json`.

This document locks the Pymes -> Companion migration requirements on top of
the existing schema.

## Required semantics

- A capability is a named business intent, never a generic endpoint wrapper.
- Product backends own execution and final authorization.
- Companion discovers, selects and invokes capabilities, but does not grant
  domain permissions.
- Writes must declare governance metadata and a Nexus `action_type`.
- Manifests must not contain URLs, credentials, secrets, DSNs or deploy env.

## Required actor context for execution

Capability execution requests must carry:

- `tenant_id`
- `actor_id`
- `actor_type`
- `role`
- `scopes`
- `conversation_id`
- `trace_id`

The product backend must revalidate this context against its own membership and
RBAC model before reading or writing data.

## Pymes v1 capabilities

First vertical slice:

- `pymes.customers.search`
- `pymes.services.search`
- `pymes.inventory.search`
- `pymes.scheduling.book`

Disallowed shapes:

- `pymes.execute_sql`
- `pymes.call_endpoint`
- `pymes.update_entity`
- any capability that lets Companion choose arbitrary tables, URLs or resource
  names.
