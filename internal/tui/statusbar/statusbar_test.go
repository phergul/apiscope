package statusbar

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestRenderShowsOnlyStatusAndHelpHint(t *testing.T) {
	t.Parallel()

	content := ansi.Strip(Render(Data{
		Status:   "Request succeeded",
		HelpHint: "Help - ?",
	}, 160))

	if !strings.Contains(content, "Request succeeded") {
		t.Fatalf("expected status content to remain, got %q", content)
	}
	for _, snippet := range []string{"Source:", "State:", "Focus:", "Operation:", "Server:", "Count:", "Visible:", "Warnings:", "Keys:"} {
		if strings.Contains(content, snippet) {
			t.Fatalf("expected debug snippet %q to be removed, got %q", snippet, content)
		}
	}
}

func TestRenderAlignsHelpHintToRightSide(t *testing.T) {
	t.Parallel()

	content := ansi.Strip(Render(Data{
		Status:   "Loading spec",
		HelpHint: "Help - ?",
	}, 80))

	if !strings.HasSuffix(content, "Help - ?") {
		t.Fatalf("expected help hint to align to the far right, got %q", content)
	}
	if !strings.Contains(content, "Loading spec") {
		t.Fatalf("expected status content to remain, got %q", content)
	}
}

func TestRenderStaysSingleLineWhenStatusIsLong(t *testing.T) {
	t.Parallel()

	content := ansi.Strip(Render(Data{
		Status:   "Restored request #123 with a very long message that should not wrap onto a second line",
		HelpHint: "Help - ?",
	}, 36))

	if strings.Contains(content, "\n") {
		t.Fatalf("expected single-line status bar output, got %q", content)
	}
	if !strings.Contains(content, "Help - ?") {
		t.Fatalf("expected help hint to remain visible, got %q", content)
	}
}
