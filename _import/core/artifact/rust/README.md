# artifact/rust

Runtime Rust para las partes de `artifact` que sí ganan con un kernel reusable y determinista.

## Alcance

Este runtime agrega una implementación nueva sin reemplazar `artifact/go`.

Cobertura actual:

- `asset`: naming, content types y metadata reusable
- `attachments`: storage keys y links temporales
- `tabular`: CSV con BOM y XLSX de una sola hoja, más parsing CSV/XLSX

Queda fuera por ahora:

- `pdf`
- `qr`

Esas piezas siguen mejor resueltas hoy en `artifact/go` porque están más cerca de wrappers de librerías que de un kernel que gane mucho con Rust.

## Arquitectura

- `src/domain/`: modelos y reglas puras
- `src/application/`: casos de uso y puertos
- `src/infrastructure/`: adapters de reloj y codec XLSX

## Hardening

- `TabularService` acepta `TabularLimits` para imponer límites de tamaño, filas, columnas y bytes por celda
- el codec XLSX valida esos límites tanto al exportar como al parsear
- los errores distinguen validación tabular de errores operativos del codec

## Validación

```bash
cargo fmt --all
cargo clippy --all-targets -- -D warnings
cargo test
```
