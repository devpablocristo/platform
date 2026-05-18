# core/http/go

Primitivas HTTP reutilizables para servicios Go del ecosistema. Estable (v0.1.x).

```go
import "github.com/devpablocristo/core/http/go/httpserver"
import "github.com/devpablocristo/core/http/go/httpclient"
```

## Subpaquetes

- `httpserver/` — bootstrap de `*http.Server` con hardening (CORS, timeouts, security headers) leíble desde env vía `SecurityConfigFromEnv`
- `httpclient/` — cliente HTTP con retries, timeouts, tracing y serialización JSON
- `httperr/` — mapeo dominio → status HTTP
- `httpjson/` — helpers para encode/decode JSON con errores tipados
- `health/` — handlers `/livez` `/readyz` estándar
- `pagination/` — cursor/offset pagination types

## Consumidores

companion, nexus, pymes
