package request

import (
	"testing"

	"github.com/phergul/apiscope/internal/model"
)

func TestProjectPaneResolvesActiveSectionBeforeProjectingRows(t *testing.T) {
	t.Parallel()

	projection := ProjectPane(PaneInput{
		Selected: &model.Operation{
			Parameters: []model.Parameter{
				{
					Name: "limit",
					In:   model.ParameterLocationQuery,
				},
			},
		},
		Draft: &model.RequestDraft{
			QueryParams: map[string]string{"limit": "10"},
		},
		ActiveSection: "Missing",
	})

	if projection.Data.ActiveSection != "Query" {
		t.Fatalf("expected active section to fall back to Query, got %q", projection.Data.ActiveSection)
	}
	if len(projection.Data.Rows) != 1 {
		t.Fatalf("expected one projected row, got %d", len(projection.Data.Rows))
	}
	if projection.Data.Rows[0].Label != "limit" {
		t.Fatalf("expected projected row label limit, got %q", projection.Data.Rows[0].Label)
	}
}

func TestProjectPaneKeepsOnlyHintWhenHelpIsClosed(t *testing.T) {
	t.Parallel()

	projection := ProjectPane(PaneInput{
		Selected: &model.Operation{
			RequestBody: &model.RequestBodySpec{
				Content: []model.MediaTypeSpec{{MediaType: "application/json"}},
			},
		},
		Draft:         &model.RequestDraft{},
		ActiveSection: SectionBody,
		Editor: EditorInput{
			Kind:   model.RequestEditKindBody,
			Buffer: "{}",
		},
		HelpOpen: false,
	})

	if projection.HelpOverlay.Hint != "Help - ?" {
		t.Fatalf("expected closed help overlay hint, got %q", projection.HelpOverlay.Hint)
	}
	if projection.HelpOverlay.Title != "" || projection.HelpOverlay.Body != "" {
		t.Fatalf("expected closed help overlay to omit body, got %+v", projection.HelpOverlay)
	}
}
