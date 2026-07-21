# databases/postgres/go

Wrappers sobre `database/sql` + `gorm.io/gorm` para servicios Postgres del
ecosistema. Estable (v0.4.x).

## Qué provee

- `postgres.go` — apertura de `*sql.DB` con DSN parsing + pool tuning desde env
- `gorm.go` — bootstrap de `*gorm.DB` con logging integrado a slog
- `gorm_errors.go` — traducción de errores gorm/pg a `platform/errors/go`
- `migrate.go` / `gorm_migrate.go` — runner de migraciones embed-friendly compatible con `golang-migrate`
- `uow.go` — Unit of Work genérico con API explícita (`Do`/`TxContext`) y API
  contextual (`WithinTx`, `Tx[T]`, `AfterCommit`)
- `tenantctx/` — tenant fail-closed y GUC `app.org_id` local a la transacción para RLS

## Unit of Work contextual

`WithinTx` agrega la transacción sólo al contexto entregado al callback. `Tx[T]`
falla con `ErrNoActiveTransaction` fuera de ese scope y con
`ErrTransactionTypeMismatch` si el tipo solicitado no coincide. `AfterCommit`
registra callbacks que se ejecutan únicamente después de un commit exitoso.

```go
err := uow.WithinTx(ctx, func(ctx context.Context) error {
	tx, err := postgres.Tx[pgx.Tx](ctx)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, "UPDATE jobs SET claimed = true WHERE id = $1", jobID); err != nil {
		return err
	}
	return postgres.AfterCommit(ctx, notifyWorker)
})
```

La API previa `Do`/`TxContext` sigue disponible. Los errores y panics del
callback disparan rollback; un error posterior al commit se devuelve al caller
sin intentar revertir la transacción ya confirmada.

```go
import postgres "github.com/devpablocristo/platform/databases/postgres/go"
```

## Consumidores

companion, nexus, pymes
