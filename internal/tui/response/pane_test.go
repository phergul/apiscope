package response

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/phergul/apiscope/internal/model"
)

func TestProjectPaneFallsBackToLiveSection(t *testing.T) {
	t.Parallel()

	projected := ProjectPane(PaneInput{
		Selected: &model.Operation{
			Key:       model.NewOperationKey("GET", "/pets"),
			Method:    "GET",
			Path:      "/pets",
			Responses: []model.ResponseSpec{{StatusCode: "200", Description: "OK"}},
		},
		ActiveSection: "Missing",
	})

	if projected.Data.ActiveSection != SectionLive {
		t.Fatalf("expected missing response section to fall back to Live, got %q", projected.Data.ActiveSection)
	}
}

func TestProjectPaneClipsLiveBodyAndTracksScroll(t *testing.T) {
	t.Parallel()

	projected := ProjectPane(PaneInput{
		Selected: &model.Operation{
			Key:    model.NewOperationKey("GET", "/pets"),
			Method: "GET",
			Path:   "/pets",
		},
		LastResponse: &model.HTTPResponse{
			OperationKey: model.NewOperationKey("GET", "/pets"),
			Status:       "200 OK",
			PrettyBody:   "line1\nline2\nline3\nline4\nline5",
		},
		ActiveSection: SectionLive,
		ContentWidth:  50,
		ContentHeight: 3,
		ScrollOffset:  7,
	})

	if projected.MaxScrollOffset == 0 {
		t.Fatal("expected live response body to be scrollable")
	}

	content := ansi.Strip(Render(projected.Data))
	if strings.Contains(content, "Status:") || strings.Contains(content, "Duration:") || strings.Contains(content, "Headers:") {
		t.Fatalf("expected live response body to be clipped, got %q", content)
	}
	if !strings.Contains(content, "line1") || !strings.Contains(content, "line3") {
		t.Fatalf("expected clipped live response body to respect scroll offset, got %q", content)
	}
}

func TestProjectPaneRendersDeclaredResponses(t *testing.T) {
	t.Parallel()

	projected := ProjectPane(PaneInput{
		Selected: &model.Operation{
			Key:    model.NewOperationKey("GET", "/pets"),
			Method: "GET",
			Path:   "/pets",
			Responses: []model.ResponseSpec{
				{StatusCode: "200", Description: "OK", Content: []model.MediaTypeSpec{{MediaType: "application/json"}}},
				{StatusCode: "default", Description: "Unexpected error", Content: []model.MediaTypeSpec{{MediaType: "application/problem+json"}}},
			},
		},
		ActiveSection: "200",
		ContentWidth:  50,
		ContentHeight: 4,
	})

	content := ansi.Strip(Render(projected.Data))
	if !strings.Contains(content, "Live  200  default") {
		t.Fatalf("expected response section strip to render, got %q", content)
	}
	if !strings.Contains(content, "Description: OK") {
		t.Fatalf("expected declared response body to render, got %q", content)
	}
}
