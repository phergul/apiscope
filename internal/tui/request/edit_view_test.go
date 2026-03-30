package request

import "testing"

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

func TestBuildHelpViewUsesNamedHelpContent(t *testing.T) {
	t.Parallel()

	help := BuildHelpView(EditorState{Kind: "field"})
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
