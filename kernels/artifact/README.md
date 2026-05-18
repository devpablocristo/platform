# artifact

Capacidad reusable para generación y manejo de artefactos.

## Pertenece

- PDF
- Excel
- CSV
- QR
- report generation
- naming de archivos
- metadata de artefactos
- storage contracts de artefactos

## No pertenece

- plantillas totalmente acopladas a un solo producto
- reportes que solo existen dentro de una app y no abstraen una capacidad reusable

## Fuentes iniciales esperadas

- `/home/pablo/Projects/Pablo/pymes/pymes-core/backend/internal/pdfgen`
- `/home/pablo/Projects/Pablo/pymes/pymes-core/backend/internal/dataio`
- `/home/pablo/Projects/Pablo/pymes/pymes-core/backend/internal/paymentgateway`
- `/home/pablo/Projects/Pablo/ponti/ponti-backend/internal/labor/excel`
- `/home/pablo/Projects/AlphaCoding/kma-backend/lambdas/shared/excel`
- `/home/pablo/Projects/AlphaCoding/kma-backend/lambdas/reports-worker/internal/generate-report`

## Nota

`artifact` es buen candidato a módulo multi-lenguaje en el futuro si aparecen implementaciones equivalentes en más de un runtime.

## Bootstrap inicial creado

Implementaciones actuales:

- `artifact/go/`
- `artifact/rust/`

Paquetes ya iniciados en esta implementación:

- `artifact` root package con `Asset`, `Format`, naming y content types
- `tabular/`
- `pdf/`
- `qr/`
- `storage/`
- `attachments/`

El módulo ya cubre `Asset` común, naming/metadata, contratos de storage, export tabular CSV/XLSX, generación PDF simple, QR PNG reusable y metadata reusable de attachments.

## Alcance de Rust

`artifact/rust/` cubre:

- `asset`
- `attachments`
- `tabular`

`pdf` y `qr` siguen en `artifact/go/` por ahora.
