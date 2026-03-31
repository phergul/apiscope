package widgets

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
)

func TestRenderPaneFrameKeepsCenterTitleCenteredWhenRightTitleExists(t *testing.T) {
	t.Parallel()

	const width = 60
	const centerTitle = "3 Request"
	const rightTitle = "Send request Ctrl+R"

	withoutRight := ansi.Strip(RenderPaneFrame("", centerTitle, "", "", width, false))
	withRight := ansi.Strip(RenderPaneFrame("", centerTitle, rightTitle, "", width, false))

	withoutRightTop := strings.Split(withoutRight, "\n")[0]
	withRightTop := strings.Split(withRight, "\n")[0]
	centerText := " " + centerTitle + " "
	wantStart := 1 + (width-2-len(centerText))/2

	if got := displayIndex(withoutRightTop, centerText); got != wantStart {
		t.Fatalf("expected centered title without right title at column %d, got %d in %q", wantStart, got, withoutRightTop)
	}
	if got := displayIndex(withRightTop, centerText); got != wantStart {
		t.Fatalf("expected centered title with right title at column %d, got %d in %q", wantStart, got, withRightTop)
	}
}

func TestRenderPaneFrameKeepsRightTitleAnchoredToRightBorder(t *testing.T) {
	t.Parallel()

	const width = 60
	const centerTitle = "3 Request"
	const rightTitle = "Send request Ctrl+R"

	rendered := ansi.Strip(RenderPaneFrame("", centerTitle, rightTitle, "", width, false))
	top := strings.Split(rendered, "\n")[0]
	rightText := " " + rightTitle + " "
	wantStart := width - 1 - len(rightText)

	if got := displayIndex(top, rightText); got != wantStart {
		t.Fatalf("expected right title at column %d, got %d in %q", wantStart, got, top)
	}
}

func displayIndex(line, segment string) int {
	byteIndex := strings.Index(line, segment)
	if byteIndex < 0 {
		return -1
	}

	return utf8.RuneCountInString(line[:byteIndex])
}
