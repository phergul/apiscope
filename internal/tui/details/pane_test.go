package details

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/phergul/apiscope/internal/model"
)

func TestProjectPaneFallsBackToFirstAvailableSection(t *testing.T) {
	t.Parallel()

	projected := ProjectPane(PaneInput{
		Selected: &model.Operation{
			Method: "GET",
			Path:   "/pets",
		},
		ActiveSection: "Missing",
	})

	if projected.Data.ActiveSection != SectionSummary {
		t.Fatalf("expected missing details section to fall back to summary, got %q", projected.Data.ActiveSection)
	}
}

func TestProjectPaneClipsSummaryBodyAndTracksScroll(t *testing.T) {
	t.Parallel()

	projected := ProjectPane(PaneInput{
		Selected: &model.Operation{
			Method:      "GET",
			Path:        "/pets",
			Summary:     "List pets",
			Description: "line1\nline2\nline3\nline4\nline5\nline6",
		},
		ActiveSection: SectionSummary,
		ContentWidth:  40,
		ContentHeight: 3,
		ScrollOffset:  2,
	})

	if projected.MaxScrollOffset == 0 {
		t.Fatal("expected summary body to be scrollable")
	}

	content := ansi.Strip(Render(projected.Data))
	if strings.Contains(content, "line1") || strings.Contains(content, "line6") {
		t.Fatalf("expected summary body to be clipped, got %q", content)
	}
	if !strings.Contains(content, "line2") || !strings.Contains(content, "line4") {
		t.Fatalf("expected clipped summary body to respect scroll offset, got %q", content)
	}
}

func TestProjectPaneRendersWarningsSection(t *testing.T) {
	t.Parallel()

	projected := ProjectPane(PaneInput{
		Selected: &model.Operation{
			Method: "GET",
			Path:   "/pets",
		},
		ActiveSection: SectionWarnings,
		Warnings: []model.SpecWarning{
			{
				Code:    model.SpecWarningUnsupportedFeature,
				Message: "callbacks are not supported",
				Path:    "#/paths/~1pets/get/callbacks",
			},
		},
		ContentWidth:  80,
		ContentHeight: 6,
	})

	content := ansi.Strip(Render(projected.Data))
	if !strings.Contains(content, "- unsupported_feature: callbacks are not supported") {
		t.Fatalf("expected warnings section to render, got %q", content)
	}
}
