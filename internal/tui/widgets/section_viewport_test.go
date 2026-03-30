package widgets

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestProjectClippedSectionViewClipsOnlyActiveSection(t *testing.T) {
	t.Parallel()

	projected := ProjectClippedSectionView(ClippedSectionViewInput{
		Sections: []Section{
			{
				Label: "Summary",
				Body:  strings.Join([]string{"line1", "line2", "line3", "line4"}, "\n"),
			},
			{
				Label: "Security",
				Body:  "security body",
			},
		},
		Active:        "Summary",
		ContentWidth:  24,
		ContentHeight: 2,
		ScrollOffset:  1,
	})

	if projected.MaxScrollOffset != 2 {
		t.Fatalf("expected max scroll offset 2, got %d", projected.MaxScrollOffset)
	}
	if projected.Data.Sections[1].Body != "security body" {
		t.Fatalf("expected inactive section body to remain unchanged, got %q", projected.Data.Sections[1].Body)
	}

	content := ansi.Strip(RenderSectionView(projected.Data))
	if strings.Contains(content, "line1") || strings.Contains(content, "line4") {
		t.Fatalf("expected active section body to be clipped, got %q", content)
	}
	if !strings.Contains(content, "line2") || !strings.Contains(content, "line3") {
		t.Fatalf("expected clipped active section body to keep visible lines, got %q", content)
	}
}

func TestProjectClippedSectionViewFallsBackToFirstSection(t *testing.T) {
	t.Parallel()

	projected := ProjectClippedSectionView(ClippedSectionViewInput{
		Sections: []Section{
			{Label: "Summary", Body: "summary"},
			{Label: "Warnings", Body: "warnings"},
		},
		Active:        "Missing",
		ContentWidth:  20,
		ContentHeight: 1,
	})

	if projected.Active != "Summary" {
		t.Fatalf("expected missing active section to fall back to Summary, got %q", projected.Active)
	}

	content := ansi.Strip(RenderSectionView(projected.Data))
	if !strings.Contains(content, "Summary  Warnings") {
		t.Fatalf("expected section strip to be preserved, got %q", content)
	}
	if !strings.Contains(content, "summary") {
		t.Fatalf("expected fallback section body to render, got %q", content)
	}
}
