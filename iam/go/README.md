# iam/go

Primitivas de identidad SaaS agnósticas del proveedor y del producto. El módulo
es inestable (`v0.2.x`) y separa la identidad externa de las decisiones de
acceso locales.

## Qué provee

- tipos para organizaciones, usuarios, membresías, invitaciones y eventos
  webhook;
- un perfil componible de migraciones PostgreSQL bajo el schema `iam`;
- un store `pgx` que funciona con `pgxpool.Pool`, `pgx.Conn` o `pgx.Tx`;
- deduplicación persistente para el inbox de webhooks;
- resolución fail-closed de membresías y transacciones con contexto RLS.

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

## Sesión y contexto transaccional

`VerifiedSession` es la salida normalizada de un verificador externo; no
verifica firmas por sí sola. `SessionTransactor` exige una sesión vigente,
resuelve una membresía local activa dentro de la transacción y recién entonces
aplica `app.user_id` y `app.org_id` con alcance local.

```go
transactor, err := iam.NewSessionTransactor(
	pool,
	iam.NewPostgresMembershipResolver(),
	iam.SessionTransactorConfig{},
)

err = transactor.WithinSessionTx(ctx, verified, func(
	ctx context.Context,
	tx pgx.Tx,
	active iam.ActiveMembership,
) error {
	// El mismo ctx conserva Unit of Work y AfterCommit.
	return nil
})
```

`PostgresMembershipResolver` consulta las tablas IAM directamente. Cuando el
consumer protege también el bootstrap con RLS, debe implementar
`MembershipResolver` mediante una función `SECURITY DEFINER` mínima y entregar
esa implementación al transactor.

## Límites

- Los roles son valores opacos definidos por cada consumer.
- El módulo no crea usuarios, organizaciones ni sesiones en proveedores
  externos.
- No almacena tokens, tickets ni secretos.
- Las políticas RLS, permisos de producto y reglas como “un único owner”
  pertenecen al consumer.
- Los roles/permisos del proveedor viajan como claims opacos; el consumer
  calcula su intersección con el rol local.
- Los payloads webhook se conservan para procesamiento explícito; recibir el
  mismo `(provider, external_id)` es idempotente.
