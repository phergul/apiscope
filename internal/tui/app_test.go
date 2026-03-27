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
	if updated.activeDetailsSection != detailsSectionSummary {
		t.Fatalf("expected summary details section after load, got %q", updated.activeDetailsSection)
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

func TestModelFilterModeUpdatesVisibleOperationsLive(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForNavigation()

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	updated := updatedModel.(*Model)
	if updated.viewState.ActiveEditorMode != model.EditorModeFilter {
		t.Fatalf("expected filter mode, got %q", updated.viewState.ActiveEditorMode)
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("admin")})
	updated = updatedModel.(*Model)
	if updated.viewState.FilterText != "admin" {
		t.Fatalf("expected filter text admin, got %q", updated.viewState.FilterText)
	}
	if len(updated.viewState.VisibleOperationKeys) != 1 {
		t.Fatalf("expected 1 visible operation, got %d", len(updated.viewState.VisibleOperationKeys))
	}
	if updated.session.SelectedOperationKey != model.NewOperationKey("POST", "/pets") {
		t.Fatalf("expected filtered selection to move to POST /pets, got %q", updated.session.SelectedOperationKey)
	}
}

func TestModelFilterModeBackspaceAndDeleteTrimCharacters(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForNavigation()
	m.viewState.ActiveEditorMode = model.EditorModeFilter
	m.viewState.FilterText = "pets"

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	updated := updatedModel.(*Model)
	if updated.viewState.FilterText != "pet" {
		t.Fatalf("expected backspace to trim one character, got %q", updated.viewState.FilterText)
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyDelete})
	updated = updatedModel.(*Model)
	if updated.viewState.FilterText != "pe" {
		t.Fatalf("expected delete to trim one character, got %q", updated.viewState.FilterText)
	}
}

func TestModelFilterModeExitsOnEnterAndEsc(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForNavigation()
	m.viewState.ActiveEditorMode = model.EditorModeFilter
	m.viewState.FilterText = "pets"

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := updatedModel.(*Model)
	if updated.viewState.ActiveEditorMode != model.EditorModeBrowse {
		t.Fatalf("expected enter to exit filter mode, got %q", updated.viewState.ActiveEditorMode)
	}

	updated.viewState.ActiveEditorMode = model.EditorModeFilter
	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated = updatedModel.(*Model)
	if updated.viewState.ActiveEditorMode != model.EditorModeBrowse {
		t.Fatalf("expected esc to exit filter mode, got %q", updated.viewState.ActiveEditorMode)
	}
	if updated.viewState.FilterText != "" {
		t.Fatalf("expected esc to clear filter text, got %q", updated.viewState.FilterText)
	}
}

func TestModelEscClearsFilterOutsideFilterMode(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForNavigation()
	m.viewState.FilterText = "pets"
	m.syncVisibleOperations()

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated := updatedModel.(*Model)
	if updated.viewState.FilterText != "" {
		t.Fatalf("expected esc to clear filter text, got %q", updated.viewState.FilterText)
	}
	if len(updated.viewState.VisibleOperationKeys) != 3 {
		t.Fatalf("expected visible operations to reset after clearing filter, got %d", len(updated.viewState.VisibleOperationKeys))
	}
}

func TestModelOperationsMovementUpdatesSelectionAndCursor(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForNavigation()
	m.viewState.FocusedPane = model.FocusedPaneOperations

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	updated := updatedModel.(*Model)
	if updated.session.SelectedOperationKey != model.NewOperationKey("POST", "/pets") {
		t.Fatalf("expected j to move to next operation, got %q", updated.session.SelectedOperationKey)
	}
	if updated.viewState.OperationsCursor != 1 {
		t.Fatalf("expected cursor 1, got %d", updated.viewState.OperationsCursor)
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEnd})
	updated = updatedModel.(*Model)
	if updated.viewState.OperationsCursor != 2 {
		t.Fatalf("expected end to jump to last operation, got %d", updated.viewState.OperationsCursor)
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyHome})
	updated = updatedModel.(*Model)
	if updated.viewState.OperationsCursor != 0 {
		t.Fatalf("expected home to jump to first operation, got %d", updated.viewState.OperationsCursor)
	}
}

func TestModelOperationsSectionJumpMovesBetweenGroups(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForNavigation()
	m.viewState.FocusedPane = model.FocusedPaneOperations

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	updated := updatedModel.(*Model)
	if updated.session.SelectedOperationKey != model.NewOperationKey("POST", "/pets") {
		t.Fatalf("expected ] to jump to next group, got %q", updated.session.SelectedOperationKey)
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}})
	updated = updatedModel.(*Model)
	if updated.session.SelectedOperationKey != model.NewOperationKey("GET", "/pets") {
		t.Fatalf("expected [ to jump to previous group, got %q", updated.session.SelectedOperationKey)
	}
}

func TestModelDetailsSectionNavigationSkipsUnavailableSections(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForNavigation()
	m.viewState.FocusedPane = model.FocusedPaneDetails

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	updated := updatedModel.(*Model)
	if updated.activeDetailsSection != detailsSectionParameters {
		t.Fatalf("expected ] to move to parameters, got %q", updated.activeDetailsSection)
	}

	updated.session.SelectedOperationKey = model.NewOperationKey("POST", "/pets")
	updated.syncActiveDetailsSection()
	if updated.activeDetailsSection != detailsSectionSummary {
		t.Fatalf("expected active section to fall back to summary, got %q", updated.activeDetailsSection)
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEnd})
	updated = updatedModel.(*Model)
	if updated.activeDetailsSection != detailsSectionSecurity {
		t.Fatalf("expected end to jump to last available details section, got %q", updated.activeDetailsSection)
	}
}

func TestModelFilterWithNoMatchesClearsSelection(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForNavigation()
	m.viewState.ActiveEditorMode = model.EditorModeFilter

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("zzz")})
	updated := updatedModel.(*Model)
	if len(updated.viewState.VisibleOperationKeys) != 0 {
		t.Fatalf("expected no visible operations, got %d", len(updated.viewState.VisibleOperationKeys))
	}
	if updated.session.SelectedOperationKey != "" {
		t.Fatalf("expected selection to clear, got %q", updated.session.SelectedOperationKey)
	}
}

func newLoadedModelForNavigation() *Model {
	spec := &model.APISpec{
		Operations: []model.Operation{
			{
				Key:         model.NewOperationKey("GET", "/pets"),
				Method:      "GET",
				Path:        "/pets",
				Summary:     "List pets",
				Description: "Returns pets.",
				Tags:        []string{"pets"},
				Parameters: []model.Parameter{
					{Name: "petId", In: model.ParameterLocationPath, Required: true, Schema: &model.Schema{Type: "string"}},
				},
				RequestBody: &model.RequestBodySpec{
					Required: true,
					Content:  []model.MediaTypeSpec{{MediaType: "application/json"}},
				},
				Responses: []model.ResponseSpec{
					{StatusCode: "200", Description: "OK", Content: []model.MediaTypeSpec{{MediaType: "application/json"}}},
				},
				Security: &model.SecurityRequirement{
					Alternatives: []model.SecurityAlternative{
						{Schemes: []model.SecurityRequirementRef{{Name: "api_key"}}},
					},
				},
			},
			{
				Key:        model.NewOperationKey("POST", "/pets"),
				Method:     "POST",
				Path:       "/pets",
				Summary:    "Create pet",
				Tags:       []string{"admin"},
				Deprecated: true,
			},
			{
				Key:     model.NewOperationKey("GET", "/health"),
				Method:  "GET",
				Path:    "/health",
				Summary: "Health",
			},
		},
		Security: &model.SecurityRequirement{
			Alternatives: []model.SecurityAlternative{
				{Schemes: []model.SecurityRequirementRef{{Name: "global_auth"}}},
			},
		},
	}

	return &Model{
		activeDetailsSection: detailsSectionSummary,
		session: model.SessionState{
			Spec:                 spec,
			SelectedOperationKey: model.NewOperationKey("GET", "/pets"),
		},
		viewState: model.ViewState{
			FocusedPane: model.FocusedPaneOperations,
			VisibleOperationKeys: []model.OperationKey{
				model.NewOperationKey("GET", "/pets"),
				model.NewOperationKey("POST", "/pets"),
				model.NewOperationKey("GET", "/health"),
			},
			OperationsCursor: 0,
			ActiveEditorMode: model.EditorModeBrowse,
		},
	}
}
