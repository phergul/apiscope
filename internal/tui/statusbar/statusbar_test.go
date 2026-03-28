package statusbar

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestRenderIncludesOperationIdentityAndCount(t *testing.T) {
	t.Parallel()

	content := ansi.Strip(Render(Data{
		Source:         "demo.yaml",
		State:          "loaded",
		Focus:          "operations",
		OperationLabel: "GET /pets",
		HasSpec:        true,
		OperationCount: 2,
		VisibleCount:   2,
		WarningCount:   2,
	}))

	wantSnippets := []string{
		"Source: demo.yaml",
		"State: loaded",
		"Focus: operations",
		"Operation: GET /pets",
		"Count: 2",
		"Visible: 2",
		"Warnings: 2",
		"Keys: 1-4 switch Tab cycle z zoom q quit",
	}
	for _, snippet := range wantSnippets {
		if !strings.Contains(content, snippet) {
			t.Fatalf("expected status bar to include %q, got %q", snippet, content)
		}
	}
}
