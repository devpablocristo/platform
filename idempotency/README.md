# idempotency

Capacidad agnostica para hacer idempotentes comandos HTTP sobre PostgreSQL.

## Go

`idempotency/go` provee:

- un store PostgreSQL con claim atomico, lease, reclaim y expiracion;
- replay de status, headers y body para requests ya completados;
- conflicto cuando una key se reutiliza con otro fingerprint;
- middleware `net/http` con fingerprint SHA-256 y espera con backoff acotado;
- migracion SQL embebida y componible con el runner del consumidor.

El modulo usa un `scope` opaco para aislar espacios de keys sin conocer tenants,
organizaciones ni reglas de producto. El consumer debe derivar ese scope desde
contexto confiable; nunca desde un identificador arbitrario del request.

```go
store, err := idempotency.NewPostgresStore(pool, idempotency.DefaultStoreConfig())
if err != nil {
	return err
}

cfg := idempotency.DefaultMiddlewareConfig()
cfg.Scope = func(r *http.Request) (string, error) {
	return trustedScopeFromContext(r.Context())
}

middleware, err := idempotency.NewMiddleware(store, cfg)
if err != nil {
	return err
}

handler := middleware.Wrap(mux)
```

Aplicar `idempotency.Migrations` antes de construir el store:

```go
sql, err := idempotency.Migrations.ReadFile(idempotency.InitialMigration)
if err != nil {
	return err
}
_, err = pool.Exec(ctx, string(sql))
```

## Errores publicos

- `ErrKeyRequired` / `ErrInvalidKey`: key ausente o invalida.
- `ErrScopeRequired`: el resolver no produjo un namespace confiable.
- `ErrFingerprintMismatch`: la misma key no representa el mismo request.
- `ErrInProgress`: otro request conserva un lease vigente.
- `ErrLeaseLost`: el owner original ya no puede completar o abandonar.
- `ErrRequestTooLarge` / `ErrResponseTooLarge`: se excedio un limite del
  middleware.

`NewMiddleware` exige un `ScopeResolver` y falla cerrado si devuelve un scope
vacio. El store y el constraint PostgreSQL tambien rechazan scopes vacios. El
writer HTTP por defecto traduce errores a `400`, `409`, `413`, `500` o `503`.
`MiddlewareConfig.WriteError` permite integrar el contrato de errores de cada
producto.

## Garantias y limites

El claim y el replay son seguros ante concurrencia dentro de PostgreSQL. El
middleware bufferiza la respuesta y no soporta streaming, hijacking ni flush
incremental. El lease debe superar la duracion maxima esperada del handler.

Persistir la respuesta no vuelve atomicos efectos externos o escrituras hechas
fuera de la misma transaccion. Esos flujos deben combinar idempotencia con una
transaccion de negocio y outbox.

`utils/idempotency/go` es un modulo legacy distinto para leases de consumers de
colas sobre DynamoDB. No se importa ni se replica su API en este modulo.
