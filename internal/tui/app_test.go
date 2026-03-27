package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

	"api-tui/internal/app"
	"api-tui/internal/model"
	"api-tui/internal/spec"

	tea "github.com/charmbracelet/bubbletea"
)

type stubLoader struct {
	result *model.APISpec
	err    error
}

func (l *stubLoader) Load(ctx context.Context, source spec.Source) (*model.APISpec, error) {
	if l.err != nil {
		return nil, l.err
	}

	return l.result, nil
}

func TestModelInitLoadsSpecAsynchronously(t *testing.T) {
	t.Parallel()

	service := app.NewService(&stubLoader{
		result: &model.APISpec{
			Fingerprint: "spec-1",
			Title:       "Demo API",
			Operations: []model.Operation{
				{Key: model.NewOperationKey("GET", "/pets"), Method: "GET", Path: "/pets"},
			},
		},
	})

	m := NewModel(service, "demo.yaml")
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("expected init command")
	}
	if !m.viewState.LoadInFlight {
		t.Fatal("expected load to start immediately")
	}
	if m.viewState.ActiveLoadRequestID == 0 {
		t.Fatal("expected active load request id to be set")
	}

	msg := cmd()
	if _, ok := msg.(specLoadedMsg); !ok {
		t.Fatalf("expected specLoadedMsg, got %T", msg)
	}

	updatedModel, _ := m.Update(msg)
	updated := updatedModel.(*Model)
	if updated.session.Spec == nil {
		t.Fatal("expected loaded spec to be stored")
	}
	if updated.session.SelectedOperationKey != model.NewOperationKey("GET", "/pets") {
		t.Fatalf("expected first operation to be selected, got %q", updated.session.SelectedOperationKey)
	}
	if updated.viewState.LoadInFlight {
		t.Fatal("expected loading state to clear after success")
	}
}

func TestModelUpdatesFocusFromNumberKeys(t *testing.T) {
	t.Parallel()

	m := NewModel(app.NewService(&stubLoader{}), "demo.yaml")

	testCases := []struct {
		key  string
		want model.FocusedPane
	}{
		{key: "1", want: model.FocusedPaneOperations},
		{key: "2", want: model.FocusedPaneDetails},
		{key: "3", want: model.FocusedPaneRequest},
		{key: "4", want: model.FocusedPaneResponse},
	}

	for _, testCase := range testCases {
		updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(testCase.key)})
		updated := updatedModel.(*Model)
		if updated.viewState.FocusedPane != testCase.want {
			t.Fatalf("key %s expected focus %q, got %q", testCase.key, testCase.want, updated.viewState.FocusedPane)
		}
	}
}

func TestModelRotatesFocusWithTabAndShiftTab(t *testing.T) {
	t.Parallel()

	m := NewModel(app.NewService(&stubLoader{}), "demo.yaml")

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	updated := updatedModel.(*Model)
	if updated.viewState.FocusedPane != model.FocusedPaneDetails {
		t.Fatalf("expected tab to move focus to details, got %q", updated.viewState.FocusedPane)
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	updated = updatedModel.(*Model)
	if updated.viewState.FocusedPane != model.FocusedPaneOperations {
		t.Fatalf("expected shift-tab to move focus back to operations, got %q", updated.viewState.FocusedPane)
	}
}

func TestModelIgnoresStaleLoadResults(t *testing.T) {
	t.Parallel()

	m := NewModel(app.NewService(&stubLoader{}), "demo.yaml")
	m.viewState.LoadInFlight = true
	m.viewState.ActiveLoadRequestID = 2

	updatedModel, _ := m.Update(specLoadedMsg{
		requestID: 1,
		result: app.LoadResult{
			Session: model.SessionState{
				Spec: &model.APISpec{Title: "Should not apply"},
			},
		},
	})
	updated := updatedModel.(*Model)

	if updated.session.Spec != nil {
		t.Fatal("expected stale result to be ignored")
	}
	if !updated.viewState.LoadInFlight {
		t.Fatal("expected loading state to remain unchanged for stale result")
	}
}

func TestModelSelectsLayoutPresetFromWindowSize(t *testing.T) {
	t.Parallel()

	m := NewModel(app.NewService(&stubLoader{}), "demo.yaml")

	updatedModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	updated := updatedModel.(*Model)
	if updated.viewState.RightPaneLayoutPreset != layoutPresetWide {
		t.Fatalf("expected wide preset, got %q", updated.viewState.RightPaneLayoutPreset)
	}

	updatedModel, _ = updated.Update(tea.WindowSizeMsg{Width: 80, Height: 40})
	updated = updatedModel.(*Model)
	if updated.viewState.RightPaneLayoutPreset != layoutPresetNarrow {
		t.Fatalf("expected narrow preset, got %q", updated.viewState.RightPaneLayoutPreset)
	}
}

func TestModelRendersLoadFailureWithoutCrashing(t *testing.T) {
	t.Parallel()

	m := NewModel(app.NewService(&stubLoader{}), "broken.yaml")
	m.width = 120
	m.height = 30
	m.viewState.LoadInFlight = true
	m.viewState.ActiveLoadRequestID = 1

	updatedModel, _ := m.Update(specLoadedMsg{
		requestID: 1,
		err:       errors.New("unable to parse spec"),
	})
	updated := updatedModel.(*Model)
	view := updated.View()

	if !strings.Contains(view, "Failed to load spec.") {
		t.Fatalf("expected view to render load failure, got %q", view)
	}
	if !strings.Contains(view, "unable to parse spec") {
		t.Fatalf("expected load error to appear in view, got %q", view)
	}
	if !strings.Contains(view, "State: load failed") {
		t.Fatalf("expected status bar to show failed load state, got %q", view)
	}
	if !strings.Contains(view, "Keys: 1-4 switch Tab cycle q quit") {
		t.Fatalf("expected key hints in status bar, got %q", view)
	}
}
