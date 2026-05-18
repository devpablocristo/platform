package fsm

import (
	"fmt"
)

var (
	// ErrTerminal indica que el estado actual no admite cambios (salvo permanecer igual).
	ErrTerminal = fmt.Errorf("fsm: cannot leave terminal state")
)

// StringMachine es inmutable tras Build.
type StringMachine struct {
	terminals             map[string]struct{}
	freeGroup             map[string]struct{}
	explicit              map[string]map[string]struct{}
	allowAnyNonTerminalTo map[string]struct{}
}

// NewBuilder construye un Builder vacío.
func NewBuilder() *Builder {
	return &Builder{
		m: StringMachine{
			terminals:             make(map[string]struct{}),
			freeGroup:             make(map[string]struct{}),
			explicit:              make(map[string]map[string]struct{}),
			allowAnyNonTerminalTo: make(map[string]struct{}),
		},
	}
}

// Builder configura una StringMachine (patrón fluido).
type Builder struct {
	m StringMachine
}

// Terminal marca estados sin transiciones salientes (excepto from==to en Validate).
func (b *Builder) Terminal(states ...string) *Builder {
	for _, s := range states {
		if s == "" {
			continue
		}
		b.m.terminals[s] = struct{}{}
	}
	return b
}

// FreeTransitionsAmong permite cualquier transición entre pares de estados del conjunto
// (subgrafo completo). No incluye bucles implícitos más allá de lo que CanTransition resuelve.
func (b *Builder) FreeTransitionsAmong(states ...string) *Builder {
	for _, s := range states {
		if s == "" {
			continue
		}
		b.m.freeGroup[s] = struct{}{}
	}
	return b
}

// AllowFromStatesTo registra transiciones explícitas hacia `to` desde cada `from`.
func (b *Builder) AllowFromStatesTo(to string, froms ...string) *Builder {
	if to == "" {
		return b
	}
	for _, from := range froms {
		if from == "" {
			continue
		}
		if b.m.explicit[from] == nil {
			b.m.explicit[from] = make(map[string]struct{})
		}
		b.m.explicit[from][to] = struct{}{}
	}
	return b
}

// Allow registra una sola transición from→to.
func (b *Builder) Allow(from, to string) *Builder {
	return b.AllowFromStatesTo(to, from)
}

// AllowAnyTo permite from→to para cualquier `from` que no sea terminal.
// Útil para estados globales (p. ej. "cancelled"). No aplicar a destinos que requieran origen acotado.
func (b *Builder) AllowAnyTo(to string) *Builder {
	if to != "" {
		b.m.allowAnyNonTerminalTo[to] = struct{}{}
	}
	return b
}

// Build fija la máquina para consultas.
func (b *Builder) Build() *StringMachine {
	out := b.m
	return &out
}

// CanTransition informa si from→to está permitido (from y to ya normalizados por el caller).
func (m *StringMachine) CanTransition(from, to string) bool {
	if from == to {
		return true
	}
	if _, ok := m.terminals[from]; ok {
		return false
	}
	if _, ok := m.freeGroup[from]; ok {
		if _, ok2 := m.freeGroup[to]; ok2 {
			return true
		}
	}
	if targets, ok := m.explicit[from]; ok {
		if _, ok2 := targets[to]; ok2 {
			return true
		}
	}
	if _, ok := m.allowAnyNonTerminalTo[to]; ok {
		return true
	}
	return false
}

// Validate devuelve nil si la transición es válida (incluye from==to).
func (m *StringMachine) Validate(from, to string) error {
	if from == to {
		return nil
	}
	if _, ok := m.terminals[from]; ok {
		return fmt.Errorf("%w (%q)", ErrTerminal, from)
	}
	if !m.CanTransition(from, to) {
		return fmt.Errorf("%w (%q -> %q)", ErrInvalidTransition, from, to)
	}
	return nil
}
