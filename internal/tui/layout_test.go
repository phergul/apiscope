package tui

import (
	"errors"
	"strings"
	"testing"

	"api-tui/internal/model"
)

func TestOperationsPaneContentHighlightsSelectedOperationAndPreservesOrder(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForRendering()

	content := m.operationsPaneContent()

	firstGroup := strings.Index(content, "PETS")
	secondGroup := strings.Index(content, "ADMIN")
	selected := strings.Index(content, "> GET    /pets")
	second := strings.Index(content, "  POST   /pets")
	if firstGroup == -1 || secondGroup == -1 {
		t.Fatalf("expected grouped operations list, got %q", content)
	}
	if selected == -1 || second == -1 {
		t.Fatalf("expected operations list to contain selected and unselected rows, got %q", content)
	}
	if firstGroup > secondGroup || selected > second {
		t.Fatalf("expected operations to preserve visible order, got %q", content)
	}
	if strings.Contains(content, "List pets") || strings.Contains(content, "Create pet") {
		t.Fatalf("expected operations rows to omit summaries, got %q", content)
	}
}

func TestOperationsPaneContentFallsBackToFirstVisibleWhenSelectionMissing(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForRendering()
	m.session.SelectedOperationKey = model.NewOperationKey("DELETE", "/missing")

	content := m.operationsPaneContent()

	if !strings.Contains(content, "> GET    /pets") {
		t.Fatalf("expected first visible operation to be highlighted, got %q", content)
	}
}

func TestOperationsPaneContentShowsEmptyState(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForRendering()
	m.session.Spec.Operations = nil
	m.viewState.VisibleOperationKeys = nil

	content := m.operationsPaneContent()

	if !strings.Contains(content, "does not define any operations") {
		t.Fatalf("expected empty operations state, got %q", content)
	}
}

func TestOperationsPaneContentShowsFilteredEmptyState(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForRendering()
	m.viewState.FilterText = "zzz"
	m.viewState.VisibleOperationKeys = nil

	content := m.operationsPaneContent()

	if !strings.Contains(content, "No operations match the current filter.") {
		t.Fatalf("expected filtered empty state, got %q", content)
	}
	if !strings.Contains(content, "Press Esc to clear the filter.") {
		t.Fatalf("expected filtered empty state to mention Esc, got %q", content)
	}
}

func TestDetailsPaneContentRendersSummarySection(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForRendering()
	m.activeDetailsSection = detailsSectionSummary

	content := m.detailsPaneContent()

	wantSnippets := []string{
		"[Summary]  Security  Warnings",
		"Operation: GET /pets",
		"Summary: List pets",
		"Description: Returns pets.",
		"Tags: pets, public",
		"Deprecated: no",
	}
	for _, snippet := range wantSnippets {
		if !strings.Contains(content, snippet) {
			t.Fatalf("expected details content to include %q, got %q", snippet, content)
		}
	}
}

func TestDetailsPaneContentShowsSecuritySectionWhenActive(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForRendering()
	m.activeDetailsSection = detailsSectionSecurity

	content := m.detailsPaneContent()

	wantSnippets := []string{
		"Summary  [Security]  Warnings",
		"- api_key AND secondary_header",
		"OR",
		"- oauth (pets:read)",
	}
	for _, snippet := range wantSnippets {
		if !strings.Contains(content, snippet) {
			t.Fatalf("expected security section to include %q, got %q", snippet, content)
		}
	}
}

func TestDetailsPaneContentShowsWarningsSectionWhenActive(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForRendering()
	m.activeDetailsSection = detailsSectionWarnings

	content := m.detailsPaneContent()

	wantSnippets := []string{
		"Summary  Security  [Warnings]",
		"- unsupported_feature: callbacks are not supported in v1",
		"  path: #/paths/~1pets/get/callbacks",
		"- downgraded_feature: collectionFormat was simplified during normalization",
	}
	for _, snippet := range wantSnippets {
		if !strings.Contains(content, snippet) {
			t.Fatalf("expected warnings section to include %q, got %q", snippet, content)
		}
	}
}

func TestDetailsPaneContentDoesNotShowRequestOrResponseSections(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForRendering()

	content := m.detailsPaneContent()

	unwantedSnippets := []string{
		"Parameters",
		"Request Body",
		"Responses",
		"PATH:",
		"QUERY:",
		"Required: required",
		"Media types: application/json, application/xml",
		"- 200: OK [application/json]",
	}
	for _, snippet := range unwantedSnippets {
		if strings.Contains(content, snippet) {
			t.Fatalf("expected details pane to omit %q, got %q", snippet, content)
		}
	}
}

func TestDetailsPaneContentUsesTopLevelSecurityFallback(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForRendering()
	m.session.SelectedOperationKey = model.NewOperationKey("POST", "/pets")
	m.syncActiveDetailsSection()
	m.activeDetailsSection = detailsSectionSecurity

	content := m.detailsPaneContent()

	if !strings.Contains(content, "- global_auth") {
		t.Fatalf("expected top-level security fallback, got %q", content)
	}
}

func TestDetailsPaneContentExplainsMissingSelection(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForRendering()
	m.viewState.FilterText = "zzz"
	m.viewState.VisibleOperationKeys = nil
	m.session.SelectedOperationKey = ""

	content := m.detailsPaneContent()

	wantSnippets := []string{
		"No operation selected.",
		"Choose an operation in pane 1",
		"press Esc to clear the filter",
	}
	for _, snippet := range wantSnippets {
		if !strings.Contains(content, snippet) {
			t.Fatalf("expected missing selection copy to include %q, got %q", snippet, content)
		}
	}
}

func TestRenderLoadErrorContentUsesStructuredMessage(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForRendering()
	m.loadErr = errors.New("boom")

	content := m.renderLoadErrorContent()

	wantSnippets := []string{
		"Failed to load spec",
		"Category: load error",
		"Source: demo.yaml",
		"Try this: Check the source and try again.",
	}
	for _, snippet := range wantSnippets {
		if !strings.Contains(content, snippet) {
			t.Fatalf("expected structured load error copy to include %q, got %q", snippet, content)
		}
	}
}

func TestRenderBlockingLoadErrorShowsCenteredQuitPopup(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForRendering()
	m.loadErr = errors.New("boom")

	content := m.render()

	wantSnippets := []string{
		"Failed to load spec",
		"Category: load error",
		"Source: demo.yaml",
		"Try this: Check the source and try again.",
		"[ Quit ]",
	}
	for _, snippet := range wantSnippets {
		if !strings.Contains(content, snippet) {
			t.Fatalf("expected blocking load popup to include %q, got %q", snippet, content)
		}
	}
	if strings.Contains(content, "1 Operations") || strings.Contains(content, "2 Details") {
		t.Fatalf("expected blocking load popup to replace pane layout, got %q", content)
	}
}

func TestRenderZoomLayoutShowsOnlyFocusedPaneAndStatusBar(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForRendering()
	m.width = 120
	m.height = 30
	m.viewState.FocusedPane = model.FocusedPaneResponse
	m.viewState.ExpandedRightPane = model.FocusedPaneResponse
	m.viewState.ZoomedPane = true

	content := m.render()

	if !strings.Contains(content, "> 4 Response") {
		t.Fatalf("expected focused response pane to render in zoom mode, got %q", content)
	}
	if strings.Contains(content, "1 Operations") || strings.Contains(content, "2 Details") || strings.Contains(content, "3 Request") {
		t.Fatalf("expected only the focused pane to render in zoom mode, got %q", content)
	}
	if !strings.Contains(content, "z zoom") || !strings.Contains(content, "q quit") {
		t.Fatalf("expected status bar to remain visible in zoom mode, got %q", content)
	}
}

func TestDetailsPaneContentShowsExplicitNoneStates(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForRendering()
	m.session.SelectedOperationKey = model.NewOperationKey("POST", "/pets")
	m.syncActiveDetailsSection()

	content := m.detailsPaneContent()

	wantSnippets := []string{
		"[Summary]  Security  Warnings",
		"Summary: Create pet",
		"Description: None",
		"Tags: admin",
		"Deprecated: yes",
	}
	for _, snippet := range wantSnippets {
		if !strings.Contains(content, snippet) {
			t.Fatalf("expected details content to include %q, got %q", snippet, content)
		}
	}
}

func TestRequestAndResponsePaneCopyExplainsFutureOwnership(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForRendering()

	requestContent := m.requestPaneContent()
	if !strings.Contains(requestContent, "path/query/header params, auth, and request body input") {
		t.Fatalf("expected request pane copy to explain ownership, got %q", requestContent)
	}

	responseContent := m.responsePaneContent()
	if !strings.Contains(responseContent, "response details and examples") {
		t.Fatalf("expected response pane copy to explain ownership, got %q", responseContent)
	}
}

func TestStatusBarIncludesOperationIdentityAndCount(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForRendering()

	content := m.renderStatusBar(200)

	wantSnippets := []string{
		"Source: demo.yaml",
		"State: loaded",
		"Focus: operations",
		"Operation: GET /pets",
		"Count: 2",
		"Visible: 2",
		"Warnings: 2",
		"Keys: 1-4 switch Tab cycle z zoom q quit",
	}
	for _, snippet := range wantSnippets {
		if !strings.Contains(content, snippet) {
			t.Fatalf("expected status bar to include %q, got %q", snippet, content)
		}
	}
}

func TestComputeWidePaneHeightsPreserveTotalAndExpandedPriority(t *testing.T) {
	t.Parallel()

	heights := computeWidePaneHeights(24)
	if heights.Details+heights.Expanded+heights.Folded != 24 {
		t.Fatalf("expected wide pane heights to preserve total height, got %+v", heights)
	}
	if heights.Expanded <= heights.Folded {
		t.Fatalf("expected expanded pane to be taller than folded pane, got %+v", heights)
	}
}

func TestComputeWidePaneHeightsCanCollapseFoldedPaneInCompactMode(t *testing.T) {
	t.Parallel()

	heights := computeWidePaneHeights(10)
	if heights.Folded != 0 {
		t.Fatalf("expected folded pane to collapse first in compact mode, got %+v", heights)
	}
	if heights.Details < 4 {
		t.Fatalf("expected details pane to respect its hard minimum, got %+v", heights)
	}
	if heights.Details+heights.Expanded+heights.Folded != 10 {
		t.Fatalf("expected compact wide heights to preserve total height, got %+v", heights)
	}
}

func TestComputeNarrowPaneHeightsPreserveTotalAcrossPresets(t *testing.T) {
	t.Parallel()

	for _, total := range []int{30, 24, 12} {
		heights := computeNarrowPaneHeights(total)
		if heights.Operations+heights.Details+heights.Expanded+heights.Folded != total {
			t.Fatalf("expected narrow pane heights to preserve total %d, got %+v", total, heights)
		}
		if heights.Expanded < heights.Folded {
			t.Fatalf("expected expanded pane to keep at least as much space as folded pane, got %+v", heights)
		}
		if heights.Operations < 4 || heights.Details < 4 {
			t.Fatalf("expected narrow layout hard minimums to hold, got %+v", heights)
		}
	}
}

func TestRenderWideLayoutKeepsRequestAboveResponseWhenResponseIsExpanded(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForRendering()
	m.width = 120
	m.height = 30
	m.viewState.ExpandedRightPane = model.FocusedPaneResponse

	content := m.render()
	responseIndex := strings.Index(content, "4 Response")
	requestIndex := strings.Index(content, "3 Request")
	if responseIndex == -1 || requestIndex == -1 {
		t.Fatalf("expected request and response panes to render, got %q", content)
	}
	if requestIndex > responseIndex {
		t.Fatalf("expected request pane to remain above response pane, got %q", content)
	}
}

func newLoadedModelForRendering() *Model {
	spec := &model.APISpec{
		Title: "Demo API",
		Operations: []model.Operation{
			{
				Key:         model.NewOperationKey("GET", "/pets"),
				Method:      "GET",
				Path:        "/pets",
				Summary:     "List pets",
				Description: "Returns pets.",
				Tags:        []string{"pets", "public"},
				Parameters: []model.Parameter{
					{Name: "petId", In: model.ParameterLocationPath, Required: true, Schema: &model.Schema{Type: "string"}},
					{Name: "limit", In: model.ParameterLocationQuery, Schema: &model.Schema{Type: "integer", Format: "int32"}},
					{Name: "X-Trace-ID", In: model.ParameterLocationHeader, Schema: &model.Schema{Type: "string"}},
					{Name: "session", In: model.ParameterLocationCookie, Schema: &model.Schema{Type: "string"}},
				},
				RequestBody: &model.RequestBodySpec{
					Required:    true,
					Description: "Pet filter payload",
					Content: []model.MediaTypeSpec{
						{MediaType: "application/json"},
						{MediaType: "application/xml"},
					},
				},
				Responses: []model.ResponseSpec{
					{StatusCode: "200", Description: "OK", Content: []model.MediaTypeSpec{{MediaType: "application/json"}}},
					{StatusCode: "default", Description: "Unexpected error", Content: []model.MediaTypeSpec{{MediaType: "application/problem+json"}}},
				},
				Security: &model.SecurityRequirement{
					Alternatives: []model.SecurityAlternative{
						{Schemes: []model.SecurityRequirementRef{{Name: "api_key"}, {Name: "secondary_header"}}},
						{Schemes: []model.SecurityRequirementRef{{Name: "oauth", Scopes: []string{"pets:read"}}}},
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
		},
		Security: &model.SecurityRequirement{
			Alternatives: []model.SecurityAlternative{
				{Schemes: []model.SecurityRequirementRef{{Name: "global_auth"}}},
			},
		},
		Warnings: []model.SpecWarning{
			{
				Code:    model.SpecWarningUnsupportedFeature,
				Message: "callbacks are not supported in v1",
				Path:    "#/paths/~1pets/get/callbacks",
			},
			{
				Code:    model.SpecWarningDowngradedFeature,
				Message: "collectionFormat was simplified during normalization",
			},
		},
	}

	return &Model{
		source:               "demo.yaml",
		activeDetailsSection: detailsSectionSummary,
		session: model.SessionState{
			Spec:                 spec,
			SelectedOperationKey: model.NewOperationKey("GET", "/pets"),
		},
		viewState: model.ViewState{
			FocusedPane:          model.FocusedPaneOperations,
			ExpandedRightPane:    model.FocusedPaneRequest,
			VisibleOperationKeys: []model.OperationKey{model.NewOperationKey("GET", "/pets"), model.NewOperationKey("POST", "/pets")},
		},
	}
}
