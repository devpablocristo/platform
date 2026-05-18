# Capability Manifest v1

`capability-manifest.schema.json` is the neutral source of truth for AI
capabilities in the ecosystem. Go and Python types mirror this schema, but the
schema owns the wire contract.

## Ownership Rules

- Product services own transactional business behavior.
- Companion owns orchestration, task flow, memory, routing, and UX.
- Nexus Governance owns policy decisions, approvals, evidence, and replay.
- Core owns reusable contracts and runtime primitives.

## Security Rules

- Manifests describe capabilities only; they never execute work.
- Write tools must declare governance metadata and a Nexus `action_type`.
- Manifests must not include URLs, secrets, API keys, tokens, DSNs, hosts, or
  environment-specific configuration.
- `executor_ref` is a stable logical reference that Companion maps to an
  adapter later; it is not a network address.

## Versioning

- `schema_version` is fixed to `capability_manifest.v1` for this contract.
- `version` is the product capability version and must be semver without a
  leading `v`.
- Breaking manifest changes require a new schema directory.
