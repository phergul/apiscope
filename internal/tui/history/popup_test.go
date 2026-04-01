package history

import (
	"strings"
	"testing"

	"github.com/phergul/apiscope/internal/model"
)

func TestProjectPopupShowsOperationScopedEmptyState(t *testing.T) {
	t.Parallel()

	data := ProjectPopup(PopupInput{
		Selected: &model.Operation{
			Method: "GET",
			Path:   "/pets",
		},
	})

	if !strings.Contains(data.Body, "No previous requests for this operation yet.") {
		t.Fatalf("expected empty-state copy, got %q", data.Body)
	}
	if data.Meta != "GET /pets" {
		t.Fatalf("expected operation meta, got %q", data.Meta)
	}
}

func TestProjectPopupRendersHistoryRowsAndSelectedRequestDetails(t *testing.T) {
	t.Parallel()

	data := ProjectPopup(PopupInput{
		Selected: &model.Operation{
			Method: "POST",
			Path:   "/pets/{petId}",
		},
		Entries: []model.HistoryEntry{
			{
				RequestID:     8,
				ServerURL:     "https://staging.example.com",
				TransportNote: "dial tcp timeout",
				Request: model.ExecutedRequestSnapshot{
					ServerURL: "https://staging.example.com",
					Draft: model.RequestDraft{
						PathParams:     map[string]string{"petId": "abc"},
						FormFileParams: map[string]string{"file": "/tmp/demo.txt"},
						BodyMediaType:  "application/json",
						BodyRaw:        "{\n  \"name\": \"fido\"\n}",
					},
					AuthState: map[string]model.AuthValue{
						"api_key": {Type: model.AuthSchemeValueTypeAPIKey, APIKey: "secret"},
					},
				},
			},
			{
				RequestID: 7,
				ServerURL: "https://api.example.com",
				Response:  &model.HTTPResponse{Status: "200 OK"},
				Request: model.ExecutedRequestSnapshot{
					ServerURL: "https://api.example.com",
				},
			},
		},
		ActiveRow:     0,
		ContentWidth:  72,
		ContentHeight: 14,
	})

	for _, snippet := range []string{
		"Requests",
		"#8  transport failed  https://staging.example.com",
		"#7  200 OK  https://api.example.com",
		"Selected request",
		"Server: https://staging.example.com",
		"Path: petId=abc",
		"Files: file=/tmp/demo.txt",
		"Body media type: application/json",
		"Body preview:",
		"Auth: api_key key set",
	} {
		if !strings.Contains(data.Body, snippet) {
			t.Fatalf("expected popup body to include %q, got %q", snippet, data.Body)
		}
	}
}

func TestMoveAndBoundaryActiveRowClampSelection(t *testing.T) {
	t.Parallel()

	entries := []model.HistoryEntry{{RequestID: 1}, {RequestID: 2}, {RequestID: 3}}

	if got := MoveActiveRow(entries, 0, -1); got != 0 {
		t.Fatalf("expected clamp at first row, got %d", got)
	}
	if got := MoveActiveRow(entries, 1, 2); got != 2 {
		t.Fatalf("expected clamp at last row, got %d", got)
	}
	if got := BoundaryActiveRow(entries, false); got != 0 {
		t.Fatalf("expected home boundary, got %d", got)
	}
	if got := BoundaryActiveRow(entries, true); got != 2 {
		t.Fatalf("expected end boundary, got %d", got)
	}
}

func TestBuildHelpViewIncludesHistoryControls(t *testing.T) {
	t.Parallel()

	help := BuildHelpView()
	if help.Title != "History help" {
		t.Fatalf("expected history help title, got %q", help.Title)
	}
	for _, snippet := range []string{"Enter load response", "r restore request"} {
		if !strings.Contains(help.Body, snippet) {
			t.Fatalf("expected history help to include %q, got %q", snippet, help.Body)
		}
	}
}
