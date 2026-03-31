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
		SelectedServer: "https://api.example.com",
		HasSpec:        true,
		OperationCount: 2,
		VisibleCount:   2,
		WarningCount:   2,
	}, 160))

	wantSnippets := []string{
		"Source: demo.yaml",
		"State: loaded",
		"Focus: operations",
		"Operation: GET /pets",
		"Server: https://api.example.com",
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

func TestRenderAlignsHelpHintToRightSide(t *testing.T) {
	t.Parallel()

	content := ansi.Strip(Render(Data{
		Source:   "demo.yaml",
		State:    "loaded",
		Focus:    "request",
		HelpHint: "Help - ?",
	}, 80))

	if !strings.HasSuffix(content, "Help - ?") {
		t.Fatalf("expected help hint to align to the far right, got %q", content)
	}
	if !strings.Contains(content, "Source: demo.yaml") {
		t.Fatalf("expected normal status content to remain, got %q", content)
	}
}
