package response

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/phergul/apiscope/internal/model"
	"github.com/phergul/apiscope/internal/tui/widgets"
)

func TestRenderShowsDeclaredResponses(t *testing.T) {
	t.Parallel()

	content := ansi.Strip(Render(Data{
		Sections: []widgets.Section{
			{
				Label: "Live",
				Body:  "No request has been sent for this operation yet.",
			},
			{
				Label: "200",
				Body: strings.Join([]string{
					"Description: OK",
					"Headers:",
					"- X-Rate-Limit:",
					"  integer",
					"- X-Trace-ID:",
					"  string",
					"Body:",
					"│ application/json",
				}, "\n"),
			},
			{
				Label: "default",
				Body: strings.Join([]string{
					"Description: Unexpected error",
					"Headers:",
					"- none",
					"Body:",
					"│ application/problem+json",
				}, "\n"),
			},
		},
		ActiveSection: "200",
	}))

	wantSnippets := []string{
		"Live  200  default",
		"Description: OK",
		"Headers:",
		"- X-Rate-Limit:",
		"  integer",
		"- X-Trace-ID:",
		"  string",
		"Body:",
		"│ application/json",
	}
	for _, snippet := range wantSnippets {
		if !strings.Contains(content, snippet) {
			t.Fatalf("expected response pane to include %q, got %q", snippet, content)
		}
	}
}

func TestLiveSectionWrapsLongHeadersAndRendersBodyBlock(t *testing.T) {
	t.Parallel()

	section := LiveSection(&model.HTTPResponse{
		OperationKey: model.NewOperationKey("GET", "/pets"),
		Status:       "401 Unauthorized",
		ContentType:  "application/json",
		Headers: map[string][]string{
			"Access-Control-Allow-Headers": {"Accept", "App-Platform", "Authorization", "Content-Type", "Origin"},
		},
		PrettyBody: "{\n  \"error\": true\n}",
	}, &model.Operation{
		Key: model.NewOperationKey("GET", "/pets"),
	}, 32)

	content := ansi.Strip(section.Body)
	for _, snippet := range []string{
		"Status: ",
		"Headers:",
		"- Access-Control-Allow-Headers:",
		"  Accept, App-Platform,",
		"Body:",
		"│ {",
		"│   \"error\": true",
	} {
		if !strings.Contains(content, snippet) {
			t.Fatalf("expected live response section to include %q, got %q", snippet, content)
		}
	}
}

func TestRenderShowsLiveSectionEmptyState(t *testing.T) {
	t.Parallel()

	content := ansi.Strip(Render(Data{
		Sections: []widgets.Section{
			{
				Label: "Live",
				Body:  "No request has been sent for this operation yet.",
			},
			{
				Label: "200",
				Body:  "Description: OK",
			},
		},
		ActiveSection: "Live",
	}))

	if !strings.Contains(content, "No request has been sent for this operation yet.") {
		t.Fatalf("expected live section placeholder, got %q", content)
	}
}

func TestRenderShowsExplicitEmptyState(t *testing.T) {
	t.Parallel()

	content := ansi.Strip(Render(Data{
		EmptyState: "This operation does not declare any responses.",
	}))

	if !strings.Contains(content, "This operation does not declare any responses.") {
		t.Fatalf("expected response pane empty state, got %q", content)
	}
}

func TestRenderNormalisesEmbeddedDescriptionLineBreaks(t *testing.T) {
	t.Parallel()

	content := ansi.Strip(Render(Data{
		Sections: []widgets.Section{
			{
				Label: "401",
				Body: strings.Join([]string{
					"Description: Bad or expired token. This can happen if the user revoked a token or the access token has expired. You should re-authenticate the user.",
					"Headers:",
					"- none",
					"Body:",
					"│ application/json",
				}, "\n"),
			},
		},
		ActiveSection: "401",
	}))

	if strings.Contains(content, "token or\nthe access token") {
		t.Fatalf("expected response description to collapse embedded line breaks, got %q", content)
	}
}

func TestSectionBodyUsesLiveStyleLayoutForDeclaredResponses(t *testing.T) {
	t.Parallel()

	content := ansi.Strip(sectionBody(model.ResponseSpec{
		Description: "OK",
		Headers: []model.Parameter{
			{Name: "X-Trace-ID", Schema: &model.Schema{Type: "string"}},
		},
		Content: []model.MediaTypeSpec{{MediaType: "application/json"}},
	}))

	for _, snippet := range []string{
		"Description: OK",
		"Headers:",
		"- X-Trace-ID:",
		"  string",
		"Body:",
		"│ application/json",
	} {
		if !strings.Contains(content, snippet) {
			t.Fatalf("expected declared response layout to include %q, got %q", snippet, content)
		}
	}
}
