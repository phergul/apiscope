package request

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestRenderShowsGroupedInputsAndAuthSummary(t *testing.T) {
	t.Parallel()

	content := ansi.Strip(Render(Data{
		Sections:      []string{"Path", "Query", "Body", "Auth"},
		ActiveSection: "Path",
		Rows: []Row{
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
	}))

	wantSnippets := []string{
		"Path  Query  Body  Auth",
		" petId (required, string) = <unset>",
		"legacy (optional, content) = <unsupported: content-based parameter> [read-only]",
	}
	for _, snippet := range wantSnippets {
		if !strings.Contains(content, snippet) {
			t.Fatalf("expected request pane to include %q, got %q", snippet, content)
		}
	}
}

func TestRenderShowsExplicitEmptyState(t *testing.T) {
	t.Parallel()

	content := ansi.Strip(Render(Data{
		EmptyState: "This operation does not declare request parameters, request body, or auth requirements.",
	}))

	if !strings.Contains(content, "This operation does not declare request parameters, request body, or auth requirements.") {
		t.Fatalf("expected request pane empty state, got %q", content)
	}
}

func TestRenderShowsBodyEditorState(t *testing.T) {
	t.Parallel()

	content := ansi.Strip(Render(Data{
		Sections:      []string{"Path", "Body", "Auth"},
		ActiveSection: "Body",
		Edit: EditView{
			Kind:      "body",
			MediaType: "application/json",
			Buffer:    "{\n  \"name\": \"fido\"\n}",
		},
	}))

	wantSnippets := []string{
		"Path  Body  Auth",
		"Media type: application/json",
		"Ctrl+S save | Esc cancel",
		"  \"name\": \"fido\"",
	}
	for _, snippet := range wantSnippets {
		if !strings.Contains(content, snippet) {
			t.Fatalf("expected body editor state to include %q, got %q", snippet, content)
		}
	}
}

func TestRenderShowsInlineValidationState(t *testing.T) {
	t.Parallel()

	content := ansi.Strip(Render(Data{
		Sections:         []string{"Path", "Body"},
		ActiveSection:    "Path",
		ValidationNotice: []string{"Required value missing."},
		Rows: []Row{
			{
				Label:    "petId",
				Meta:     "required, string",
				Value:    "<unset>",
				Editable: true,
				Error:    "Required value missing.",
			},
		},
		ActiveRow: 0,
	}))

	wantSnippets := []string{
		"Validation:",
		"- Required value missing.",
		"petId (required, string) = <unset>",
	}
	for _, snippet := range wantSnippets {
		if !strings.Contains(content, snippet) {
			t.Fatalf("expected request validation content to include %q, got %q", snippet, content)
		}
	}
}
