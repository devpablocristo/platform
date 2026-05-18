package fsm

import (
	"errors"
	"testing"
)

func TestSameStateAlwaysAllowed(t *testing.T) {
	t.Parallel()
	m := NewBuilder().Terminal("done").Build()
	if err := m.Validate("done", "done"); err != nil {
		t.Fatalf("Validate same: %v", err)
	}
	if !m.CanTransition("done", "done") {
		t.Fatal("CanTransition same")
	}
}

func TestTerminalBlocksLeave(t *testing.T) {
	t.Parallel()
	m := NewBuilder().Terminal("done").Build()
	err := m.Validate("done", "open")
	if !errors.Is(err, ErrTerminal) {
		t.Fatalf("want ErrTerminal, got %v", err)
	}
}

func TestFreeGroup(t *testing.T) {
	t.Parallel()
	m := NewBuilder().FreeTransitionsAmong("a", "b", "c").Build()
	if !m.CanTransition("a", "b") || !m.CanTransition("b", "c") {
		t.Fatal("free group transitions")
	}
	if m.CanTransition("a", "x") {
		t.Fatal("should not allow a->x")
	}
}

func TestExplicitAndAllowAnyTo(t *testing.T) {
	t.Parallel()
	m := NewBuilder().
		Terminal("closed").
		FreeTransitionsAmong("draft", "review").
		AllowFromStatesTo("published", "review").
		AllowAnyTo("cancelled").
		Build()

	if err := m.Validate("draft", "cancelled"); err != nil {
		t.Fatalf("draft->cancelled: %v", err)
	}
	if err := m.Validate("review", "published"); err != nil {
		t.Fatalf("review->published: %v", err)
	}
	if m.CanTransition("draft", "published") {
		t.Fatal("draft should not reach published without review")
	}
}

func TestOrderExplicitBeforeAllowAnyToDoesNotWidenInvoiced(t *testing.T) {
	t.Parallel()
	m := NewBuilder().
		FreeTransitionsAmong("a", "b").
		AllowFromStatesTo("z", "b").
		AllowAnyTo("cancelled").
		Build()

	if m.CanTransition("a", "z") {
		t.Fatal("a->z should be denied")
	}
	if err := m.Validate("a", "cancelled"); err != nil {
		t.Fatalf("a->cancelled: %v", err)
	}
}
