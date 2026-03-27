package panes

import (
	"strings"
	"testing"

	"api-tui/internal/model"
)

func TestRenderDetailsRendersSummarySection(t *testing.T) {
	t.Parallel()

	content := RenderDetails(newDetailsData())

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

func TestRenderDetailsShowsSecuritySectionWhenActive(t *testing.T) {
	t.Parallel()

	data := newDetailsData()
	data.ActiveSection = DetailsSectionSecurity

	content := RenderDetails(data)

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

func TestRenderDetailsShowsWarningsSectionWhenActive(t *testing.T) {
	t.Parallel()

	data := newDetailsData()
	data.ActiveSection = DetailsSectionWarnings

	content := RenderDetails(data)

	wantSnippets := []string{
		"Summary  Security  [Warnings]",
		"- unsupported_feature: callbacks are not supported in v1",
		"  path: #/paths/~1pets/get/callbacks",
		"- downgraded_feature: collectionFormat was simplified during normalisation",
	}
	for _, snippet := range wantSnippets {
		if !strings.Contains(content, snippet) {
			t.Fatalf("expected warnings section to include %q, got %q", snippet, content)
		}
	}
}

func TestRenderDetailsDoesNotShowRequestOrResponseSections(t *testing.T) {
	t.Parallel()

	content := RenderDetails(newDetailsData())

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

func TestRenderDetailsExplainsMissingSelection(t *testing.T) {
	t.Parallel()

	content := RenderDetails(DetailsData{
		FilterText: "zzz",
	})

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

func TestRenderDetailsShowsExplicitNoneStates(t *testing.T) {
	t.Parallel()

	data := newDetailsData()
	data.Selected = &model.Operation{
		Method:     "POST",
		Path:       "/pets",
		Summary:    "Create pet",
		Tags:       []string{"admin"},
		Deprecated: true,
	}

	content := RenderDetails(data)

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

func newDetailsData() DetailsData {
	return DetailsData{
		Selected: &model.Operation{
			Method:      "GET",
			Path:        "/pets",
			Summary:     "List pets",
			Description: "Returns pets.",
			Tags:        []string{"pets", "public"},
		},
		Sections:      []string{DetailsSectionSummary, DetailsSectionSecurity, DetailsSectionWarnings},
		ActiveSection: DetailsSectionSummary,
		Security: &model.SecurityRequirement{
			Alternatives: []model.SecurityAlternative{
				{Schemes: []model.SecurityRequirementRef{{Name: "api_key"}, {Name: "secondary_header"}}},
				{Schemes: []model.SecurityRequirementRef{{Name: "oauth", Scopes: []string{"pets:read"}}}},
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
}
