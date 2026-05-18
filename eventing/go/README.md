# core/eventing/go

**Estado: experimental — no consumir desde código productivo.**

Sin tag estable. La API puede cambiar sin previo aviso hasta que se publique
`eventing/go/v0.1.0`. Para proponer promoción a estable, abrir issue en el repo.

## Qué es

Envelope genérico tipado para eventos asíncronos. Provee `Metadata` estándar
(id, kind, source, tenant_id, version, sent_at, extra) y `Envelope[T]` con
payload tipado vía generics. El constructor `New` aplica defaults razonables
(UTC, version=1, timestamp si está vacío).
