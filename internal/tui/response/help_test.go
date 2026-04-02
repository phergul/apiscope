package response

import (
	"strings"
	"testing"
)

func TestBuildBrowseHelpViewIncludesResponseControls(t *testing.T) {
	t.Parallel()

	help := BuildBrowseHelpView()
	if help.Title != "Response help" {
		t.Fatalf("expected response help title, got %q", help.Title)
	}
	for _, snippet := range []string{"switch responses", "t / T switch theme", "z zoom focused pane"} {
		if !strings.Contains(help.Body, snippet) {
			t.Fatalf("expected response help to include %q, got %q", snippet, help.Body)
		}
	}
}
