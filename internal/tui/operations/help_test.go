package operations

import (
	"strings"
	"testing"
)

func TestBuildBrowseHelpViewIncludesBrowseControls(t *testing.T) {
	t.Parallel()

	help := BuildBrowseHelpView()
	if help.Title != "Operations help" {
		t.Fatalf("expected operations help title, got %q", help.Title)
	}
	for _, snippet := range []string{"/ filter operations", "t / T switch theme", "z zoom focused pane"} {
		if !strings.Contains(help.Body, snippet) {
			t.Fatalf("expected operations browse help to include %q, got %q", snippet, help.Body)
		}
	}
}

func TestBuildFilterHelpViewIncludesFilterControls(t *testing.T) {
	t.Parallel()

	help := BuildFilterHelpView()
	if help.Title != "Filter help" {
		t.Fatalf("expected filter help title, got %q", help.Title)
	}
	for _, snippet := range []string{"Type to filter operations", "Esc clear and close filter"} {
		if !strings.Contains(help.Body, snippet) {
			t.Fatalf("expected filter help to include %q, got %q", snippet, help.Body)
		}
	}
}
