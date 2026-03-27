package tui

import (
	"errors"
	"strings"
	"testing"

	"api-tui/internal/model"
)

func TestOperationsPaneContentFallsBackToFirstVisibleWhenSelectionMissing(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForRendering()
	m.session.SelectedOperationKey = model.NewOperationKey("DELETE", "/missing")

	content := m.operationsPaneContent()

	if !strings.Contains(content, "> GET    /pets") {
		t.Fatalf("expected first visible operation to be highlighted, got %q", content)
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
				Message: "collectionFormat was simplified during normalisation",
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
