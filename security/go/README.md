# core/security/go

Primitivas de seguridad cross-app: API keys, context keys para identidad, hashing utilitario. Estable (v0.1.0).

## Subpaquetes

- `apikey/` — parser + scope checks para `X-API-Key` headers; soporte multi-tenant
- `contextkeys/` — claves canónicas para inyectar `subject_id` / `org_id` / `request_id` en `context.Context`
- `hashutil/` — wrappers convenience sobre `crypto/sha256` para tokens y digests

```go
import "github.com/devpablocristo/core/security/go/apikey"
import "github.com/devpablocristo/core/security/go/contextkeys"
```

## Consumidores

companion, nexus, pymes
