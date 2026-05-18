package fsm

import "testing"

func TestTransition_ok(t *testing.T) {
	t.Parallel()
	type state string
	type event string
	m := New([]Rule[state, event]{
		{From: "a", Event: "go", To: "b"},
		{From: "b", Event: "go", To: "c"},
	})
	to, err := m.Transition("a", "go")
	if err != nil || to != "b" {
		t.Fatalf("got %q %v", to, err)
	}
	to, err = m.Transition("b", "go")
	if err != nil || to != "c" {
		t.Fatalf("got %q %v", to, err)
	}
}

func TestTransition_invalid(t *testing.T) {
	t.Parallel()
	m := New([]Rule[string, string]{
		{From: "idle", Event: "start", To: "running"},
	})
	_, err := m.Transition("idle", "stop")
	if err != ErrInvalidTransition {
		t.Fatalf("expected ErrInvalidTransition, got %v", err)
	}
}
