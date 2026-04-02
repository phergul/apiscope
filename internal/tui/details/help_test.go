package details

import (
	"strings"
	"testing"
)

func TestBuildBrowseHelpViewIncludesSectionControls(t *testing.T) {
	t.Parallel()

	help := BuildBrowseHelpView()
	if help.Title != "Details help" {
		t.Fatalf("expected details help title, got %q", help.Title)
	}
	for _, snippet := range []string{"switch sections", "t / T switch theme", "z zoom focused pane"} {
		if !strings.Contains(help.Body, snippet) {
			t.Fatalf("expected details help to include %q, got %q", snippet, help.Body)
		}
	}
}
