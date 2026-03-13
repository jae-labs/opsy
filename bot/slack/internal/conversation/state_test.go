package conversation

import (
	"testing"
)

func TestStore_Create(t *testing.T) {
	s := NewStore()

	state := s.Create("ts1", "C123", "U123")
	if state.Phase != PhaseIdle {
		t.Errorf("got phase=%v, want PhaseIdle", state.Phase)
	}
	if state.ChannelID != "C123" {
		t.Errorf("got channel=%q, want C123", state.ChannelID)
	}
	if state.ThreadTS != "ts1" {
		t.Errorf("got threadTS=%q, want ts1", state.ThreadTS)
	}
	if state.UserID != "U123" {
		t.Errorf("got userID=%q, want U123", state.UserID)
	}
}

func TestStore_Get(t *testing.T) {
	s := NewStore()

	if got := s.Get("ts1"); got != nil {
		t.Error("expected nil for unknown thread")
	}

	created := s.Create("ts1", "C123", "U123")
	created.Phase = PhaseCategorySelected

	got := s.Get("ts1")
	if got == nil {
		t.Fatal("expected state for ts1")
	}
	if got.Phase != PhaseCategorySelected {
		t.Errorf("got phase=%v, want PhaseCategorySelected", got.Phase)
	}

	if s.Get("ts2") != nil {
		t.Error("expected nil for different thread")
	}
}

func TestStore_Delete(t *testing.T) {
	s := NewStore()

	state := s.Create("ts1", "C123", "U123")
	state.Phase = PhaseCategorySelected

	s.Delete("ts1")

	if s.Get("ts1") != nil {
		t.Error("expected nil after delete")
	}
}
