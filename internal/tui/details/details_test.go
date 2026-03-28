package details

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/phergul/apiscope/internal/model"
)

func TestRenderRendersSummarySection(t *testing.T) {
	t.Parallel()

	content := ansi.Strip(Render(newTestData()))

	wantSnippets := []string{
		"Summary  Security  Warnings",
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
	if strings.Contains(content, "Operation: GET /pets") {
		t.Fatalf("expected details summary to omit redundant operation row, got %q", content)
	}
}

func TestRenderShowsSecuritySectionWhenActive(t *testing.T) {
	t.Parallel()

	data := newTestData()
	data.ActiveSection = SectionSecurity

	content := ansi.Strip(Render(data))

	wantSnippets := []string{
		"Summary  Security  Warnings",
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

func TestRenderShowsWarningsSectionWhenActive(t *testing.T) {
	t.Parallel()

	data := newTestData()
	data.ActiveSection = SectionWarnings

	content := ansi.Strip(Render(data))

	wantSnippets := []string{
		"Summary  Security  Warnings",
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

func TestRenderDoesNotShowRequestOrResponseSections(t *testing.T) {
	t.Parallel()

	content := ansi.Strip(Render(newTestData()))

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

func TestRenderExplainsMissingSelection(t *testing.T) {
	t.Parallel()

	content := ansi.Strip(Render(Data{
		FilterText: "zzz",
	}))

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

func TestRenderShowsExplicitNoneStates(t *testing.T) {
	t.Parallel()

	data := newTestData()
	data.Selected = &model.Operation{
		Method:     "POST",
		Path:       "/pets",
		Summary:    "Create pet",
		Tags:       []string{"admin"},
		Deprecated: true,
	}

	content := ansi.Strip(Render(data))

	wantSnippets := []string{
		"Summary  Security  Warnings",
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

func newTestData() Data {
	return Data{
		Selected: &model.Operation{
			Method:      "GET",
			Path:        "/pets",
			Summary:     "List pets",
			Description: "Returns pets.",
			Tags:        []string{"pets", "public"},
		},
		ActiveSection: SectionSummary,
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
