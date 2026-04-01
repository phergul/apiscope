package widgets

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestRenderCenteredModalCentersContentWithinViewport(t *testing.T) {
	t.Parallel()

	content := ansi.Strip(RenderCenteredModal(40, 10, CenteredModalData{
		Body:  "Failed to load spec\n\n[ Quit ]",
		Width: 24,
	}))

	lines := strings.Split(content, "\n")
	if len(lines) != 10 {
		t.Fatalf("expected 10 viewport lines, got %d", len(lines))
	}
	if strings.TrimSpace(lines[0]) != "" {
		t.Fatalf("expected vertical padding above modal, got %q", lines[0])
	}
	if strings.TrimSpace(lines[len(lines)-1]) != "" {
		t.Fatalf("expected vertical padding below modal, got %q", lines[len(lines)-1])
	}
	if !strings.Contains(content, "Failed to load") || !strings.Contains(content, "spec") || !strings.Contains(content, "[ Quit ]") {
		t.Fatalf("expected centered modal content, got %q", content)
	}
}

func TestOverlayBottomRightAnchorsPopupAboveBottomInset(t *testing.T) {
	t.Parallel()

	base := strings.Join([]string{
		"row 1",
		"row 2",
		"row 3",
		"row 4",
		"status line",
	}, "\n")

	content := ansi.Strip(OverlayBottomRight(BottomRightOverlayData{
		Base:        base,
		Popup:       "POP",
		BottomInset: 1,
	}))

	lines := strings.Split(content, "\n")
	if lines[len(lines)-1] != "status line" {
		t.Fatalf("expected bottom inset to preserve the status line, got %q", lines[len(lines)-1])
	}
	if !strings.Contains(lines[len(lines)-2], "POP") {
		t.Fatalf("expected popup to anchor above the bottom inset, got %q", lines[len(lines)-2])
	}
	if !strings.Contains(content, "row 1") {
		t.Fatalf("expected overlay to preserve earlier base content, got %q", content)
	}
}

func TestOverlayCenteredPlacesPopupInViewportMiddle(t *testing.T) {
	t.Parallel()

	base := strings.Join([]string{
		"row 1",
		"row 2",
		"row 3",
		"row 4",
		"row 5",
	}, "\n")

	content := ansi.Strip(OverlayCentered(CenteredOverlayData{
		Base:  base,
		Popup: "POP",
	}))

	lines := strings.Split(content, "\n")
	if !strings.Contains(lines[2], "POP") {
		t.Fatalf("expected popup near the vertical middle, got %q", lines[2])
	}
	if !strings.Contains(content, "row 1") || !strings.Contains(content, "row 5") {
		t.Fatalf("expected centered overlay to preserve surrounding base content, got %q", content)
	}
}
