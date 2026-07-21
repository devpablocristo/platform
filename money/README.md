# money

Capacidad agnostica para representar importes monetarios exactos.

## Go

`money/go` provee:

- `Decimal` exacto basado en coeficiente entero y escala decimal.
- JSON de importes como strings decimales.
- `driver.Valuer` y `sql.Scanner` para columnas PostgreSQL `NUMERIC`; `NULL`
  se rechaza para evitar convertir ausencia de datos en cero.
- `Currency` valida membresia ISO 4217 mediante datos mantenidos por
  `golang.org/x/text/currency` y exige el codigo alpha-3 canonico en mayusculas.
- redondeo explicito mediante `RoundMode`; los modos desconocidos siempre se
  rechazan, incluso si la escala solicitada no cambia el valor.

No usa `float64` ni conversiones binarias para valores monetarios.
