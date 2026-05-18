// Package fsm ofrece máquinas de estados finitas: una genérica tipada (reglas From+Event→To)
// y un builder string-based con terminales, grupos libres y aristas explícitas.
package fsm

import "errors"

// ErrInvalidTransition indica que no hay transición definida para (from, event).
var ErrInvalidTransition = errors.New("fsm: invalid transition")

// Rule describe una transición permitida.
type Rule[S comparable, E comparable] struct {
	From  S
	Event E
	To    S
}

// Machine tabla de transiciones inmutable.
type Machine[S comparable, E comparable] struct {
	transitions map[key[S, E]]S
}

type key[S, E comparable] struct {
	from S
	ev   E
}

// New construye la máquina a partir de reglas; la última regla gana si hay duplicado (from, event).
func New[S comparable, E comparable](rules []Rule[S, E]) *Machine[S, E] {
	m := make(map[key[S, E]]S, len(rules))
	for _, r := range rules {
		m[key[S, E]{from: r.From, ev: r.Event}] = r.To
	}
	return &Machine[S, E]{transitions: m}
}

// Transition devuelve el estado destino o ErrInvalidTransition.
func (m *Machine[S, E]) Transition(from S, event E) (S, error) {
	to, ok := m.transitions[key[S, E]{from: from, ev: event}]
	if !ok {
		var zero S
		return zero, ErrInvalidTransition
	}
	return to, nil
}
