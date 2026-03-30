package tui

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/phergul/apiscope/internal/app"
	"github.com/phergul/apiscope/internal/model"
	"github.com/phergul/apiscope/internal/spec"
	detailsui "github.com/phergul/apiscope/internal/tui/details"

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
	if updated.viewState.ExpandedRightPane != model.FocusedPaneRequest {
		t.Fatalf("expected request pane to start expanded, got %q", updated.viewState.ExpandedRightPane)
	}
	if updated.viewState.ZoomedPane {
		t.Fatal("expected zoom mode to start disabled")
	}
	if updated.activeDetailsSection != detailsui.SectionSummary {
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

func TestModelRightPaneExpansionTracksFocusChanges(t *testing.T) {
	t.Parallel()

	m := NewModel(app.NewService(&stubLoader{}), "demo.yaml")
	if m.viewState.ExpandedRightPane != model.FocusedPaneRequest {
		t.Fatalf("expected request pane to be expanded by default, got %q", m.viewState.ExpandedRightPane)
	}

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4'}})
	updated := updatedModel.(*Model)
	if updated.viewState.ExpandedRightPane != model.FocusedPaneResponse {
		t.Fatalf("expected response pane to expand when focused, got %q", updated.viewState.ExpandedRightPane)
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	updated = updatedModel.(*Model)
	if updated.viewState.ExpandedRightPane != model.FocusedPaneResponse {
		t.Fatalf("expected details focus to preserve right pane emphasis, got %q", updated.viewState.ExpandedRightPane)
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	updated = updatedModel.(*Model)
	if updated.viewState.ExpandedRightPane != model.FocusedPaneRequest {
		t.Fatalf("expected request pane to expand when focused, got %q", updated.viewState.ExpandedRightPane)
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
	if updated.viewState.ExpandedRightPane != model.FocusedPaneRequest {
		t.Fatalf("expected details focus to preserve request emphasis, got %q", updated.viewState.ExpandedRightPane)
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	updated = updatedModel.(*Model)
	if updated.viewState.FocusedPane != model.FocusedPaneOperations {
		t.Fatalf("expected shift-tab to move focus back to operations, got %q", updated.viewState.FocusedPane)
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyTab})
	updated = updatedModel.(*Model)
	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyTab})
	updated = updatedModel.(*Model)
	if updated.viewState.FocusedPane != model.FocusedPaneRequest {
		t.Fatalf("expected tab to move focus to request, got %q", updated.viewState.FocusedPane)
	}
	if updated.viewState.ExpandedRightPane != model.FocusedPaneRequest {
		t.Fatalf("expected request focus to expand request pane, got %q", updated.viewState.ExpandedRightPane)
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyTab})
	updated = updatedModel.(*Model)
	if updated.viewState.FocusedPane != model.FocusedPaneResponse {
		t.Fatalf("expected tab to move focus to response, got %q", updated.viewState.FocusedPane)
	}
	if updated.viewState.ExpandedRightPane != model.FocusedPaneResponse {
		t.Fatalf("expected response focus to expand response pane, got %q", updated.viewState.ExpandedRightPane)
	}
}

func TestModelZoomToggleAndFocusChangesFollowWhileZoomed(t *testing.T) {
	t.Parallel()

	m := NewModel(app.NewService(&stubLoader{}), "demo.yaml")

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	updated := updatedModel.(*Model)
	if !updated.viewState.ZoomedPane {
		t.Fatal("expected z to enable zoom mode")
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4'}})
	updated = updatedModel.(*Model)
	if !updated.viewState.ZoomedPane {
		t.Fatal("expected zoom mode to remain enabled after focus change")
	}
	if updated.viewState.FocusedPane != model.FocusedPaneResponse {
		t.Fatalf("expected focus to move to response, got %q", updated.viewState.FocusedPane)
	}
	if updated.viewState.ExpandedRightPane != model.FocusedPaneResponse {
		t.Fatalf("expected response emphasis to update while zoomed, got %q", updated.viewState.ExpandedRightPane)
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	updated = updatedModel.(*Model)
	if updated.viewState.ZoomedPane {
		t.Fatal("expected z to disable zoom mode")
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
	view = stripANSI(view)

	if !strings.Contains(view, "Failed to load spec") {
		t.Fatalf("expected view to render load failure, got %q", view)
	}
	if !strings.Contains(view, "unable to parse spec") {
		t.Fatalf("expected load error to appear in view, got %q", view)
	}
	if !strings.Contains(view, "[ Quit ]") {
		t.Fatalf("expected blocking load popup to show quit action, got %q", view)
	}
	if strings.Contains(view, "1 Operations") {
		t.Fatalf("expected blocking load popup instead of pane layout, got %q", view)
	}
}

func TestModelBlockingLoadErrorOnlyAllowsQuitKeys(t *testing.T) {
	t.Parallel()

	m := NewModel(app.NewService(&stubLoader{}), "broken.yaml")
	m.loadErr = errors.New("boom")

	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	updated := updatedModel.(*Model)
	if cmd != nil {
		t.Fatal("expected non-quit key to be ignored while blocking load error is shown")
	}
	if updated.viewState.FocusedPane != model.FocusedPaneOperations {
		t.Fatalf("expected focus to remain unchanged, got %q", updated.viewState.FocusedPane)
	}

	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected enter to quit while blocking load error is shown")
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

func TestModelFilterStillMatchesOperationSummaryWhenSummaryIsNotRendered(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForNavigation()

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	updated := updatedModel.(*Model)
	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("list")})
	updated = updatedModel.(*Model)

	if len(updated.viewState.VisibleOperationKeys) != 1 {
		t.Fatalf("expected 1 visible operation after summary filter, got %d", len(updated.viewState.VisibleOperationKeys))
	}
	if updated.session.SelectedOperationKey != model.NewOperationKey("GET", "/pets") {
		t.Fatalf("expected summary filter to match GET /pets, got %q", updated.session.SelectedOperationKey)
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

func TestModelOperationsMovementFollowsRenderedGroupedOrder(t *testing.T) {
	t.Parallel()

	m := &Model{
		session: model.SessionState{
			Spec: &model.APISpec{
				Operations: []model.Operation{
					{Key: model.NewOperationKey("GET", "/albums"), Method: "GET", Path: "/albums", Tags: []string{"albums"}},
					{Key: model.NewOperationKey("GET", "/artists"), Method: "GET", Path: "/artists", Tags: []string{"artists"}},
					{Key: model.NewOperationKey("GET", "/me/albums"), Method: "GET", Path: "/me/albums", Tags: []string{"albums"}},
				},
			},
			SelectedOperationKey: model.NewOperationKey("GET", "/albums"),
		},
		viewState: model.ViewState{
			FocusedPane: model.FocusedPaneOperations,
		},
		activeDetailsSection: detailsui.SectionSummary,
	}
	m.syncVisibleOperations()

	if got := m.viewState.VisibleOperationKeys; len(got) != 3 ||
		got[0] != model.NewOperationKey("GET", "/albums") ||
		got[1] != model.NewOperationKey("GET", "/me/albums") ||
		got[2] != model.NewOperationKey("GET", "/artists") {
		t.Fatalf("expected grouped visible order, got %#v", got)
	}

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	updated := updatedModel.(*Model)
	if updated.session.SelectedOperationKey != model.NewOperationKey("GET", "/me/albums") {
		t.Fatalf("expected cursor to move to grouped sibling /me/albums, got %q", updated.session.SelectedOperationKey)
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
	if updated.activeDetailsSection != detailsui.SectionSecurity {
		t.Fatalf("expected ] to move to security, got %q", updated.activeDetailsSection)
	}

	updated.session.SelectedOperationKey = model.NewOperationKey("POST", "/pets")
	updated.syncActiveDetailsSection()
	if updated.activeDetailsSection != detailsui.SectionSecurity {
		t.Fatalf("expected active section to stay on security when still available, got %q", updated.activeDetailsSection)
	}

	updated.session.Spec.Security = nil
	updated.session.SelectedOperationKey = model.NewOperationKey("GET", "/health")
	updated.syncActiveDetailsSection()
	if updated.activeDetailsSection != detailsui.SectionSummary {
		t.Fatalf("expected active section to fall back to summary when security is unavailable, got %q", updated.activeDetailsSection)
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEnd})
	updated = updatedModel.(*Model)
	if updated.activeDetailsSection != detailsui.SectionSummary {
		t.Fatalf("expected end to jump to last available details section, got %q", updated.activeDetailsSection)
	}
}

func TestModelRequestSectionNavigationMovesAcrossParameterBodyAndAuth(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForNavigation()
	m.viewState.FocusedPane = model.FocusedPaneRequest
	m.activeRequestSection = "Path"

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	updated := updatedModel.(*Model)
	if updated.activeRequestSection != "Body" {
		t.Fatalf("expected ] to move to request body, got %q", updated.activeRequestSection)
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	updated = updatedModel.(*Model)
	if updated.activeRequestSection != "Auth" {
		t.Fatalf("expected ] to move to auth, got %q", updated.activeRequestSection)
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}})
	updated = updatedModel.(*Model)
	if updated.activeRequestSection != "Body" {
		t.Fatalf("expected [ to move back to body, got %q", updated.activeRequestSection)
	}
}

func TestModelRequestRowNavigationMovesWithinActiveSection(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForNavigation()
	m.viewState.FocusedPane = model.FocusedPaneRequest
	m.activeRequestSection = "Body"

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnd})
	updated := updatedModel.(*Model)
	if updated.viewState.RequestActiveRow != 1 {
		t.Fatalf("expected end to jump to last request row, got %d", updated.viewState.RequestActiveRow)
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyHome})
	updated = updatedModel.(*Model)
	if updated.viewState.RequestActiveRow != 0 {
		t.Fatalf("expected home to jump to first request row, got %d", updated.viewState.RequestActiveRow)
	}
}

func TestModelRequestAndResponseSectionsResetToFirstOnOperationChange(t *testing.T) {
	t.Parallel()

	spec := &model.APISpec{
		Operations: []model.Operation{
			{
				Key:    model.NewOperationKey("GET", "/first"),
				Method: "GET",
				Path:   "/first",
				Parameters: []model.Parameter{
					{Name: "id", In: model.ParameterLocationPath, Required: true, Schema: &model.Schema{Type: "string"}},
					{Name: "market", In: model.ParameterLocationQuery, Schema: &model.Schema{Type: "string"}},
				},
				Responses: []model.ResponseSpec{
					{StatusCode: "200", Description: "OK"},
					{StatusCode: "default", Description: "Fallback"},
				},
			},
			{
				Key:    model.NewOperationKey("GET", "/second"),
				Method: "GET",
				Path:   "/second",
				Parameters: []model.Parameter{
					{Name: "owner", In: model.ParameterLocationPath, Required: true, Schema: &model.Schema{Type: "string"}},
					{Name: "region", In: model.ParameterLocationQuery, Schema: &model.Schema{Type: "string"}},
				},
				Responses: []model.ResponseSpec{
					{StatusCode: "200", Description: "OK"},
					{StatusCode: "default", Description: "Fallback"},
				},
			},
		},
	}

	m := &Model{
		session: model.SessionState{
			Spec:                 spec,
			SelectedOperationKey: model.NewOperationKey("GET", "/first"),
		},
		viewState: model.ViewState{
			FocusedPane: model.FocusedPaneOperations,
			VisibleOperationKeys: []model.OperationKey{
				model.NewOperationKey("GET", "/first"),
				model.NewOperationKey("GET", "/second"),
			},
			RequestActiveRow:    1,
			RequestScrollOffset: 1,
		},
		activeDetailsSection:  detailsui.SectionSummary,
		activeRequestSection:  "Query",
		activeResponseSection: "default",
	}

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	updated := updatedModel.(*Model)
	if updated.session.SelectedOperationKey != model.NewOperationKey("GET", "/second") {
		t.Fatalf("expected second operation to be selected, got %q", updated.session.SelectedOperationKey)
	}
	if updated.activeRequestSection != "Path" {
		t.Fatalf("expected request section to reset to first available, got %q", updated.activeRequestSection)
	}
	if updated.viewState.RequestActiveRow != 0 {
		t.Fatalf("expected request row to reset on operation change, got %d", updated.viewState.RequestActiveRow)
	}
	if updated.viewState.RequestScrollOffset != 0 {
		t.Fatalf("expected request scroll offset to reset on operation change, got %d", updated.viewState.RequestScrollOffset)
	}
	if updated.activeResponseSection != "Live" {
		t.Fatalf("expected response section to reset to live, got %q", updated.activeResponseSection)
	}
}

func TestModelResponseSectionNavigationAndFallback(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForNavigation()
	m.viewState.FocusedPane = model.FocusedPaneResponse
	m.activeResponseSection = "200"

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	updated := updatedModel.(*Model)
	if updated.activeResponseSection != "default" {
		t.Fatalf("expected ] to move to default response, got %q", updated.activeResponseSection)
	}

	updated.session.SelectedOperationKey = model.NewOperationKey("POST", "/pets")
	updated.syncActivePaneSections()
	if updated.activeResponseSection != "Live" {
		t.Fatalf("expected response section to fall back to live, got %q", updated.activeResponseSection)
	}
}

func TestModelCtrlRShowsInlineValidationWithoutExecuting(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForNavigation()
	m.viewState.FocusedPane = model.FocusedPaneRequest

	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	updated := updatedModel.(*Model)
	if cmd != nil {
		t.Fatal("expected ctrl+r to skip execution when validation fails")
	}
	if updated.viewState.FocusedPane != model.FocusedPaneRequest {
		t.Fatalf("expected focus to stay in request pane, got %q", updated.viewState.FocusedPane)
	}
	if !updated.requestValidation.HasIssues() {
		t.Fatal("expected validation issues to be stored")
	}

	data := updated.projectRequestPane()
	if len(data.ValidationNotice) == 0 {
		t.Fatal("expected validation summary to be projected into the request pane")
	}
	if data.Rows[0].Error == "" {
		t.Fatal("expected first request row to show an inline validation error")
	}
}

func TestModelCtrlRExecutesAndSelectsLiveResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got != "/pets" {
			t.Fatalf("expected request path /pets, got %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("expected content type application/json, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	m := newLoadedModelForNavigation()
	m.service = app.NewService(nil)
	m.viewState.FocusedPane = model.FocusedPaneRequest
	m.session.SelectedServerURL = server.URL
	draft := m.ensureSelectedRequestDraft()
	draft.PathParams["petId"] = "abc"
	draft.BodyRaw = `{"name":"fido"}`
	draft.BodyMediaType = "application/json"

	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	updated := updatedModel.(*Model)
	if cmd == nil {
		t.Fatal("expected ctrl+r to start execution")
	}
	if !updated.viewState.ExecuteInFlight {
		t.Fatal("expected execute-in-flight to be set")
	}

	msg := cmd()
	updatedModel, _ = updated.Update(msg)
	updated = updatedModel.(*Model)
	if updated.viewState.ExecuteInFlight {
		t.Fatal("expected execute-in-flight to clear after the result arrives")
	}
	if updated.viewState.FocusedPane != model.FocusedPaneResponse {
		t.Fatalf("expected focus to move to response, got %q", updated.viewState.FocusedPane)
	}
	if updated.activeResponseSection != "Live" {
		t.Fatalf("expected live response section to be selected, got %q", updated.activeResponseSection)
	}
	if updated.session.LastResponse == nil {
		t.Fatal("expected last response to be stored")
	}
	if updated.session.LastResponse.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP 200 response, got %d", updated.session.LastResponse.StatusCode)
	}
}

func TestModelIgnoresStaleExecuteResults(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForNavigation()
	m.viewState.ExecuteInFlight = true
	m.viewState.ActiveExecuteRequestID = 2

	updatedModel, _ := m.Update(executeFinishedMsg{
		requestID: 1,
		result: app.ExecuteResult{
			OperationKey: model.NewOperationKey("GET", "/pets"),
			Response: &model.HTTPResponse{
				OperationKey: model.NewOperationKey("GET", "/pets"),
				Status:       "200 OK",
			},
		},
	})
	updated := updatedModel.(*Model)
	if updated.session.LastResponse != nil {
		t.Fatal("expected stale execute result to be ignored")
	}
	if !updated.viewState.ExecuteInFlight {
		t.Fatal("expected execute state to remain unchanged for stale result")
	}
}

func TestModelRequestEditSavesAndCancelsParameterDrafts(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForNavigation()
	m.viewState.FocusedPane = model.FocusedPaneRequest
	m.activeRequestSection = "Path"

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := updatedModel.(*Model)
	if updated.viewState.ActiveEditorMode != model.EditorModeEdit {
		t.Fatalf("expected enter to start field edit mode, got %q", updated.viewState.ActiveEditorMode)
	}
	if updated.viewState.RequestEditKind != model.RequestEditKindField {
		t.Fatalf("expected field request edit kind, got %q", updated.viewState.RequestEditKind)
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("pet-123")})
	updated = updatedModel.(*Model)
	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(*Model)

	draft := updated.ensureSelectedRequestDraft()
	if got := draft.PathParams["petId"]; got != "pet-123" {
		t.Fatalf("expected saved path param value, got %q", got)
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(*Model)
	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	updated = updatedModel.(*Model)
	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated = updatedModel.(*Model)

	draft = updated.ensureSelectedRequestDraft()
	if got := draft.PathParams["petId"]; got != "pet-123" {
		t.Fatalf("expected esc to discard in-progress field edit, got %q", got)
	}
}

func TestModelRequestBodyMediaTypeCyclesOnEnter(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForNavigation()
	m.session.Spec.Operations[0].RequestBody.Content = []model.MediaTypeSpec{
		{MediaType: "application/json"},
		{MediaType: "application/xml"},
	}
	m.viewState.FocusedPane = model.FocusedPaneRequest
	m.activeRequestSection = "Body"

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := updatedModel.(*Model)
	draft := updated.ensureSelectedRequestDraft()
	if draft.BodyMediaType != "application/xml" {
		t.Fatalf("expected enter on media type row to cycle to application/xml, got %q", draft.BodyMediaType)
	}
}

func TestModelRequestBodyEditorSavesAndCancels(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForNavigation()
	m.viewState.FocusedPane = model.FocusedPaneRequest
	m.activeRequestSection = "Body"
	m.viewState.RequestActiveRow = 1

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := updatedModel.(*Model)
	if updated.viewState.RequestEditKind != model.RequestEditKindBody {
		t.Fatalf("expected body edit mode, got %q", updated.viewState.RequestEditKind)
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("{")})
	updated = updatedModel.(*Model)
	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(*Model)
	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("}")})
	updated = updatedModel.(*Model)
	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	updated = updatedModel.(*Model)

	draft := updated.ensureSelectedRequestDraft()
	if got := draft.BodyRaw; got != "{\n}" {
		t.Fatalf("expected ctrl+s to save body editor contents, got %q", got)
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(*Model)
	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	updated = updatedModel.(*Model)
	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated = updatedModel.(*Model)

	draft = updated.ensureSelectedRequestDraft()
	if got := draft.BodyRaw; got != "{\n}" {
		t.Fatalf("expected esc to discard in-progress body edits, got %q", got)
	}
}

func TestModelRequestEditingBlocksNavigationUntilExit(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForNavigation()
	m.viewState.FocusedPane = model.FocusedPaneRequest
	m.activeRequestSection = "Path"

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := updatedModel.(*Model)

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4'}})
	updated = updatedModel.(*Model)
	if updated.viewState.FocusedPane != model.FocusedPaneRequest {
		t.Fatalf("expected request pane focus to remain while editing, got %q", updated.viewState.FocusedPane)
	}
	if updated.viewState.RequestEditBuffer != "4" {
		t.Fatalf("expected rune input to be captured by edit buffer, got %q", updated.viewState.RequestEditBuffer)
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	updated = updatedModel.(*Model)
	if updated.activeRequestSection != "Path" {
		t.Fatalf("expected request section navigation to be blocked during edit, got %q", updated.activeRequestSection)
	}
	if updated.session.SelectedOperationKey != model.NewOperationKey("GET", "/pets") {
		t.Fatalf("expected operation selection to remain unchanged during edit, got %q", updated.session.SelectedOperationKey)
	}
}

func TestModelQuestionMarkTogglesRequestPopupHelp(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForNavigation()
	m.viewState.FocusedPane = model.FocusedPaneRequest
	m.activeRequestSection = "Path"

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := updatedModel.(*Model)
	if updated.requestEditHelpOpen {
		t.Fatal("expected popup help to start hidden")
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	updated = updatedModel.(*Model)
	if !updated.requestEditHelpOpen {
		t.Fatal("expected question mark to open popup help")
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	updated = updatedModel.(*Model)
	if updated.requestEditHelpOpen {
		t.Fatal("expected next keypress to close popup help")
	}
}

func TestModelRequestDraftPersistsAcrossOperationAndFilterChanges(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForNavigation()
	m.viewState.FocusedPane = model.FocusedPaneRequest
	m.activeRequestSection = "Path"

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := updatedModel.(*Model)
	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("keep-me")})
	updated = updatedModel.(*Model)
	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(*Model)

	updated.viewState.FocusedPane = model.FocusedPaneOperations
	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	updated = updatedModel.(*Model)
	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	updated = updatedModel.(*Model)

	draft := updated.ensureSelectedRequestDraft()
	if got := draft.PathParams["petId"]; got != "keep-me" {
		t.Fatalf("expected draft to persist across operation changes, got %q", got)
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	updated = updatedModel.(*Model)
	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("admin")})
	updated = updatedModel.(*Model)
	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated = updatedModel.(*Model)
	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	updated = updatedModel.(*Model)

	draft = updated.ensureSelectedRequestDraft()
	if got := draft.PathParams["petId"]; got != "keep-me" {
		t.Fatalf("expected draft to persist across filter changes, got %q", got)
	}
}

func TestModelRequestScrollKeepsActiveRowVisible(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForNavigation()
	m.session.Spec.Operations[0].Parameters = []model.Parameter{
		{Name: "a", In: model.ParameterLocationQuery, Schema: &model.Schema{Type: "string"}},
		{Name: "b", In: model.ParameterLocationQuery, Schema: &model.Schema{Type: "string"}},
		{Name: "c", In: model.ParameterLocationQuery, Schema: &model.Schema{Type: "string"}},
		{Name: "d", In: model.ParameterLocationQuery, Schema: &model.Schema{Type: "string"}},
		{Name: "e", In: model.ParameterLocationQuery, Schema: &model.Schema{Type: "string"}},
	}
	m.width = 80
	m.height = 12
	m.viewState.RightPaneLayoutPreset = layoutPresetNarrow
	m.viewState.FocusedPane = model.FocusedPaneRequest
	m.activeRequestSection = "Query"

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	updated := updatedModel.(*Model)
	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	updated = updatedModel.(*Model)
	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	updated = updatedModel.(*Model)

	if updated.viewState.RequestActiveRow != 3 {
		t.Fatalf("expected request row cursor to move, got %d", updated.viewState.RequestActiveRow)
	}
	if updated.viewState.RequestScrollOffset == 0 {
		t.Fatalf("expected request scroll offset to advance to keep active row visible, got %d", updated.viewState.RequestScrollOffset)
	}
}

func TestModelDetailsScrollingUsesJKAndResetsOnSectionChange(t *testing.T) {
	t.Parallel()

	m := &Model{
		session: model.SessionState{
			Spec: &model.APISpec{
				Operations: []model.Operation{
					{
						Key:         model.NewOperationKey("GET", "/pets"),
						Method:      "GET",
						Path:        "/pets",
						Summary:     "List pets",
						Description: "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8",
						Tags:        []string{"pets"},
						Security: &model.SecurityRequirement{
							Alternatives: []model.SecurityAlternative{
								{Schemes: []model.SecurityRequirementRef{{Name: "api_key"}}},
							},
						},
					},
				},
			},
			SelectedOperationKey: model.NewOperationKey("GET", "/pets"),
		},
		viewState: model.ViewState{
			FocusedPane: model.FocusedPaneDetails,
		},
		width:                80,
		height:               12,
		activeDetailsSection: detailsui.SectionSummary,
	}
	m.viewState.RightPaneLayoutPreset = chooseLayoutPreset(m.width)

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	updated := updatedModel.(*Model)
	if updated.viewState.DetailsScrollOffset == 0 {
		t.Fatal("expected details scroll offset to increase with j")
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	updated = updatedModel.(*Model)
	if updated.activeDetailsSection != detailsui.SectionSecurity {
		t.Fatalf("expected ] to switch to security, got %q", updated.activeDetailsSection)
	}
	if updated.viewState.DetailsScrollOffset != 0 {
		t.Fatalf("expected details scroll offset to reset on section change, got %d", updated.viewState.DetailsScrollOffset)
	}
}

func TestModelDetailsHomeAndEndControlScroll(t *testing.T) {
	t.Parallel()

	m := &Model{
		session: model.SessionState{
			Spec: &model.APISpec{
				Operations: []model.Operation{
					{
						Key:         model.NewOperationKey("GET", "/pets"),
						Method:      "GET",
						Path:        "/pets",
						Summary:     "List pets",
						Description: "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8",
					},
				},
			},
			SelectedOperationKey: model.NewOperationKey("GET", "/pets"),
		},
		viewState: model.ViewState{
			FocusedPane:           model.FocusedPaneDetails,
			DetailsScrollOffset:   2,
			RightPaneLayoutPreset: layoutPresetNarrow,
		},
		width:                80,
		height:               12,
		activeDetailsSection: detailsui.SectionSummary,
	}

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnd})
	updated := updatedModel.(*Model)
	if updated.viewState.DetailsScrollOffset == 0 {
		t.Fatal("expected end to jump to the bottom of details content")
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyHome})
	updated = updatedModel.(*Model)
	if updated.viewState.DetailsScrollOffset != 0 {
		t.Fatalf("expected home to jump to top of details content, got %d", updated.viewState.DetailsScrollOffset)
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

func TestModelOperationsScrollingKeepsFiveRowsBelowCursorWhenMovingDown(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForNavigation()
	m.height = 18
	m.width = 80
	m.viewState.ZoomedPane = true
	m.viewState.FocusedPane = model.FocusedPaneOperations
	m.viewState.RightPaneLayoutPreset = layoutPresetWide
	m.session.Spec.Operations = nil
	m.viewState.VisibleOperationKeys = nil
	for index := 0; index < 20; index++ {
		path := "/pets/" + strconv.Itoa(index)
		key := model.NewOperationKey("GET", path)
		m.session.Spec.Operations = append(m.session.Spec.Operations, model.Operation{
			Key:    key,
			Method: "GET",
			Path:   path,
			Tags:   []string{"pets"},
		})
		m.viewState.VisibleOperationKeys = append(m.viewState.VisibleOperationKeys, key)
	}
	m.session.SelectedOperationKey = m.viewState.VisibleOperationKeys[0]
	m.viewState.OperationsCursor = 0

	for step := 0; step < 10; step++ {
		updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		m = updatedModel.(*Model)
	}

	if m.viewState.OperationsCursor != 10 {
		t.Fatalf("expected cursor to move to row 10, got %d", m.viewState.OperationsCursor)
	}
	if m.viewState.OperationsScrollOffset != 5 {
		t.Fatalf("expected scroll offset 5 to preserve five-row scrolloff at the bottom edge, got %d", m.viewState.OperationsScrollOffset)
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
					{StatusCode: "default", Description: "Unexpected error", Content: []model.MediaTypeSpec{{MediaType: "application/problem+json"}}},
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
		activeDetailsSection:  detailsui.SectionSummary,
		activeRequestSection:  "Path",
		activeResponseSection: "200",
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
