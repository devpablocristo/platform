# money

Capacidad agnostica para representar importes monetarios exactos.

## Go

`money/go` provee:

- `Decimal` exacto basado en coeficiente entero y escala decimal.
- JSON de importes como strings decimales.
- `driver.Valuer` y `sql.Scanner` para columnas PostgreSQL `NUMERIC`.
- `Currency` ISO 4217 y `Money` con validacion de moneda.
- redondeo explicito mediante `RoundMode`.

No usa `float64` ni conversiones binarias para valores monetarios.
