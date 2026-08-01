# http/go

Primitivas HTTP reutilizables para servicios Go del ecosistema. Estable (v0.1.x).

```go
import "github.com/devpablocristo/platform/http/go/httpserver"
import "github.com/devpablocristo/platform/http/go/httpclient"
```

## Subpaquetes

- `httpserver/` — bootstrap de `*http.Server` con hardening (CORS, timeouts, security headers) leíble desde env vía `SecurityConfigFromEnv`
- `httpclient/` — cliente HTTP con retries, timeouts, tracing y serialización JSON
- `httperr/` — mapeo dominio → status HTTP
- `httpjson/` — helpers para encode/decode JSON con errores tipados
- `health/` — handlers `/livez` `/readyz` estándar
- `pagination/` — cursor/offset pagination types

## Notas de comportamiento

- `httpjson.WriteJSON` **serializa antes de escribir la cabecera** (v0.3.0). Al revés, un
  payload que no serializa dejaba el status ya emitido y el cuerpo truncado: un 200 con JSON
  inválido, imposible de corregir porque la cabecera no se reescribe. Ahora ese caso da 500.
  El salto de línea final se conserva, así que el cuerpo no cambia para nadie.
- `httperr.WriteFrom` **loguea el error interno que oculta** (v0.3.0). Un error no
  clasificado sale como `{"code":"INTERNAL"}` y sin log ese 500 no se puede debuggear. Los
  errores de dominio NO se loguean: son respuestas esperadas del negocio, y ensuciarían el
  log justo donde se buscan incidentes.

## Consumidores

companion, nexus, pymes, medmory
