package widgets

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestRenderPopupShowsMetaAndHelpWhenEnabled(t *testing.T) {
	t.Parallel()

	content := ansi.Strip(RenderPopup(PopupData{
		Title:       "Edit value",
		Meta:        "hint",
		Body:        "limit (optional, integer)\n\n42",
		Help:        "Enter save\nEsc cancel",
		HelpVisible: true,
		Width:       36,
		Focused:     true,
	}))

	for _, snippet := range []string{"Edit value", "hint", "limit (optional, integer)", "Enter save", "Esc cancel"} {
		if !strings.Contains(content, snippet) {
			t.Fatalf("expected popup to include %q, got %q", snippet, content)
		}
	}

	for _, line := range strings.Split(strings.TrimRight(content, "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if !strings.HasPrefix(line, "│") && !strings.HasPrefix(line, "┌") && !strings.HasPrefix(line, "└") {
			t.Fatalf("expected framed popup line, got %q", line)
		}
		if !strings.HasSuffix(line, "│") && !strings.HasSuffix(line, "┐") && !strings.HasSuffix(line, "┘") {
			t.Fatalf("expected popup line to include right border, got %q", line)
		}
	}
}

func TestOverlayPlacesPopupOverBaseContent(t *testing.T) {
	t.Parallel()

	content := Overlay("base line 1\nbase line 2", RenderPopup(PopupData{
		Title:   "Edit",
		Body:    "x",
		Width:   24,
		Focused: true,
	}), 2, 1)

	if !strings.Contains(content, "Edit") {
		t.Fatalf("expected overlay to include popup content, got %q", content)
	}
	if !strings.Contains(content, "base line 1") {
		t.Fatalf("expected overlay to preserve untouched base content, got %q", content)
	}
}

func TestOverlayPreservesUnderlyingContentOutsidePopupWidth(t *testing.T) {
	t.Parallel()

	base := "0123456789abcdefghij"
	content := ansi.Strip(Overlay(base, "POP", 5, 0))
	if !strings.Contains(content, "01234POP89abcdefghij") {
		t.Fatalf("expected overlay to preserve content outside popup width, got %q", content)
	}
}
