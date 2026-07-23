# iam/go

Primitivas de identidad SaaS agnósticas del proveedor y del producto. El módulo
es inestable (`v0.x`) y separa la identidad externa de las decisiones de acceso
locales.

## Qué provee

- tipos para organizaciones, usuarios, membresías, invitaciones y eventos
  webhook;
- un perfil componible de migraciones PostgreSQL bajo el schema `iam`;
- un store `pgx` que funciona con `pgxpool.Pool`, `pgx.Conn` o `pgx.Tx`;
- deduplicación persistente para el inbox de webhooks.

```go
import (
	iam "github.com/devpablocristo/platform/iam/go"
	postgres "github.com/devpablocristo/platform/databases/postgres/go"
)

err := postgres.MigrateProfiles(ctx, db, iam.MigrationProfile())

store, err := iam.NewPostgresStore(pool)
```

Los stores no abren ni confirman transacciones. El caller puede entregar una
transacción para componer los cambios IAM con outbox u otras escrituras.

## Límites

- Los roles son valores opacos definidos por cada consumer.
- El módulo no crea usuarios, organizaciones ni sesiones en proveedores
  externos.
- No almacena tokens, tickets ni secretos.
- Las políticas RLS, permisos de producto y reglas como “un único owner”
  pertenecen al consumer.
- Los payloads webhook se conservan para procesamiento explícito; recibir el
  mismo `(provider, external_id)` es idempotente.
