package usagemetering

import "context"

// MeteringPort define el adapter de incremento de contadores.
type MeteringPort interface {
	Increment(context.Context, string, string) error
}
