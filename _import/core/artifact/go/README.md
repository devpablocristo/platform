# core/artifact/go

**Estado: experimental — no consumir desde código productivo.**

Sin tag estable. La API puede cambiar sin previo aviso hasta que se publique
`artifact/go/v0.1.0`. Para proponer promoción a estable, abrir issue en el repo.

## Qué es

Manejo unificado de assets multi-formato (CSV / XLSX / PDF / JSON / TXT / PNG / JPG)
con normalización de filenames, content-types, y backends de storage opcionales.
Subpaquetes:

- `attachments/` — adjuntos genéricos
- `pdf/` · `qr/` · `tabular/` — generadores específicos por formato
- `storage/` — backend abstraction (filesystem / S3 / etc.)
