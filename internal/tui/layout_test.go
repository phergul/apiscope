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
		"[Summary]  Parameters  Request Body  Responses  Security  Warnings",
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

func TestDetailsPaneContentShowsParametersSectionWhenActive(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForRendering()
	m.activeDetailsSection = detailsSectionParameters

	content := m.detailsPaneContent()

	wantSnippets := []string{
		"Summary  [Parameters]  Request Body  Responses  Security  Warnings",
		"PATH:",
		"- petId (required, string)",
		"QUERY:",
		"- limit (optional, integer/int32)",
		"HEADER:",
		"- X-Trace-ID (optional, string)",
		"COOKIE:",
		"- session (optional, string)",
	}
	for _, snippet := range wantSnippets {
		if !strings.Contains(content, snippet) {
			t.Fatalf("expected details parameters section to include %q, got %q", snippet, content)
		}
	}
}

func TestDetailsPaneContentShowsRequestBodySectionWhenActive(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForRendering()
	m.activeDetailsSection = detailsSectionRequestBody

	content := m.detailsPaneContent()

	wantSnippets := []string{
		"Summary  Parameters  [Request Body]  Responses  Security  Warnings",
		"Required: required",
		"Media types: application/json, application/xml",
	}
	for _, snippet := range wantSnippets {
		if !strings.Contains(content, snippet) {
			t.Fatalf("expected request body section to include %q, got %q", snippet, content)
		}
	}
}

func TestDetailsPaneContentShowsResponsesSectionWhenActive(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForRendering()
	m.activeDetailsSection = detailsSectionResponses

	content := m.detailsPaneContent()

	wantSnippets := []string{
		"Summary  Parameters  Request Body  [Responses]  Security  Warnings",
		"- 200: OK [application/json]",
		"- default: Unexpected error [application/problem+json]",
	}
	for _, snippet := range wantSnippets {
		if !strings.Contains(content, snippet) {
			t.Fatalf("expected responses section to include %q, got %q", snippet, content)
		}
	}
}

func TestDetailsPaneContentShowsSecuritySectionWhenActive(t *testing.T) {
	t.Parallel()

	m := newLoadedModelForRendering()
	m.activeDetailsSection = detailsSectionSecurity

	content := m.detailsPaneContent()

	wantSnippets := []string{
		"Summary  Parameters  Request Body  Responses  [Security]  Warnings",
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
		"Summary  Parameters  Request Body  Responses  Security  [Warnings]",
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
		"Keys: 1-4 switch Tab cycle q quit",
	}
	for _, snippet := range wantSnippets {
		if !strings.Contains(content, snippet) {
			t.Fatalf("expected status bar to include %q, got %q", snippet, content)
		}
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
			VisibleOperationKeys: []model.OperationKey{model.NewOperationKey("GET", "/pets"), model.NewOperationKey("POST", "/pets")},
		},
	}
}
