package operations

import (
	"strings"
	"testing"

	"github.com/phergul/apiscope/internal/model"

	tea "github.com/charmbracelet/bubbletea"
)

func TestFilterVisibleKeysMatchesMethodPathSummaryAndTags(t *testing.T) {
	t.Parallel()

	operations := []model.Operation{
		{Key: model.NewOperationKey("GET", "/pets"), Method: "GET", Path: "/pets", Summary: "List pets", Tags: []string{"public"}},
		{Key: model.NewOperationKey("POST", "/pets"), Method: "POST", Path: "/pets", Summary: "Create pet", Tags: []string{"admin"}},
	}

	if got := FilterVisibleKeys(operations, "post"); len(got) != 1 || got[0] != model.NewOperationKey("POST", "/pets") {
		t.Fatalf("expected method filter to match POST /pets, got %#v", got)
	}
	if got := FilterVisibleKeys(operations, "list"); len(got) != 1 || got[0] != model.NewOperationKey("GET", "/pets") {
		t.Fatalf("expected summary filter to match GET /pets, got %#v", got)
	}
	if got := FilterVisibleKeys(operations, "admin"); len(got) != 1 || got[0] != model.NewOperationKey("POST", "/pets") {
		t.Fatalf("expected tag filter to match POST /pets, got %#v", got)
	}
}

func TestRenderFooterUsesEditorViewOnlyWhileEditing(t *testing.T) {
	t.Parallel()

	footer := RenderFooter(FilterFooterInput{
		Editing:    true,
		FilterText: "pets",
		EditorView: "editor view",
	})
	if !strings.Contains(footer, "editor view") {
		t.Fatalf("expected editing footer to render editor view, got %q", footer)
	}

	footer = RenderFooter(FilterFooterInput{FilterText: "pets"})
	if !strings.Contains(footer, "Filter: pets") {
		t.Fatalf("expected idle footer to render filter summary, got %q", footer)
	}
}

func TestUpdateFilterEditorHandlesExitAndTrimKeys(t *testing.T) {
	t.Parallel()

	update := UpdateFilterEditor(tea.KeyMsg{Type: tea.KeyBackspace}, "pets")
	if update.FilterText != "pet" || !update.Editing || !update.RefreshVisible {
		t.Fatalf("expected backspace to trim text and stay editing, got %+v", update)
	}

	update = UpdateFilterEditor(tea.KeyMsg{Type: tea.KeyEnter}, "pets")
	if update.Editing {
		t.Fatalf("expected enter to exit filter mode, got %+v", update)
	}

	update = UpdateFilterEditor(tea.KeyMsg{Type: tea.KeyEsc}, "pets")
	if update.FilterText != "" || update.Editing || !update.RefreshVisible {
		t.Fatalf("expected esc to clear and exit filter mode, got %+v", update)
	}
}
