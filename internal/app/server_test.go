package app

import (
	"testing"

	"github.com/phergul/apiscope/internal/model"
)

func TestSetSelectedServerUpdatesSessionAndExistingDrafts(t *testing.T) {
	t.Parallel()

	key := model.NewDraftKey("spec-123", model.NewOperationKey("GET", "/pets"))
	session := model.SessionState{
		SelectedServerURL: "https://old.example.com",
		RequestDrafts: map[model.DraftKey]*model.RequestDraft{
			key: {
				Key:             key,
				SpecFingerprint: "spec-123",
				OperationKey:    model.NewOperationKey("GET", "/pets"),
				ServerURL:       "https://old.example.com",
			},
		},
	}

	changed := SetSelectedServer(&session, "https://new.example.com")
	if !changed {
		t.Fatal("expected selected server change to succeed")
	}
	if session.SelectedServerURL != "https://new.example.com" {
		t.Fatalf("expected selected server url to update, got %q", session.SelectedServerURL)
	}
	if got := session.RequestDrafts[key].ServerURL; got != "https://new.example.com" {
		t.Fatalf("expected draft server url to stay in sync, got %q", got)
	}
}

func TestCycleSelectedServerAdvancesThroughProvidedServers(t *testing.T) {
	t.Parallel()

	session := model.SessionState{
		SelectedServerURL: "https://api.example.com",
	}

	ok := CycleSelectedServer(&session, []model.Server{
		{URL: "https://api.example.com"},
		{URL: "https://staging.example.com"},
	})
	if !ok {
		t.Fatal("expected server cycle to succeed")
	}
	if session.SelectedServerURL != "https://staging.example.com" {
		t.Fatalf("expected selected server to advance, got %q", session.SelectedServerURL)
	}
}

func TestCycleSelectedServerFallsBackToFirstWhenSelectionIsMissing(t *testing.T) {
	t.Parallel()

	session := model.SessionState{
		SelectedServerURL: "https://missing.example.com",
	}

	ok := CycleSelectedServer(&session, []model.Server{
		{URL: "https://api.example.com"},
		{URL: "https://staging.example.com"},
	})
	if !ok {
		t.Fatal("expected server cycle to recover missing selection")
	}
	if session.SelectedServerURL != "https://api.example.com" {
		t.Fatalf("expected selected server to fall back to first option, got %q", session.SelectedServerURL)
	}
}
