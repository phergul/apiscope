package operations

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestRenderHighlightsSelectedOperationAndPreservesOrder(t *testing.T) {
	t.Parallel()

	content := ansi.Strip(Render(Data{
		HasSpec:         true,
		TotalOperations: 2,
		Groups: []Group{
			{
				Name: "pets",
				Rows: []Row{
					{Method: "GET", Path: "/pets", Selected: true},
					{Method: "POST", Path: "/pets"},
				},
			},
			{
				Name: "admin",
				Rows: []Row{
					{Method: "DELETE", Path: "/pets/{id}"},
				},
			},
		},
	}))

	firstGroup := strings.Index(content, "PETS")
	secondGroup := strings.Index(content, "ADMIN")
	selected := strings.Index(content, " GET    /pets")
	second := strings.Index(content, " POST   /pets")
	if firstGroup == -1 || secondGroup == -1 {
		t.Fatalf("expected grouped operations list, got %q", content)
	}
	if selected == -1 || second == -1 {
		t.Fatalf("expected operations list to contain selected and unselected rows, got %q", content)
	}
	if firstGroup > secondGroup || selected > second {
		t.Fatalf("expected operations to preserve visible order, got %q", content)
	}
	if strings.Contains(content, "List pets") || strings.Contains(content, "Create pet") {
		t.Fatalf("expected operations rows to omit summaries, got %q", content)
	}
}

func TestRenderShowsEmptyState(t *testing.T) {
	t.Parallel()

	content := ansi.Strip(Render(Data{
		HasSpec:         true,
		TotalOperations: 0,
	}))

	if !strings.Contains(content, "does not define any operations") {
		t.Fatalf("expected empty operations state, got %q", content)
	}
}

func TestRenderShowsFilteredEmptyState(t *testing.T) {
	t.Parallel()

	content := ansi.Strip(Render(Data{
		HasSpec:         true,
		TotalOperations: 2,
	}))

	if !strings.Contains(content, "No operations match the current filter.") {
		t.Fatalf("expected filtered empty state, got %q", content)
	}
	if !strings.Contains(content, "Press Esc to clear the filter.") {
		t.Fatalf("expected filtered empty state to mention Esc, got %q", content)
	}
}
