# session

Eco HTTP del **Principal** autenticado (kernel), sin lógica de producto.

## Qué hace

- `GET /session` (detrás del mismo `authMW` que el resto del mux SaaS): responde **200** con el cuerpo JSON del `kerneldomain.Principal` (`tenant_id`, `actor`, `role`, `scopes`, `auth_method`).
- Sin campos de UI ni mapeos tipo `product_role`: eso vive en la app (p. ej. Pymes en `wire/saas_http.go` como `handleSessionEnriched`).

## Uso

```go
import "github.com/devpablocristo/core/saas/go/session"

session.RegisterProtected(mux, authMW)
```

## Frontera

- **Pertenece acá**: serializar el principal ya resuelto por `middleware`.
- **No pertenece**: copy de producto, onboarding, pantallas.
