# core/errors/go

Tipos de error de dominio reusables y stable (v0.1.0). Permite a las capas
de transporte (HTTP, gRPC) mapear errores semánticos sin acoplar a status codes.

```go
import "github.com/devpablocristo/core/errors/go/domainerr"

return domainerr.NotFound("user", id)
return domainerr.Conflict("email already taken")
return domainerr.Validation("name", "required")
```

## Consumidores

companion, nexus, pymes
