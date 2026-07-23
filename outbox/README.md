# outbox

Outbox transaccional agnostico para PostgreSQL.

## Go

`outbox/go` provee:

- `Store.Append(ctx, pgx.Tx, MessageInput)` para escribir el mensaje dentro de
  la misma transaccion que el cambio de negocio;
- leasing atomico con `FOR UPDATE SKIP LOCKED` y tokens opacos;
- reintentos con backoff exponencial explicito y maximo de intentos por mensaje;
- transiciones `MarkPublished` y `MarkFailed` protegidas por el lease vigente;
- `Dispatcher` con `Publisher` inyectable y metadatos estables
  (`MessageID`/`IdempotencyKey`);
- reloj, jitter, IDs y tokens de lease inyectables para tests deterministas.

`MigrationProfile()` devuelve el perfil PostgreSQL componible para la tabla por
defecto. `SchemaSQL` conserva el DDL idempotente para consumers que necesiten un
`pgx.Identifier` calificado por schema.

```go
database, _ := postgres.Open(ctx, databaseURL)
defer database.Close()

err := postgres.MigrateProfiles(ctx, database, outbox.MigrationProfile())

store, _ := outbox.NewStore(database.Pool(), outbox.StoreConfig{
    DefaultMaxAttempts: 5,
})

tx, _ := pool.Begin(ctx)
_, err := store.Append(ctx, tx, outbox.MessageInput{
    Topic:          "entity.changed",
    IdempotencyKey: commandID,
    Payload:        payload,
})
if err == nil {
    err = tx.Commit(ctx)
}
```

Para una tabla custom, el consumer puede aplicar
`SchemaSQL(pgx.Identifier{"app", "outbox_messages"})` y pasar el mismo
identificador a `StoreConfig.Table`.

El payload es opaco (`BYTEA`) y los headers son `JSONB`; el modulo no conoce
tenants, productos, rutas ni reglas de negocio. Las politicas RLS y el contexto
tenant pertenecen al consumer. Si varias organizaciones comparten una tabla, el
consumer debe construir `IdempotencyKey` dentro de un namespace confiable para
evitar colisiones entre ellas.

Un `Publisher` debe propagar `Publication.IdempotencyKey` al transporte destino.
Si el publish tiene exito pero se pierde el lease antes de persistir
`published_at`, el downstream puede recibir el mismo `MessageID` nuevamente y
debe deduplicarlo. Por esa ventana inevitable, el dispatcher ofrece entrega
**at-least-once** y no promete exactly-once sin idempotencia en el destino.
