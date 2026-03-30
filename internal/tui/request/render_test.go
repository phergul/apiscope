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
		ContentWidth:  80,
		ContentHeight: 12,
		Edit: EditView{
			Kind:      "body",
			MediaType: "application/json",
			Buffer:    "{\n  \"name\": \"fido\"\n}",
			View:      "{\n  \"name\": \"fido\"\n}",
			Title:     "Edit body",
			Context:   "Media type: application/json",
			Meta:      "Help - ?",
		},
	}))

	wantSnippets := []string{
		"Path  Body  Auth",
		"Edit body",
		"Media type: application/json",
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

func TestRenderShowsFieldEditorAsPopupWithoutDefaultControls(t *testing.T) {
	t.Parallel()

	content := ansi.Strip(Render(Data{
		Sections:      []string{"Query", "Auth"},
		ActiveSection: "Query",
		ContentWidth:  80,
		ContentHeight: 12,
		Rows: []Row{
			{Label: "limit", Meta: "optional, integer", Value: "<unset>", Editable: true},
			{Label: "offset", Meta: "optional, integer", Value: "<unset>", Editable: true},
		},
		Edit: EditView{
			Kind:    "field",
			View:    "42",
			Title:   "Edit value",
			Context: "limit (optional, integer)",
			Meta:    "Help - ?",
		},
	}))

	for _, snippet := range []string{"Edit value", "limit (optional, integer)", "42"} {
		if !strings.Contains(content, snippet) {
			t.Fatalf("expected request popup to include %q, got %q", snippet, content)
		}
	}
	if strings.Contains(content, "Help - ?") {
		t.Fatalf("expected help hint to stay out of the editor popup, got %q", content)
	}
	if strings.Contains(content, "Enter save") {
		t.Fatalf("expected controls to stay hidden by default, got %q", content)
	}
}

func TestRenderShowsPopupHelpWhenEnabled(t *testing.T) {
	t.Parallel()

	content := ansi.Strip(Render(Data{
		Sections:      []string{"Query"},
		ActiveSection: "Query",
		ContentWidth:  80,
		ContentHeight: 12,
		Rows: []Row{
			{Label: "limit", Meta: "optional, integer", Value: "<unset>", Editable: true},
		},
		Edit: EditView{
			Kind:     "field",
			View:     "42",
			Title:    "Edit value",
			Context:  "limit (optional, integer)",
			Meta:     "Help - ?",
			Help:     "Enter save\nEsc cancel\n? toggle help",
			ShowHelp: true,
		},
	}))

	for _, snippet := range []string{"Enter save", "Esc cancel", "? toggle help"} {
		if strings.Contains(content, snippet) {
			t.Fatalf("expected help popup content to stay out of request pane rendering, got %q", content)
		}
	}
	if !strings.Contains(content, "Edit value") {
		t.Fatalf("expected main editor popup to remain visible, got %q", content)
	}
}
