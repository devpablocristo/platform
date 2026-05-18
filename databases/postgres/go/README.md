# core/databases/postgres/go

Wrappers sobre `database/sql` + `gorm.io/gorm` para servicios Postgres del
ecosistema. Estable (v0.1.x).

## Qué provee

- `postgres.go` — apertura de `*sql.DB` con DSN parsing + pool tuning desde env
- `gorm.go` — bootstrap de `*gorm.DB` con logging integrado a slog
- `gorm_errors.go` — traducción de errores gorm/pg a `core/errors/go`
- `migrate.go` / `gorm_migrate.go` — runner de migraciones embed-friendly compatible con `golang-migrate`

```go
import postgres "github.com/devpablocristo/core/databases/postgres/go"
```

## Consumidores

companion, nexus, pymes
