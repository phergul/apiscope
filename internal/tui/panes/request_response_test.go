package panes

import (
	"strings"
	"testing"

	"github.com/phergul/apiscope/internal/tui/widgets"
)

func TestRenderRequestShowsGroupedInputsAndAuthSummary(t *testing.T) {
	t.Parallel()

	requestContent := RenderRequest(RequestData{
		Sections:      []string{"Path", "Query", "Body", "Auth"},
		ActiveSection: "Path",
		Rows: []RequestRow{
			{
				Label:    "petId",
				Meta:     "required, string",
				Value:    "<unset>",
				Editable: true,
			},
			{
				Label:    "legacy",
				Meta:     "optional, content",
				Value:    "<unsupported: content-based parameter>",
				Editable: false,
			},
		},
		ActiveRow: 0,
	})
	requestContent = stripANSI(requestContent)

	wantRequestSnippets := []string{
		"Path  Query  Body  Auth",
		" petId (required, string) = <unset>",
		"legacy (optional, content) = <unsupported: content-based parameter> [read-only]",
	}
	for _, snippet := range wantRequestSnippets {
		if !strings.Contains(requestContent, snippet) {
			t.Fatalf("expected request pane to include %q, got %q", snippet, requestContent)
		}
	}
}

func TestRenderRequestShowsExplicitEmptyStates(t *testing.T) {
	t.Parallel()

	requestContent := RenderRequest(RequestData{
		EmptyState: "This operation does not declare request parameters, request body, or auth requirements.",
	})
	requestContent = stripANSI(requestContent)

	if !strings.Contains(requestContent, "This operation does not declare request parameters, request body, or auth requirements.") {
		t.Fatalf("expected request pane empty state, got %q", requestContent)
	}
}

func TestRenderRequestShowsBodyEditorState(t *testing.T) {
	t.Parallel()

	requestContent := RenderRequest(RequestData{
		Sections:      []string{"Path", "Body", "Auth"},
		ActiveSection: "Body",
		Edit: RequestEditView{
			Kind:      "body",
			MediaType: "application/json",
			Buffer:    "{\n  \"name\": \"fido\"\n}",
		},
	})
	requestContent = stripANSI(requestContent)

	wantRequestSnippets := []string{
		"Path  Body  Auth",
		"Media type: application/json",
		"Ctrl+S save | Esc cancel",
		"  \"name\": \"fido\"",
	}
	for _, snippet := range wantRequestSnippets {
		if !strings.Contains(requestContent, snippet) {
			t.Fatalf("expected body editor state to include %q, got %q", snippet, requestContent)
		}
	}
}

func TestRenderResponseShowsDeclaredResponses(t *testing.T) {
	t.Parallel()

	responseContent := RenderResponse(ResponseData{
		Sections: []widgets.Section{
			{
				Label: "200",
				Body: strings.Join([]string{
					"Description: OK",
					"Headers:",
					"- X-Rate-Limit (integer)",
					"- X-Trace-ID (string)",
					"Media types: application/json",
				}, "\n"),
			},
			{
				Label: "default",
				Body: strings.Join([]string{
					"Description: Unexpected error",
					"Headers:",
					"- none",
					"Media types: application/problem+json",
				}, "\n"),
			},
		},
		ActiveSection: "200",
	})
	responseContent = stripANSI(responseContent)

	wantResponseSnippets := []string{
		"200  default",
		"Description: OK",
		"Headers:",
		"- X-Rate-Limit (integer)",
		"- X-Trace-ID (string)",
		"Media types: application/json",
	}
	for _, snippet := range wantResponseSnippets {
		if !strings.Contains(responseContent, snippet) {
			t.Fatalf("expected response pane to include %q, got %q", snippet, responseContent)
		}
	}
}

func TestRenderResponseShowsExplicitEmptyState(t *testing.T) {
	t.Parallel()

	responseContent := RenderResponse(ResponseData{
		EmptyState: "This operation does not declare any responses.",
	})
	responseContent = stripANSI(responseContent)

	if !strings.Contains(responseContent, "This operation does not declare any responses.") {
		t.Fatalf("expected response pane empty state, got %q", responseContent)
	}
}

func TestRenderResponseNormalisesEmbeddedDescriptionLineBreaks(t *testing.T) {
	t.Parallel()

	responseContent := RenderResponse(ResponseData{
		Sections: []widgets.Section{
			{
				Label: "401",
				Body: strings.Join([]string{
					"Description: Bad or expired token. This can happen if the user revoked a token or the access token has expired. You should re-authenticate the user.",
					"Headers:",
					"- none",
					"Media types: application/json",
				}, "\n"),
			},
		},
		ActiveSection: "401",
	})
	responseContent = stripANSI(responseContent)

	if strings.Contains(responseContent, "token or\nthe access token") {
		t.Fatalf("expected response description to collapse embedded line breaks, got %q", responseContent)
	}
}
