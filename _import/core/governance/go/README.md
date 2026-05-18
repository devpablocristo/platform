# core/governance/go

Cliente HTTP + DTOs públicos del API contract de **Nexus Governance**.

Este módulo expone **solo** la superficie HTTP. No contiene lógica de
governance: la evaluación de policies, el motor de risk, el motor de
approvals y el dominio canónico viven dentro de **Nexus**
(`nexus/governance/internal/`). Cualquier servicio que necesite tomar una
decisión gobernada (Companion, Pymes, Ponti, futuros productos) debe
llamar a Nexus por HTTP usando este cliente.

## Qué expone

```
governanceclient/
├── client.go     # Client + métodos HTTP (SimulateRequest, SubmitRequest, …)
├── dto.go        # DTOs públicos del wire (Status*, PolicyMode*, ...)
└── options.go    # RequestOption (WithOrgID, WithIdempotencyKey)
```

### Métodos principales

| Método | Endpoint Nexus | Uso típico |
|---|---|---|
| `SimulateRequest(ctx, body, opts...)` | `POST /v1/requests/simulate` | Hot path, evalúa sin persistir. Útil para decisiones síncronas. |
| `SubmitRequest(ctx, idem, body)` | `POST /v1/requests` | Audit path, persiste request + dispara approval si corresponde. |
| `GetRequest(ctx, id)` | `GET /v1/requests/{id}` | Pollear estado de un request. |
| `ReportResult(ctx, id, ...)` | `POST /v1/requests/{id}/result` | Reportar éxito/falla de ejecución post-approval. |
| `ListPolicies / GetPolicy / CreatePolicy / UpdatePolicy / DeletePolicy` | `/v1/policies` | CRUD per-tenant. Multi-tenant callers usan `WithOrgID(tenantID)` + scope `nexus:cross_org`. |
| `Approve / Reject` | `/v1/approvals/{id}/...` | Cuando un humano aprueba en una UI propia (NO en LLM). |
| `SubmitProposal` | `POST /v1/learning/proposals` | Subir un candidate proposal de policy generado externamente. |
| `ListPendingApprovals / ListRequests / ListActionTypes` | varios | Lecturas para dashboards / sync loops. |

### Multi-tenancy

Callers que sirven a varios tenants desde una sola API key (ej: Pymes
service per-customer) usan:

```go
client.SimulateRequest(ctx, body, governanceclient.WithOrgID(tenantUUID))
```

Esto setea el header `X-Org-ID`. La API key del caller debe tener scope
`nexus:cross_org` para que Nexus acepte el override.

### KnownStatuses (contract test)

`governanceclient.KnownStatuses` lista todos los valores válidos del
campo `status` que Nexus retorna. Si tu servicio mapea esos statuses a
una FSM propia, **iterá esa slice como contract test**: cuando Nexus
agregue un status nuevo a la slice, tu test falla y obliga a actualizar
el mapping. Ver
[`companion/internal/tasks/task_fsm_test.go`](../../../companion/internal/tasks/task_fsm_test.go)
como ejemplo.

## Versionado

```
v0.4.x  — solo cliente + DTOs (este módulo).
v0.3.x  — antes del trim final: incluía aún kernel/decision/policy/risk/approval embebidos.
v0.2.x  — antes de Pymes migration; engine local en productos.
v0.1.x  — primera versión.
```

`v0.4.0` removió **breakingly** los packages
`decision/policy/risk/approval/kernel`. Si ves un import a alguno de esos
paths en tu repo, ese código sigue en una versión vieja: bumpealo a
`v0.4.x` y migrá a `governanceclient.SimulateRequest`/`SubmitRequest`.

## Reglas duras

- Esta carpeta **solo** contiene `governanceclient/` + `VERSION`.
- No agregues dominio (CEL evaluator, decision engine, risk scoring,
  approval logic, audit trail) acá. Esa lógica vive en
  `nexus/governance/internal/`. Si hace falta una primitiva nueva, exponé
  un endpoint en Nexus y un wrapper en este cliente.
- DTOs públicos van en `dto.go` con la misma forma que el handler de
  Nexus expone — no invertimos la dirección de la fuente de verdad.

## Lockdown

El script
[`companion/scripts/quality/check-governance-imports.sh`](../../../companion/scripts/quality/check-governance-imports.sh)
falla en CI si algún `.go` del repo importa los paths viejos
(`core/governance/go/{decision,policy,risk,approval,kernel}`). Replicá
ese check en cualquier repo nuevo que consuma governance.
