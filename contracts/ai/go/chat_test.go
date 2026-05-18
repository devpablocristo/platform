package contracts

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestChatResponseRoundTrip(t *testing.T) {
	cid := uuid.New()
	original := ChatResponse{
		ChatID: cid,
		Reply:  "Hola",
		Blocks: []ChatBlock{{Type: "text", Text: "Hola"}},
	}
	raw, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got ChatResponse
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.ChatID != cid || got.Reply != "Hola" || len(got.Blocks) != 1 || got.Blocks[0].Type != "text" || got.Blocks[0].Text != "Hola" {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
}

func TestChatBlockTextOnlyMarshal(t *testing.T) {
	b := ChatBlock{Type: "text", Text: "x"}
	raw, err := json.Marshal(b)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// omitempty: solo type + text
	expected := `{"type":"text","text":"x"}`
	if string(raw) != expected {
		t.Fatalf("marshal mismatch: got %s want %s", raw, expected)
	}
}

func TestConversationSummaryMarshal(t *testing.T) {
	cs := ConversationSummary{
		ID:           uuid.New(),
		Title:        "t",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		MessageCount: 3,
	}
	if _, err := json.Marshal(cs); err != nil {
		t.Fatalf("marshal: %v", err)
	}
}
