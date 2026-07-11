// Package civil expone un type `Date` para fechas calendarizadas sin
// timezone: una fecha de nacimiento, de un estudio, de vencimiento o de
// agenda. Diferencia crítica con time.Time: una fecha calendarizada no tiene
// hora ni zona — el 2024-03-15 es el 2024-03-15 sin importar dónde se abra.
// Usar time.Time fuerza un timezone arbitrario y lleva a bugs off-by-one
// (UTC midnight → fecha local previa).
//
// Formato canónico: ISO 8601 calendar date "YYYY-MM-DD".
//
// Decisiones:
//   - Tolerante en parse: acepta RFC3339 ("2024-01-15T10:30:00Z") y descarta
//     la parte de hora, para no romper datos legacy.
//   - Estricto en formato: solo "YYYY-MM-DD" o RFC3339 reconocidos.
//   - JSON: string crudo. database/sql: implementa Valuer/Scanner (DATE/text).
//   - Empty/zero value: Date{} con String()=="" e IsZero()==true.
//
// Solo-stdlib a propósito (sin acople a un driver de DB en particular). El
// campo interno no se exporta: el único modo de construir un Date válido es
// `Parse`, cerrando la puerta a strings inválidos viajando como dominio.
// Consumidores DynamoDB (sin database/sql) mapean Date<->string en su capa de
// repositorio (String()/Parse()).
package civil

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"
)

// Date representa una fecha calendarizada YYYY-MM-DD sin timezone.
type Date struct {
	s string
}

// Parse acepta:
//   - "" → Date{} (zero value, IsZero true)
//   - "YYYY-MM-DD" → Date directo
//   - RFC3339 con T o " " separador → toma solo la parte fecha
//
// Cualquier otra cosa devuelve error wrappeado.
func Parse(s string) (Date, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Date{}, nil
	}
	// Tolerar RFC3339 legacy: "2024-01-15T10:30:00Z" → "2024-01-15".
	// Cuidado: requiere len(s) > 10 — un input "2024-01-15" exacto NO
	// debe tocarse (s[10] OOB).
	if len(s) > 10 && (s[10] == 'T' || s[10] == ' ') {
		s = s[:10]
	}
	if _, err := time.Parse("2006-01-02", s); err != nil {
		return Date{}, fmt.Errorf("invalid civil date %q: must be YYYY-MM-DD", s)
	}
	return Date{s: s}, nil
}

// MustParse es como Parse pero entra en panic ante error. Uso para literales
// de test/constantes conocidas, nunca sobre input externo.
func MustParse(s string) Date {
	d, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return d
}

// FromTime construye una Date tomando la parte calendario de t (en su
// location actual). No aplica conversión de zona.
func FromTime(t time.Time) Date {
	return Date{s: t.Format("2006-01-02")}
}

// Today devuelve la fecha de hoy en UTC.
func Today() Date {
	return Date{s: time.Now().UTC().Format("2006-01-02")}
}

// String formato canónico YYYY-MM-DD; "" si es zero value.
func (d Date) String() string { return d.s }

// IsZero indica si la fecha está vacía (no fue seteada).
func (d Date) IsZero() bool { return d.s == "" }

// Time devuelve la fecha como time.Time a medianoche UTC. Útil para aritmética;
// zero value devuelve el zero time.
func (d Date) Time() time.Time {
	if d.s == "" {
		return time.Time{}
	}
	t, _ := time.Parse("2006-01-02", d.s)
	return t
}

// Before: comparación lexicográfica (válida porque YYYY-MM-DD es lex-sortable).
// Si alguno es zero, devuelve false.
func (d Date) Before(other Date) bool {
	return d.s != "" && other.s != "" && d.s < other.s
}

// After: simétrico de Before. Si alguno es zero, devuelve false.
func (d Date) After(other Date) bool {
	return d.s != "" && other.s != "" && d.s > other.s
}

// Equal: dos Dates son iguales si su string canónico coincide.
func (d Date) Equal(other Date) bool { return d.s == other.s }

// MarshalJSON: serializa como string JSON. Zero value → "" para no romper
// clientes que esperan ausencia. Usar `omitempty` en DTOs si querés saltar.
func (d Date) MarshalJSON() ([]byte, error) {
	if d.IsZero() {
		return []byte(`""`), nil
	}
	return []byte(`"` + d.s + `"`), nil
}

// UnmarshalJSON acepta string JSON o null.
func (d *Date) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	if s == "null" || s == `""` {
		*d = Date{}
		return nil
	}
	if len(s) < 2 || s[0] != '"' || s[len(s)-1] != '"' {
		return fmt.Errorf("civil date: expected JSON string, got %s", s)
	}
	parsed, err := Parse(s[1 : len(s)-1])
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}

// Value implementa driver.Valuer: persiste como string "YYYY-MM-DD" (o NULL si
// zero). Compatible con columnas DATE o TEXT en Postgres.
func (d Date) Value() (driver.Value, error) {
	if d.IsZero() {
		return nil, nil
	}
	return d.s, nil
}

// Scan implementa sql.Scanner: acepta nil, string, []byte o time.Time (lo que
// devuelven los drivers para columnas DATE).
func (d *Date) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*d = Date{}
		return nil
	case string:
		parsed, err := Parse(v)
		if err != nil {
			return err
		}
		*d = parsed
		return nil
	case []byte:
		parsed, err := Parse(string(v))
		if err != nil {
			return err
		}
		*d = parsed
		return nil
	case time.Time:
		*d = FromTime(v)
		return nil
	default:
		return fmt.Errorf("civil date: cannot scan %T", src)
	}
}
