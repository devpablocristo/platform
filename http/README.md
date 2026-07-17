# http

Helpers reutilizables para transporte HTTP frontend.

Implementación actual: `http/ts/`

## Pertenece

- `fetch` JSON reusable
- parseo uniforme de errores HTTP
- `normalizeHttpError` para errores fetch, Axios-like y envelopes extensibles
- helpers para `event-stream` JSON

La normalización es agnóstica al producto. Un consumer puede aportar un
`bodyAdapter` para extraer campos de su envelope sin agregar DTOs ni reglas de
negocio a `platform`.

## No pertenece

- storage de tokens
- providers de auth
- schemas o DTOs de producto
