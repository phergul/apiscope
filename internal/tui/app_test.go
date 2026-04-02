package tui

import (
	"context"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/phergul/apiscope/internal/app"
	"github.com/phergul/apiscope/internal/model"
	"github.com/phergul/apiscope/internal/spec"
	detailsui "github.com/phergul/apiscope/internal/tui/details"
	requestui "github.com/phergul/apiscope/internal/tui/request"
	"github.com/phergul/apiscope/internal/tui/widgets"

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
	}, nil)

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
	if updated.viewState.Notice != "Spec loaded" {
		t.Fatalf("expected success notice, got %q", updated.viewState.Notice)
	}
	if updated.viewState.ExpandedRightPane != model.FocusedPaneRequest {
		t.Fatalf("expected request pane to start expanded, got %q", updated.viewState.ExpandedRightPane)
	}
	if updated.viewState.ZoomedPane {
		t.Fatal("expected zoom mode to start disabled")
	}
	if updated.panes.activeDetailsSection != detailsui.SectionSummary {
		t.Fatalf("expected summary details section after load, got %q", updated.panes.activeDetailsSection)
	}
}

func TestModelUpdatesFocusFromNumberKeys(t *testing.T) {
	t.Parallel()

	m := NewModel(app.NewService(&stubLoader{}, nil), "demo.yaml")

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

	m := NewModel(app.NewService(&stubLoader{}, nil), "demo.yaml")
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

	m := NewModel(app.NewService(&stubLoader{}, nil), "demo.yaml")

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

	m := NewModel(app.NewService(&stubLoader{}, nil), "demo.yaml")

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

	m := NewModel(app.NewService(&stubLoader{}, nil), "demo.yaml")
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

	m := NewModel(app.NewService(&stubLoader{}, nil), "demo.yaml")

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

	m := NewModel(app.NewService(&stubLoader{}, nil), "broken.yaml")
	m.shell.width = 120
	m.shell.height = 30
	m.viewState.LoadInFlight = true
	m.viewState.ActiveLoadRequestID = 1

	updatedModel, _ := m.Update(specLoadedMsg{
		requestID: 1,
		err:       errors.New("unable to parse spec"),
	})
	updated := updatedModel.(*Model)
	if updated.viewState.Notice != "Spec load failed" {
		t.Fatalf("expected load failure notice, got %q", updated.viewState.Notice)
	}
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

	m := NewModel(app.NewService(&stubLoader{}, nil), "broken.yaml")
	m.shell.loadErr = errors.New("boom")

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
		panes: paneState{
			activeDetailsSection: detailsui.SectionSummary,
		},
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
	if updated.panes.activeDetailsSection != detailsui.SectionSecurity {
		t.Fatalf("expected ] to move to security, got %q", updated.panes.activeDetailsSection)
	}

	updated.session.SelectedOperationKey = model.NewOperationKey("POST", "/pets")
	updated.syncActiveDetailsSection()
	if updated.panes.activeDetailsSection != detailsui.SectionSecurity {
		t.Fatalf("expected active section to stay on security when still available, got %q", updated.panes.activeDetailsSection)
	}

	updated.session.Spec.Security = nil
	updated.session.SelectedOperationKey = model.NewOperationKey("GET", "/health")
	updated.syncActiveDetailsSection()
	if updated.panes.activeDetailsSection != detailsui.SectionSummary {
		t.Fatalf("expected active section to fall back to summary when security is unavailable, got %q", updated.panes.activeDetailsSection)
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEnd})
	updated = updatedModel.(*Model)
	if updated.panes.activeDetailsSection != detailsui.SectionSummary {
		t.Fatalf("expected end to jump to last available details section, got %q", updated.panes.activeDetailsSection)
	}
}

func TestModelRequestSectionNavigationMovesAcrossParameterBodyAndAuth(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForNavigation()
	m.viewState.FocusedPane = model.FocusedPaneRequest
	m.panes.activeRequestSection = "Path"

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	updated := updatedModel.(*Model)
	if updated.panes.activeRequestSection != "Body" {
		t.Fatalf("expected ] to move to request body, got %q", updated.panes.activeRequestSection)
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	updated = updatedModel.(*Model)
	if updated.panes.activeRequestSection != "Auth" {
		t.Fatalf("expected ] to move to auth, got %q", updated.panes.activeRequestSection)
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}})
	updated = updatedModel.(*Model)
	if updated.panes.activeRequestSection != "Body" {
		t.Fatalf("expected [ to move back to body, got %q", updated.panes.activeRequestSection)
	}
}

func TestModelProjectRequestPaneShowsSupportNotesForUnsupportedAndDowngradedInputs(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForNavigation()
	m.viewState.FocusedPane = model.FocusedPaneRequest
	m.panes.activeRequestSection = "Query"
	m.session.Spec.Operations[0].Parameters = append(m.session.Spec.Operations[0].Parameters,
		model.Parameter{
			Name:             "tags",
			In:               model.ParameterLocationQuery,
			CollectionFormat: "pipes",
			Schema:           &model.Schema{Type: "array"},
		},
		model.Parameter{
			Name:    "legacy",
			In:      model.ParameterLocationQuery,
			Content: []model.MediaTypeSpec{{MediaType: "application/json"}},
		},
	)

	data := m.projectRequestPane()
	if len(data.SupportNotice) != 2 {
		t.Fatalf("expected section support notes for query inputs, got %#v", data.SupportNotice)
	}
	if len(data.Rows) != 2 {
		t.Fatalf("expected projected query rows, got %#v", data.Rows)
	}
	if len(data.Rows[0].Support) != 1 || len(data.Rows[1].Support) != 1 {
		t.Fatalf("expected row support notes, got %#v", data.Rows)
	}
}

func TestModelRequestRowNavigationMovesWithinActiveSection(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForNavigation()
	m.viewState.FocusedPane = model.FocusedPaneRequest
	m.panes.activeRequestSection = "Body"

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

func TestModelRequestCursorSkipsAuthOptionHeaderRows(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForNavigation()
	m.viewState.FocusedPane = model.FocusedPaneRequest
	m.panes.activeRequestSection = requestui.SectionAuth
	m.syncActiveRequestRow()

	if m.viewState.RequestActiveRow != 1 {
		t.Fatalf("expected auth section to start on first editable field row, got %d", m.viewState.RequestActiveRow)
	}

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyHome})
	updated := updatedModel.(*Model)
	if updated.viewState.RequestActiveRow != 1 {
		t.Fatalf("expected home to stay on first editable auth row, got %d", updated.viewState.RequestActiveRow)
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
		panes: paneState{
			activeDetailsSection:  detailsui.SectionSummary,
			activeRequestSection:  "Query",
			activeResponseSection: "default",
		},
	}

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	updated := updatedModel.(*Model)
	if updated.session.SelectedOperationKey != model.NewOperationKey("GET", "/second") {
		t.Fatalf("expected second operation to be selected, got %q", updated.session.SelectedOperationKey)
	}
	if updated.panes.activeRequestSection != "Path" {
		t.Fatalf("expected request section to reset to first available, got %q", updated.panes.activeRequestSection)
	}
	if updated.viewState.RequestActiveRow != 0 {
		t.Fatalf("expected request row to reset on operation change, got %d", updated.viewState.RequestActiveRow)
	}
	if updated.viewState.RequestScrollOffset != 0 {
		t.Fatalf("expected request scroll offset to reset on operation change, got %d", updated.viewState.RequestScrollOffset)
	}
	if updated.panes.activeResponseSection != "Live" {
		t.Fatalf("expected response section to reset to live, got %q", updated.panes.activeResponseSection)
	}
}

func TestModelResponseSectionNavigationAndFallback(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForNavigation()
	m.viewState.FocusedPane = model.FocusedPaneResponse
	m.panes.activeResponseSection = "200"

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	updated := updatedModel.(*Model)
	if updated.panes.activeResponseSection != "default" {
		t.Fatalf("expected ] to move to default response, got %q", updated.panes.activeResponseSection)
	}

	updated.session.SelectedOperationKey = model.NewOperationKey("POST", "/pets")
	updated.syncActivePaneSections()
	if updated.panes.activeResponseSection != "Live" {
		t.Fatalf("expected response section to fall back to live, got %q", updated.panes.activeResponseSection)
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
	if !updated.requestUI.validation.HasIssues() {
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
		if got := r.Header.Get("X-API-Key"); got != "secret" {
			t.Fatalf("expected X-API-Key header secret, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	m := newLoadedModelForNavigation()
	m.service = app.NewService(nil, nil)
	m.viewState.FocusedPane = model.FocusedPaneRequest
	m.session.SelectedServerURL = server.URL
	m.session.AuthState = map[string]model.AuthValue{
		"api_key": {Type: model.AuthSchemeValueTypeAPIKey, APIKey: "secret"},
	}
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
	if updated.panes.activeResponseSection != "Live" {
		t.Fatalf("expected live response section to be selected, got %q", updated.panes.activeResponseSection)
	}
	if updated.session.LastResponse == nil {
		t.Fatal("expected last response to be stored")
	}
	if updated.session.LastResponse.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP 200 response, got %d", updated.session.LastResponse.StatusCode)
	}
}

func TestModelCtrlRStillExecutesWhenOperationShowsSupportNotes(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	m := newLoadedModelForNavigation()
	m.service = app.NewService(nil, nil)
	m.viewState.FocusedPane = model.FocusedPaneRequest
	m.session.SelectedServerURL = server.URL
	m.session.AuthState = map[string]model.AuthValue{
		"api_key": {Type: model.AuthSchemeValueTypeAPIKey, APIKey: "secret"},
	}
	m.session.Spec.Operations[0].Parameters = append(m.session.Spec.Operations[0].Parameters, model.Parameter{
		Name:             "tags",
		In:               model.ParameterLocationQuery,
		CollectionFormat: "pipes",
		Schema:           &model.Schema{Type: "array"},
	})
	draft := m.ensureSelectedRequestDraft()
	draft.PathParams["petId"] = "abc"
	draft.BodyRaw = `{"name":"fido"}`
	draft.BodyMediaType = "application/json"
	draft.QueryParams["tags"] = "a|b"

	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	updated := updatedModel.(*Model)
	if cmd == nil {
		t.Fatal("expected ctrl+r to execute even with support notes present")
	}

	msg := cmd()
	updatedModel, _ = updated.Update(msg)
	updated = updatedModel.(*Model)
	if updated.session.LastResponse == nil {
		t.Fatal("expected response after execution")
	}
	if updated.session.LastResponse.TransportError != "" {
		t.Fatalf("expected successful execution, got %q", updated.session.LastResponse.TransportError)
	}
}

func TestModelCtrlRExecutesUrlencodedFormOperation(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Method; got != http.MethodPost {
			t.Fatalf("expected POST request, got %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
			t.Fatalf("expected form content type, got %q", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("ReadAll returned error: %v", err)
		}
		if got := string(body); got != "name=fido" {
			t.Fatalf("expected urlencoded form body, got %q", got)
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	m := newLoadedModelForNavigation()
	m.service = app.NewService(nil, nil)
	m.viewState.FocusedPane = model.FocusedPaneRequest
	m.session.SelectedOperationKey = model.NewOperationKey("POST", "/pets")
	m.session.SelectedServerURL = server.URL
	m.session.Spec.Security = nil
	m.session.Spec.Operations[1].Parameters = []model.Parameter{
		{Name: "name", In: model.ParameterLocationForm, Required: true, Schema: &model.Schema{Type: "string"}},
	}
	m.session.Spec.Operations[1].FormBodyMediaType = "application/x-www-form-urlencoded"
	m.session.Spec.Operations[1].Security = nil
	m.panes.activeRequestSection = requestui.SectionForm

	draft := m.ensureSelectedRequestDraft()
	draft.FormParams["name"] = "fido"

	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	updated := updatedModel.(*Model)
	if cmd == nil {
		t.Fatal("expected ctrl+r to execute form request")
	}

	msg := cmd()
	updatedModel, _ = updated.Update(msg)
	updated = updatedModel.(*Model)
	if updated.session.LastResponse == nil {
		t.Fatal("expected response after execution")
	}
	if updated.session.LastResponse.TransportError != "" {
		t.Fatalf("expected successful execution, got %q", updated.session.LastResponse.TransportError)
	}
}

func TestModelCtrlRExecutesMultipartFormOperation(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil {
			t.Fatalf("ParseMediaType returned error: %v", err)
		}
		if mediaType != "multipart/form-data" {
			t.Fatalf("expected multipart form content type, got %q", mediaType)
		}
		reader := multipart.NewReader(r.Body, params["boundary"])
		part, err := reader.NextPart()
		if err != nil {
			t.Fatalf("NextPart returned error: %v", err)
		}
		body, err := io.ReadAll(part)
		if err != nil {
			t.Fatalf("ReadAll returned error: %v", err)
		}
		if got := part.FormName(); got != "name" {
			t.Fatalf("expected multipart field name, got %q", got)
		}
		if got := string(body); got != "fido" {
			t.Fatalf("expected multipart field value, got %q", got)
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	m := newLoadedModelForNavigation()
	m.service = app.NewService(nil, nil)
	m.viewState.FocusedPane = model.FocusedPaneRequest
	m.session.SelectedOperationKey = model.NewOperationKey("POST", "/pets")
	m.session.SelectedServerURL = server.URL
	m.session.Spec.Security = nil
	m.session.Spec.Operations[1].Parameters = []model.Parameter{
		{Name: "name", In: model.ParameterLocationForm, Required: true, Schema: &model.Schema{Type: "string"}},
	}
	m.session.Spec.Operations[1].FormBodyMediaType = "multipart/form-data"
	m.session.Spec.Operations[1].Security = nil
	m.panes.activeRequestSection = requestui.SectionForm

	draft := m.ensureSelectedRequestDraft()
	draft.FormParams["name"] = "fido"

	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	updated := updatedModel.(*Model)
	if cmd == nil {
		t.Fatal("expected ctrl+r to execute multipart form request")
	}

	msg := cmd()
	updatedModel, _ = updated.Update(msg)
	updated = updatedModel.(*Model)
	if updated.session.LastResponse == nil {
		t.Fatal("expected response after execution")
	}
	if updated.session.LastResponse.TransportError != "" {
		t.Fatalf("expected successful execution, got %q", updated.session.LastResponse.TransportError)
	}
}

func TestModelCtrlRExecutesMultipartFileUploadOperation(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	uploadPath := filepath.Join(tempDir, "avatar.txt")
	if err := os.WriteFile(uploadPath, []byte("hello file"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil {
			t.Fatalf("ParseMediaType returned error: %v", err)
		}
		if mediaType != "multipart/form-data" {
			t.Fatalf("expected multipart form content type, got %q", mediaType)
		}
		reader := multipart.NewReader(r.Body, params["boundary"])
		seen := map[string]string{}
		names := map[string]string{}
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("NextPart returned error: %v", err)
			}
			body, err := io.ReadAll(part)
			if err != nil {
				t.Fatalf("ReadAll returned error: %v", err)
			}
			seen[part.FormName()] = string(body)
			names[part.FormName()] = part.FileName()
		}
		if got := seen["description"]; got != "avatar" {
			t.Fatalf("expected multipart scalar field, got %q", got)
		}
		if got := seen["file"]; got != "hello file" {
			t.Fatalf("expected multipart file body, got %q", got)
		}
		if got := names["file"]; got != "avatar.txt" {
			t.Fatalf("expected multipart filename, got %q", got)
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	m := newLoadedModelForNavigation()
	m.service = app.NewService(nil, nil)
	m.viewState.FocusedPane = model.FocusedPaneRequest
	m.session.SelectedOperationKey = model.NewOperationKey("POST", "/pets")
	m.session.SelectedServerURL = server.URL
	m.session.Spec.Security = nil
	m.session.Spec.Operations[1].Parameters = []model.Parameter{
		{Name: "description", In: model.ParameterLocationForm, Schema: &model.Schema{Type: "string"}},
		{Name: "file", In: model.ParameterLocationForm, FormInputKind: model.FormInputKindFile, Required: true},
	}
	m.session.Spec.Operations[1].FormBodyMediaType = "multipart/form-data"
	m.session.Spec.Operations[1].Security = nil
	m.panes.activeRequestSection = requestui.SectionForm

	draft := m.ensureSelectedRequestDraft()
	draft.FormParams["description"] = "avatar"
	draft.FormFileParams["file"] = uploadPath

	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	updated := updatedModel.(*Model)
	if cmd == nil {
		t.Fatal("expected ctrl+r to execute multipart file request")
	}

	msg := cmd()
	updatedModel, _ = updated.Update(msg)
	updated = updatedModel.(*Model)
	if updated.session.LastResponse == nil {
		t.Fatal("expected response after execution")
	}
	if updated.session.LastResponse.TransportError != "" {
		t.Fatalf("expected successful execution, got %q", updated.session.LastResponse.TransportError)
	}
	if got := updated.session.RequestHistory[len(updated.session.RequestHistory)-1].Request.Draft.FormFileParams["file"]; got != uploadPath {
		t.Fatalf("expected request history snapshot to preserve file path, got %q", got)
	}
}

func TestModelCtrlRShowsAuthValidationInAuthSection(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForNavigation()
	m.viewState.FocusedPane = model.FocusedPaneRequest
	draft := m.ensureSelectedRequestDraft()
	draft.PathParams["petId"] = "abc"
	draft.BodyRaw = `{"name":"fido"}`
	draft.BodyMediaType = "application/json"

	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	updated := updatedModel.(*Model)
	if cmd != nil {
		t.Fatal("expected ctrl+r to skip execution when auth validation fails")
	}
	if updated.panes.activeRequestSection != requestui.SectionAuth {
		t.Fatalf("expected auth section to become active, got %q", updated.panes.activeRequestSection)
	}
	if !updated.requestUI.validation.HasIssues() {
		t.Fatal("expected auth validation issues to be stored")
	}
	data := updated.projectRequestPane()
	if len(data.Rows) < 2 || data.Rows[1].Error == "" {
		t.Fatalf("expected auth row to show inline validation error, got %#v", data.Rows)
	}
}

func TestModelCtrlRFocusesFirstMissingFieldInBestAuthAlternative(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForNavigation()
	m.session.Spec.Operations[0].Security = &model.SecurityRequirement{
		Alternatives: []model.SecurityAlternative{
			{Schemes: []model.SecurityRequirementRef{{Name: "basic_auth"}}},
			{Schemes: []model.SecurityRequirementRef{{Name: "api_key"}}},
		},
	}
	m.session.Spec.SecuritySchemes["basic_auth"] = model.SecurityScheme{
		Name:   "basic_auth",
		Type:   model.SecuritySchemeTypeHTTP,
		Scheme: model.HTTPAuthSchemeBasic,
	}
	m.session.AuthState = map[string]model.AuthValue{
		"basic_auth": {Type: model.AuthSchemeValueTypeBasic, Username: "alice"},
	}
	m.viewState.FocusedPane = model.FocusedPaneRequest
	draft := m.ensureSelectedRequestDraft()
	draft.PathParams["petId"] = "abc"
	draft.BodyRaw = `{"name":"fido"}`
	draft.BodyMediaType = "application/json"

	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	updated := updatedModel.(*Model)
	if cmd != nil {
		t.Fatal("expected ctrl+r to stop on auth validation")
	}
	if updated.panes.activeRequestSection != requestui.SectionAuth {
		t.Fatalf("expected auth section to stay active, got %q", updated.panes.activeRequestSection)
	}
	if updated.viewState.RequestActiveRow != 2 {
		t.Fatalf("expected cursor to focus first missing field row, got %d", updated.viewState.RequestActiveRow)
	}
}

func TestModelCtrlRExecutesWhenLaterAuthAlternativeIsSatisfied(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-API-Key"); got != "secret" {
			t.Fatalf("expected api key from later auth alternative, got %q", got)
		}
		_, _ = w.Write([]byte(`ok`))
	}))
	defer server.Close()

	m := newLoadedModelForNavigation()
	m.service = app.NewService(nil, nil)
	m.session.SelectedServerURL = server.URL
	m.session.Spec.Operations[0].Security = &model.SecurityRequirement{
		Alternatives: []model.SecurityAlternative{
			{Schemes: []model.SecurityRequirementRef{{Name: "bearer_auth"}}},
			{Schemes: []model.SecurityRequirementRef{{Name: "api_key"}}},
		},
	}
	m.session.Spec.SecuritySchemes["bearer_auth"] = model.SecurityScheme{
		Name:   "bearer_auth",
		Type:   model.SecuritySchemeTypeHTTP,
		Scheme: model.HTTPAuthSchemeBearer,
	}
	m.session.AuthState = map[string]model.AuthValue{
		"api_key": {Type: model.AuthSchemeValueTypeAPIKey, APIKey: "secret"},
	}
	m.viewState.FocusedPane = model.FocusedPaneRequest
	draft := m.ensureSelectedRequestDraft()
	draft.PathParams["petId"] = "abc"
	draft.BodyRaw = `{"name":"fido"}`
	draft.BodyMediaType = "application/json"

	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	updated := updatedModel.(*Model)
	if cmd == nil {
		t.Fatal("expected ctrl+r to execute when later auth alternative is ready")
	}

	msg := cmd()
	updatedModel, _ = updated.Update(msg)
	updated = updatedModel.(*Model)
	if updated.session.LastResponse == nil {
		t.Fatal("expected response after execution")
	}
	if updated.session.LastResponse.TransportError != "" {
		t.Fatalf("expected successful execution, got %q", updated.session.LastResponse.TransportError)
	}
}

func TestModelPOpensAndClosesPreviousRequestsPopup(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForNavigation()

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	updated := updatedModel.(*Model)
	if !updated.historyPopupOpen() {
		t.Fatal("expected p to open the previous-requests popup")
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated = updatedModel.(*Model)
	if updated.historyPopupOpen() {
		t.Fatal("expected esc to close the previous-requests popup")
	}
}

func TestModelPreviousRequestsPopupNavigationUsesJKHomeAndEnd(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForNavigation()
	m.session.RequestHistory = []model.HistoryEntry{
		{RequestID: 1, OperationKey: model.NewOperationKey("GET", "/pets")},
		{RequestID: 2, OperationKey: model.NewOperationKey("GET", "/pets")},
		{RequestID: 3, OperationKey: model.NewOperationKey("GET", "/pets")},
	}

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	updated := updatedModel.(*Model)
	if updated.historyUI.activeRow != 0 {
		t.Fatalf("expected popup to start at newest entry, got %d", updated.historyUI.activeRow)
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	updated = updatedModel.(*Model)
	if updated.historyUI.activeRow != 1 {
		t.Fatalf("expected j to move selection down, got %d", updated.historyUI.activeRow)
	}
	if updated.historyUI.previewScrollOffset != 0 {
		t.Fatalf("expected selection changes to reset preview scroll, got %d", updated.historyUI.previewScrollOffset)
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEnd})
	updated = updatedModel.(*Model)
	if updated.historyUI.activeRow != 2 {
		t.Fatalf("expected end to jump to last entry, got %d", updated.historyUI.activeRow)
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyHome})
	updated = updatedModel.(*Model)
	if updated.historyUI.activeRow != 0 {
		t.Fatalf("expected home to jump to first entry, got %d", updated.historyUI.activeRow)
	}
}

func TestModelPreviousRequestsPopupOpenResetsPreviewScroll(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForNavigation()
	m.historyUI.previewScrollOffset = 7
	m.session.RequestHistory = []model.HistoryEntry{
		{RequestID: 1, OperationKey: model.NewOperationKey("GET", "/pets")},
	}

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	updated := updatedModel.(*Model)
	if updated.historyUI.previewScrollOffset != 0 {
		t.Fatalf("expected popup open to reset preview scroll, got %d", updated.historyUI.previewScrollOffset)
	}
}

func TestModelPreviousRequestsPopupCtrlUDScrollsPreviewOnly(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForNavigation()
	m.shell.width = 120
	m.shell.height = 20
	m.session.RequestHistory = []model.HistoryEntry{{
		RequestID:    1,
		OperationKey: model.NewOperationKey("GET", "/pets"),
		ServerURL:    "https://api.example.com",
		Request: model.ExecutedRequestSnapshot{
			ServerURL: "https://api.example.com",
			Draft: model.RequestDraft{
				BodyRaw: "{\n  \"name\": \"fido\"\n}",
			},
		},
		Response: &model.HTTPResponse{
			Status:      "200 OK",
			PrettyBody:  "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10\nline11\nline12\nline13\nline14",
			ContentType: "application/json",
		},
	}}

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	updated := updatedModel.(*Model)
	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	updated = updatedModel.(*Model)
	if updated.historyUI.activeRow != 0 {
		t.Fatalf("expected ctrl+d to leave the active row unchanged, got %d", updated.historyUI.activeRow)
	}
	if updated.historyUI.previewScrollOffset == 0 {
		t.Fatal("expected ctrl+d to scroll the preview")
	}

	before := updated.historyUI.previewScrollOffset
	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
	updated = updatedModel.(*Model)
	if updated.historyUI.previewScrollOffset >= before {
		t.Fatalf("expected ctrl+u to scroll preview upward, got %d", updated.historyUI.previewScrollOffset)
	}
}

func TestModelPreviousRequestsPopupEnterLoadsHistoricalResponse(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForNavigation()
	m.viewState.FocusedPane = model.FocusedPaneOperations
	m.session.LastResponse = &model.HTTPResponse{Status: "202 Accepted", PrettyBody: "current"}
	m.session.RequestHistory = []model.HistoryEntry{
		{
			RequestID:    7,
			OperationKey: model.NewOperationKey("GET", "/pets"),
			ServerURL:    "https://api.example.com",
			Response: &model.HTTPResponse{
				OperationKey: model.NewOperationKey("GET", "/pets"),
				Status:       "200 OK",
				PrettyBody:   "historical",
			},
			Request: model.ExecutedRequestSnapshot{
				ServerURL: "https://api.example.com",
			},
		},
	}

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	updated := updatedModel.(*Model)
	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(*Model)

	if updated.historyPopupOpen() {
		t.Fatal("expected enter to close the popup")
	}
	if updated.viewState.FocusedPane != model.FocusedPaneResponse {
		t.Fatalf("expected enter to focus response pane, got %q", updated.viewState.FocusedPane)
	}
	if updated.panes.activeResponseSection != "Live" {
		t.Fatalf("expected enter to select live response, got %q", updated.panes.activeResponseSection)
	}
	if updated.session.LastResponse == nil || updated.session.LastResponse.PrettyBody != "historical" {
		t.Fatalf("expected historical response to load, got %#v", updated.session.LastResponse)
	}
	if updated.viewState.Notice != "Loaded previous response #7" {
		t.Fatalf("expected response recall notice, got %q", updated.viewState.Notice)
	}
}

func TestModelPreviousRequestsPopupRestoreRequestFocusesPaneThree(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForNavigation()
	operationKey := model.NewOperationKey("GET", "/pets")
	draftKey := model.NewDraftKey("", operationKey)
	m.session.SelectedServerURL = "https://current.example.com"
	m.session.RequestDrafts = map[model.DraftKey]*model.RequestDraft{
		draftKey: {
			Key:          draftKey,
			OperationKey: operationKey,
			ServerURL:    "https://current.example.com",
			PathParams:   map[string]string{"petId": "current"},
		},
	}
	m.session.AuthState = map[string]model.AuthValue{
		"api_key": {Type: model.AuthSchemeValueTypeAPIKey, APIKey: "current"},
	}
	m.panes.activeRequestSection = requestui.SectionAuth
	m.session.RequestHistory = []model.HistoryEntry{
		{
			RequestID:    9,
			OperationKey: operationKey,
			ServerURL:    "https://history.example.com",
			Request: model.ExecutedRequestSnapshot{
				OperationKey: operationKey,
				ServerURL:    "https://history.example.com",
				Draft: model.RequestDraft{
					Key:          draftKey,
					OperationKey: operationKey,
					ServerURL:    "https://history.example.com",
					PathParams:   map[string]string{"petId": "from-history"},
					BodyRaw:      `{"name":"fido"}`,
				},
				AuthState: map[string]model.AuthValue{
					"api_key": {Type: model.AuthSchemeValueTypeAPIKey, APIKey: "restored"},
				},
			},
			Response: &model.HTTPResponse{Status: "200 OK"},
		},
	}

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	updated := updatedModel.(*Model)
	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	updated = updatedModel.(*Model)

	if updated.historyPopupOpen() {
		t.Fatal("expected r to close the popup")
	}
	if updated.viewState.FocusedPane != model.FocusedPaneRequest {
		t.Fatalf("expected request restore to focus pane 3, got %q", updated.viewState.FocusedPane)
	}
	if got := updated.session.SelectedServerURL; got != "https://history.example.com" {
		t.Fatalf("expected server restore, got %q", got)
	}
	if got := updated.ensureSelectedRequestDraft().PathParams["petId"]; got != "from-history" {
		t.Fatalf("expected request draft restore, got %q", got)
	}
	if got := updated.session.AuthState["api_key"].APIKey; got != "restored" {
		t.Fatalf("expected auth state restore, got %q", got)
	}
	if updated.viewState.Notice != "Restored request #9" {
		t.Fatalf("expected request restore notice, got %q", updated.viewState.Notice)
	}
}

func TestModelPreviousRequestsPopupBlocksNormalPaneNavigation(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForNavigation()
	m.session.RequestHistory = []model.HistoryEntry{
		{RequestID: 1, OperationKey: model.NewOperationKey("GET", "/pets")},
	}

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	updated := updatedModel.(*Model)
	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4'}})
	updated = updatedModel.(*Model)
	if updated.viewState.FocusedPane != model.FocusedPaneOperations {
		t.Fatalf("expected popup to block focus changes, got %q", updated.viewState.FocusedPane)
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	updated = updatedModel.(*Model)
	if updated.session.SelectedOperationKey != model.NewOperationKey("GET", "/pets") {
		t.Fatalf("expected popup to block operation navigation, got %q", updated.session.SelectedOperationKey)
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	updated = updatedModel.(*Model)
	if updated.historyUI.activeRow != 0 {
		t.Fatalf("expected preview scrolling to leave row selection unchanged, got %d", updated.historyUI.activeRow)
	}
}

func TestModelPreviousRequestsPopupShowsNewExecutionHistory(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	m := newLoadedModelForNavigation()
	m.service = app.NewService(nil, nil)
	m.session.SelectedServerURL = server.URL
	m.session.AuthState = map[string]model.AuthValue{
		"api_key": {Type: model.AuthSchemeValueTypeAPIKey, APIKey: "secret"},
	}
	draft := m.ensureSelectedRequestDraft()
	draft.PathParams["petId"] = "abc"
	draft.BodyRaw = `{"name":"fido"}`
	draft.BodyMediaType = "application/json"

	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	updated := updatedModel.(*Model)
	if cmd == nil {
		t.Fatal("expected ctrl+r to execute")
	}

	msg := cmd()
	updatedModel, _ = updated.Update(msg)
	updated = updatedModel.(*Model)

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	updated = updatedModel.(*Model)
	view := stripANSI(updated.View())
	if !strings.Contains(view, "Previous requests") {
		t.Fatalf("expected popup heading after execution, got %q", view)
	}
	if !strings.Contains(view, "Server: "+server.URL) {
		t.Fatalf("expected popup details to include executed server, got %q", view)
	}
	if !strings.Contains(view, "Path: petId=abc") {
		t.Fatalf("expected popup details to include executed inputs, got %q", view)
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

func TestModelExecuteFinishedSetsSuccessNotice(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForNavigation()
	m.viewState.ExecuteInFlight = true
	m.viewState.ActiveExecuteRequestID = 1

	updatedModel, _ := m.Update(executeFinishedMsg{
		requestID: 1,
		result: app.ExecuteResult{
			OperationKey: model.NewOperationKey("GET", "/pets"),
			Response: &model.HTTPResponse{
				OperationKey: model.NewOperationKey("GET", "/pets"),
				StatusCode:   200,
				Status:       "200 OK",
			},
		},
	})
	updated := updatedModel.(*Model)
	if updated.viewState.Notice != "Request succeeded" {
		t.Fatalf("expected success notice, got %q", updated.viewState.Notice)
	}
}

func TestModelExecuteFinishedSetsFailureNotice(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForNavigation()
	m.viewState.ExecuteInFlight = true
	m.viewState.ActiveExecuteRequestID = 1

	updatedModel, _ := m.Update(executeFinishedMsg{
		requestID: 1,
		result: app.ExecuteResult{
			OperationKey: model.NewOperationKey("GET", "/pets"),
			Response: &model.HTTPResponse{
				OperationKey:   model.NewOperationKey("GET", "/pets"),
				TransportError: "dial tcp: connection refused",
			},
		},
	})
	updated := updatedModel.(*Model)
	if updated.viewState.Notice != "Request failed" {
		t.Fatalf("expected failure notice, got %q", updated.viewState.Notice)
	}
}

func TestModelRequestEditSavesAndCancelsParameterDrafts(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForNavigation()
	m.viewState.FocusedPane = model.FocusedPaneRequest
	m.panes.activeRequestSection = "Path"

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
	m.panes.activeRequestSection = "Body"

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := updatedModel.(*Model)
	draft := updated.ensureSelectedRequestDraft()
	if draft.BodyMediaType != "application/xml" {
		t.Fatalf("expected enter on media type row to cycle to application/xml, got %q", draft.BodyMediaType)
	}
}

func TestModelRequestServerCyclesOnEnterAndExecutionUsesNewSelection(t *testing.T) {
	t.Parallel()

	production := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("production"))
	}))
	defer production.Close()

	staging := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("staging"))
	}))
	defer staging.Close()

	m := &Model{
		service: app.NewService(nil, nil),
		session: model.SessionState{
			Spec: &model.APISpec{
				Servers: []model.Server{
					{URL: production.URL, Description: "Production"},
					{URL: staging.URL, Description: "Staging"},
				},
				Operations: []model.Operation{
					{
						Key:    model.NewOperationKey("GET", "/pets"),
						Method: "GET",
						Path:   "/pets",
						Responses: []model.ResponseSpec{
							{StatusCode: "200", Description: "OK"},
						},
					},
				},
			},
			SelectedServerURL:    production.URL,
			SelectedOperationKey: model.NewOperationKey("GET", "/pets"),
			RequestDrafts:        map[model.DraftKey]*model.RequestDraft{},
		},
		viewState: model.ViewState{
			FocusedPane: model.FocusedPaneRequest,
			VisibleOperationKeys: []model.OperationKey{
				model.NewOperationKey("GET", "/pets"),
			},
		},
		panes: paneState{
			activeRequestSection: requestui.SectionServer,
		},
	}

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := updatedModel.(*Model)
	if updated.session.SelectedServerURL != staging.URL {
		t.Fatalf("expected enter on server row to cycle to staging, got %q", updated.session.SelectedServerURL)
	}

	updatedModel, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	updated = updatedModel.(*Model)
	if cmd == nil {
		t.Fatal("expected ctrl+r to start execution after switching servers")
	}

	msg := cmd()
	updatedModel, _ = updated.Update(msg)
	updated = updatedModel.(*Model)
	if updated.session.LastResponse == nil {
		t.Fatal("expected live response after execution")
	}
	if got := string(updated.session.LastResponse.Body); got != "staging" {
		t.Fatalf("expected execution to use staging server, got %q", got)
	}
}

func TestModelRequestBodyEditorSavesAndCancels(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForNavigation()
	m.viewState.FocusedPane = model.FocusedPaneRequest
	m.panes.activeRequestSection = "Body"
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
	m.panes.activeRequestSection = "Path"

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
	if updated.panes.activeRequestSection != "Path" {
		t.Fatalf("expected request section navigation to be blocked during edit, got %q", updated.panes.activeRequestSection)
	}
	if updated.session.SelectedOperationKey != model.NewOperationKey("GET", "/pets") {
		t.Fatalf("expected operation selection to remain unchanged during edit, got %q", updated.session.SelectedOperationKey)
	}
}

func TestModelQuestionMarkOpensBrowseHelpForFocusedPane(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		pane  model.FocusedPane
		title string
	}{
		{name: "operations", pane: model.FocusedPaneOperations, title: "Operations help"},
		{name: "details", pane: model.FocusedPaneDetails, title: "Details help"},
		{name: "request", pane: model.FocusedPaneRequest, title: "Request help"},
		{name: "response", pane: model.FocusedPaneResponse, title: "Response help"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := newLoadedModelForNavigation()
			m.viewState.FocusedPane = tt.pane

			updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
			updated := updatedModel.(*Model)
			if !updated.helpOverlayOpen() {
				t.Fatal("expected question mark to open contextual help")
			}
			if updated.helpUI.view.Title != tt.title {
				t.Fatalf("expected help title %q, got %q", tt.title, updated.helpUI.view.Title)
			}
		})
	}
}

func TestModelHelpOverlayBlocksBrowseMutationUntilClosed(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForNavigation()

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	updated := updatedModel.(*Model)
	if !updated.helpOverlayOpen() {
		t.Fatal("expected help overlay to open")
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	updated = updatedModel.(*Model)
	if updated.viewState.OperationsCursor != 0 {
		t.Fatalf("expected browse movement to be blocked while help is open, got cursor %d", updated.viewState.OperationsCursor)
	}
	if !updated.helpOverlayOpen() {
		t.Fatal("expected help overlay to remain open after ignored key")
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated = updatedModel.(*Model)
	if updated.helpOverlayOpen() {
		t.Fatal("expected esc to close help overlay")
	}
}

func TestModelHelpOverlayStillAllowsQuitKeys(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForNavigation()

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	updated := updatedModel.(*Model)
	if !updated.helpOverlayOpen() {
		t.Fatal("expected help overlay to open")
	}

	_, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("expected q to keep quitting while help is open")
	}
}

func TestModelThemeKeysCycleThemesInBrowseMode(t *testing.T) {
	t.Parallel()

	original := widgets.CurrentTheme()
	t.Cleanup(func() {
		widgets.SetTheme(original)
	})
	widgets.SetThemeByName("default")

	m := newLoadedModelForNavigation()

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	updated := updatedModel.(*Model)
	if widgets.CurrentTheme().Name != widgets.NextThemeName("default") {
		t.Fatalf("expected next theme after default, got %q", widgets.CurrentTheme().Name)
	}
	if updated.viewState.Notice != "Theme: harbor" {
		t.Fatalf("expected theme change notice, got %q", updated.viewState.Notice)
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'T'}})
	updated = updatedModel.(*Model)
	if widgets.CurrentTheme().Name != "default" {
		t.Fatalf("expected uppercase T to cycle back to default, got %q", widgets.CurrentTheme().Name)
	}
	if updated.viewState.Notice != "Theme: default" {
		t.Fatalf("expected previous theme notice, got %q", updated.viewState.Notice)
	}
}

func TestModelThemeKeysDoNothingWhileHelpIsOpen(t *testing.T) {
	t.Parallel()

	original := widgets.CurrentTheme()
	t.Cleanup(func() {
		widgets.SetTheme(original)
	})
	widgets.SetThemeByName("default")

	m := newLoadedModelForNavigation()

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	updated := updatedModel.(*Model)
	if !updated.helpOverlayOpen() {
		t.Fatal("expected help overlay to open")
	}
	noticeBefore := updated.viewState.Notice

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	updated = updatedModel.(*Model)
	if widgets.CurrentTheme().Name != "default" {
		t.Fatalf("expected theme to remain unchanged while help is open, got %q", widgets.CurrentTheme().Name)
	}
	if updated.viewState.Notice != noticeBefore {
		t.Fatalf("expected notice to remain unchanged while help is open, got %q", updated.viewState.Notice)
	}
}

func TestModelQuestionMarkOpensRequestEditHelpAndEscClosesHelpFirst(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForNavigation()
	m.viewState.FocusedPane = model.FocusedPaneRequest
	m.panes.activeRequestSection = "Path"

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := updatedModel.(*Model)
	if updated.viewState.RequestEditKind != model.RequestEditKindField {
		t.Fatalf("expected field edit mode, got %q", updated.viewState.RequestEditKind)
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	updated = updatedModel.(*Model)
	if !updated.helpOverlayOpen() {
		t.Fatal("expected request edit help to open")
	}
	if updated.helpUI.view.Title != "Help" {
		t.Fatalf("expected request edit help title, got %q", updated.helpUI.view.Title)
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated = updatedModel.(*Model)
	if updated.helpOverlayOpen() {
		t.Fatal("expected esc to close help overlay first")
	}
	if updated.viewState.RequestEditKind != model.RequestEditKindField {
		t.Fatalf("expected request edit to remain active after closing help, got %q", updated.viewState.RequestEditKind)
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated = updatedModel.(*Model)
	if updated.viewState.RequestEditKind != model.RequestEditKindNone {
		t.Fatalf("expected second esc to cancel request edit, got %q", updated.viewState.RequestEditKind)
	}
}

func TestModelThemeKeysDoNotInterfereWithRequestEditing(t *testing.T) {
	t.Parallel()

	original := widgets.CurrentTheme()
	t.Cleanup(func() {
		widgets.SetTheme(original)
	})
	widgets.SetThemeByName("default")

	m := newLoadedModelForNavigation()
	m.viewState.FocusedPane = model.FocusedPaneRequest
	m.panes.activeRequestSection = "Path"

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := updatedModel.(*Model)
	if updated.viewState.RequestEditKind != model.RequestEditKindField {
		t.Fatalf("expected field edit mode, got %q", updated.viewState.RequestEditKind)
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	updated = updatedModel.(*Model)
	if widgets.CurrentTheme().Name != "default" {
		t.Fatalf("expected theme to remain unchanged during request edit, got %q", widgets.CurrentTheme().Name)
	}
	if updated.viewState.RequestEditBuffer != "t" {
		t.Fatalf("expected theme key to be treated as request input, got %q", updated.viewState.RequestEditBuffer)
	}
}

func TestModelQuestionMarkOpensFilterHelpAndBlocksEditingUntilClosed(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForNavigation()

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	updated := updatedModel.(*Model)
	if updated.viewState.ActiveEditorMode != model.EditorModeFilter {
		t.Fatalf("expected filter editing mode, got %q", updated.viewState.ActiveEditorMode)
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	updated = updatedModel.(*Model)
	if !updated.helpOverlayOpen() {
		t.Fatal("expected filter help to open")
	}
	if updated.helpUI.view.Title != "Filter help" {
		t.Fatalf("expected filter help title, got %q", updated.helpUI.view.Title)
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	updated = updatedModel.(*Model)
	if updated.viewState.FilterText != "" {
		t.Fatalf("expected filter text to stay unchanged while help is open, got %q", updated.viewState.FilterText)
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated = updatedModel.(*Model)
	if updated.helpOverlayOpen() {
		t.Fatal("expected esc to close filter help")
	}
	if updated.viewState.ActiveEditorMode != model.EditorModeFilter {
		t.Fatalf("expected filter edit mode to remain active after closing help, got %q", updated.viewState.ActiveEditorMode)
	}
}

func TestModelThemeKeysDoNotInterfereWithFilterEditing(t *testing.T) {
	t.Parallel()

	original := widgets.CurrentTheme()
	t.Cleanup(func() {
		widgets.SetTheme(original)
	})
	widgets.SetThemeByName("default")

	m := newLoadedModelForNavigation()

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	updated := updatedModel.(*Model)
	if updated.viewState.ActiveEditorMode != model.EditorModeFilter {
		t.Fatalf("expected filter editing mode, got %q", updated.viewState.ActiveEditorMode)
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	updated = updatedModel.(*Model)
	if widgets.CurrentTheme().Name != "default" {
		t.Fatalf("expected theme to remain unchanged during filter edit, got %q", widgets.CurrentTheme().Name)
	}
	if updated.viewState.FilterText != "t" {
		t.Fatalf("expected theme key to be treated as filter input, got %q", updated.viewState.FilterText)
	}
}

func TestModelQuestionMarkOpensHistoryHelpAndKeepsPopupOpen(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForNavigation()
	m.session.RequestHistory = []model.HistoryEntry{{
		RequestID:     7,
		OperationKey:  model.NewOperationKey("GET", "/pets"),
		ServerURL:     "https://api.example.com",
		Request:       model.ExecutedRequestSnapshot{ServerURL: "https://api.example.com"},
		Response:      &model.HTTPResponse{Status: "200 OK"},
		TransportNote: "",
	}}

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	updated := updatedModel.(*Model)
	if !updated.historyPopupOpen() {
		t.Fatal("expected history popup to open")
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	updated = updatedModel.(*Model)
	if !updated.helpOverlayOpen() {
		t.Fatal("expected history help to open")
	}
	if updated.helpUI.view.Title != "History help" {
		t.Fatalf("expected history help title, got %q", updated.helpUI.view.Title)
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated = updatedModel.(*Model)
	if !updated.historyPopupOpen() {
		t.Fatal("expected history popup to remain open after closing help")
	}
}

func TestModelThemeKeysStillWorkFromHistoryPopup(t *testing.T) {
	t.Parallel()

	original := widgets.CurrentTheme()
	t.Cleanup(func() {
		widgets.SetTheme(original)
	})
	widgets.SetThemeByName("default")

	m := newLoadedModelForNavigation()
	m.session.RequestHistory = []model.HistoryEntry{{
		RequestID:     7,
		OperationKey:  model.NewOperationKey("GET", "/pets"),
		ServerURL:     "https://api.example.com",
		Request:       model.ExecutedRequestSnapshot{ServerURL: "https://api.example.com"},
		Response:      &model.HTTPResponse{Status: "200 OK"},
		TransportNote: "",
	}}

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	updated := updatedModel.(*Model)
	if !updated.historyPopupOpen() {
		t.Fatal("expected history popup to open")
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	updated = updatedModel.(*Model)
	if widgets.CurrentTheme().Name != "harbor" {
		t.Fatalf("expected theme cycling to work from history popup, got %q", widgets.CurrentTheme().Name)
	}
	if updated.viewState.Notice != "Theme: harbor" {
		t.Fatalf("expected history popup theme notice, got %q", updated.viewState.Notice)
	}
	if !updated.historyPopupOpen() {
		t.Fatal("expected history popup to stay open after theme switch")
	}
}

func TestModelQuestionMarkOpensBlockingLoadErrorHelp(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForNavigation()
	m.shell.loadErr = errors.New("boom")

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	updated := updatedModel.(*Model)
	if !updated.helpOverlayOpen() {
		t.Fatal("expected blocking load error help to open")
	}
	if updated.helpUI.view.Title != "Load error help" {
		t.Fatalf("expected load error help title, got %q", updated.helpUI.view.Title)
	}
}

func TestModelRequestDraftPersistsAcrossOperationAndFilterChanges(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForNavigation()
	m.viewState.FocusedPane = model.FocusedPaneRequest
	m.panes.activeRequestSection = "Path"

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
	m.shell.width = 80
	m.shell.height = 12
	m.viewState.RightPaneLayoutPreset = layoutPresetNarrow
	m.viewState.FocusedPane = model.FocusedPaneRequest
	m.panes.activeRequestSection = "Query"

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
		shell: shellState{
			width:  80,
			height: 12,
		},
		panes: paneState{
			activeDetailsSection: detailsui.SectionSummary,
		},
	}
	m.viewState.RightPaneLayoutPreset = chooseLayoutPreset(m.shell.width)

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	updated := updatedModel.(*Model)
	if updated.viewState.DetailsScrollOffset == 0 {
		t.Fatal("expected details scroll offset to increase with j")
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	updated = updatedModel.(*Model)
	if updated.panes.activeDetailsSection != detailsui.SectionSecurity {
		t.Fatalf("expected ] to switch to security, got %q", updated.panes.activeDetailsSection)
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
		shell: shellState{
			width:  80,
			height: 12,
		},
		panes: paneState{
			activeDetailsSection: detailsui.SectionSummary,
		},
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

func TestModelResponseHomeAndEndControlScroll(t *testing.T) {
	t.Parallel()

	m := &Model{
		session: model.SessionState{
			Spec: &model.APISpec{
				Operations: []model.Operation{
					{
						Key:       model.NewOperationKey("GET", "/pets"),
						Method:    "GET",
						Path:      "/pets",
						Responses: []model.ResponseSpec{{StatusCode: "200", Description: "OK"}},
					},
				},
			},
			SelectedOperationKey: model.NewOperationKey("GET", "/pets"),
			LastResponse: &model.HTTPResponse{
				OperationKey: model.NewOperationKey("GET", "/pets"),
				Status:       "200 OK",
				PrettyBody:   "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8",
			},
		},
		viewState: model.ViewState{
			FocusedPane:           model.FocusedPaneResponse,
			ResponseScrollOffset:  2,
			RightPaneLayoutPreset: layoutPresetNarrow,
		},
		shell: shellState{
			width:  80,
			height: 12,
		},
		panes: paneState{
			activeResponseSection: "Live",
		},
	}

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnd})
	updated := updatedModel.(*Model)
	if updated.viewState.ResponseScrollOffset == 0 {
		t.Fatal("expected end to jump to the bottom of response content")
	}

	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyHome})
	updated = updatedModel.(*Model)
	if updated.viewState.ResponseScrollOffset != 0 {
		t.Fatalf("expected home to jump to top of response content, got %d", updated.viewState.ResponseScrollOffset)
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
	m.shell.height = 18
	m.shell.width = 80
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
	if m.viewState.OperationsScrollOffset != 4 {
		t.Fatalf("expected scroll offset 4 with the fixed single-line status bar, got %d", m.viewState.OperationsScrollOffset)
	}
}

func newLoadedModelForNavigation() *Model {
	spec := &model.APISpec{
		SecuritySchemes: map[string]model.SecurityScheme{
			"api_key": {
				Name:          "api_key",
				Type:          model.SecuritySchemeTypeAPIKey,
				In:            model.ParameterLocationHeader,
				ParameterName: "X-API-Key",
			},
			"global_auth": {
				Name:   "global_auth",
				Type:   model.SecuritySchemeTypeHTTP,
				Scheme: model.HTTPAuthSchemeBearer,
			},
		},
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
		panes: paneState{
			activeDetailsSection:  detailsui.SectionSummary,
			activeRequestSection:  "Path",
			activeResponseSection: "200",
		},
		session: model.SessionState{
			Spec:                 spec,
			SelectedOperationKey: model.NewOperationKey("GET", "/pets"),
			AuthState:            map[string]model.AuthValue{},
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
