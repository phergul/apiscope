package request

import (
	"strings"
	"testing"
)

func TestBuildEditViewForFieldUsesRowContext(t *testing.T) {
	t.Parallel()

	view := BuildEditView(EditorState{
		Kind:           "field",
		Buffer:         "42",
		View:           "42",
		ActiveRowLabel: "limit",
		ActiveRowMeta:  "optional, integer",
	})

	if view.Title != "Edit value" {
		t.Fatalf("expected field editor title, got %q", view.Title)
	}
	if view.Context != "limit (optional, integer)" {
		t.Fatalf("expected field editor context, got %q", view.Context)
	}
}

func TestBuildEditViewForBodyUsesMediaTypeContext(t *testing.T) {
	t.Parallel()

	view := BuildEditView(EditorState{
		Kind:          "body",
		Buffer:        "{}",
		View:          "{}",
		BodyMediaType: "application/json",
	})

	if view.Title != "Edit body" {
		t.Fatalf("expected body editor title, got %q", view.Title)
	}
	if view.Context != "Media type: application/json" {
		t.Fatalf("expected body editor context, got %q", view.Context)
	}
}

func TestBuildEditHelpViewUsesNamedHelpContent(t *testing.T) {
	t.Parallel()

	help := BuildEditHelpView(EditorState{Kind: "field"})
	if help.Hint != "Help - ?" {
		t.Fatalf("expected help hint, got %q", help.Hint)
	}
	if help.Title != "Help" {
		t.Fatalf("expected help title, got %q", help.Title)
	}
	if help.Body != "Enter save\nEsc cancel\n? toggle help" {
		t.Fatalf("expected field help body, got %q", help.Body)
	}
}

func TestBuildBrowseHelpViewUsesBrowseControls(t *testing.T) {
	t.Parallel()

	help := BuildBrowseHelpView()
	if help.Title != "Request help" {
		t.Fatalf("expected request browse help title, got %q", help.Title)
	}
	for _, snippet := range []string{"Ctrl+R send request", "Enter edit value or cycle option"} {
		if !strings.Contains(help.Body, snippet) {
			t.Fatalf("expected request browse help to include %q, got %q", snippet, help.Body)
		}
	}
}
