package history

import (
	"strings"
	"testing"

	"github.com/phergul/apiscope/internal/model"

	"github.com/charmbracelet/x/ansi"
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
		ContentWidth:  88,
		ContentHeight: 24,
	})

	for _, snippet := range []string{
		"#8  transport failed",
		"#7  200 OK",
		"Request",
		"Request ID: 8",
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
	for _, snippet := range []string{"Response", "Transport: dial tcp timeout"} {
		if !strings.Contains(data.Body, snippet) {
			t.Fatalf("expected popup response preview to include %q, got %q", snippet, data.Body)
		}
	}
	if strings.Contains(data.Body, "Selected request") {
		t.Fatalf("expected old stacked selected-request block to be removed, got %q", data.Body)
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
	for _, snippet := range []string{"Enter load response", "r restore request", "Ctrl+U / Ctrl+D scroll preview", "t / T switch theme"} {
		if !strings.Contains(help.Body, snippet) {
			t.Fatalf("expected history help to include %q, got %q", snippet, help.Body)
		}
	}
}

func TestProjectPopupClipsAndScrollsPreview(t *testing.T) {
	t.Parallel()

	data := ProjectPopup(PopupInput{
		Selected: &model.Operation{
			Method: "GET",
			Path:   "/pets",
		},
		Entries: []model.HistoryEntry{{
			RequestID:    5,
			OperationKey: model.NewOperationKey("GET", "/pets"),
			ServerURL:    "https://api.example.com",
			Request: model.ExecutedRequestSnapshot{
				ServerURL: "https://api.example.com",
				Draft: model.RequestDraft{
					BodyRaw: "{\n  \"name\": \"fido\"\n}",
				},
			},
			Response: &model.HTTPResponse{
				Status: "200 OK",
				Headers: map[string][]string{
					"X-01": {"one"},
					"X-02": {"two"},
					"X-03": {"three"},
					"X-04": {"four"},
					"X-05": {"five"},
					"X-06": {"six"},
					"X-07": {"seven"},
					"X-08": {"eight"},
				},
				PrettyBody:  "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10",
				ContentType: "application/json",
			},
		}},
		ActiveRow:           0,
		PreviewScrollOffset: 12,
		ContentWidth:        88,
		ContentHeight:       8,
	})

	if data.MaxPreviewScroll == 0 {
		t.Fatal("expected long preview content to become scrollable")
	}
	if !strings.Contains(data.Body, "X-02: two") {
		t.Fatalf("expected preview scroll offset to reveal lower body lines, got %q", data.Body)
	}
	if strings.Contains(data.Body, "Request ID: 5") {
		t.Fatalf("expected top-of-preview request details to scroll out, got %q", data.Body)
	}
}

func TestProjectPopupNormalizesCarriageReturnsInBodyPreview(t *testing.T) {
	t.Parallel()

	data := ProjectPopup(PopupInput{
		Selected: &model.Operation{
			Method: "GET",
			Path:   "/pets",
		},
		Entries: []model.HistoryEntry{{
			RequestID: 5,
			Request: model.ExecutedRequestSnapshot{
				ServerURL: "https://api.example.com",
			},
			Response: &model.HTTPResponse{
				Status: "404 Not Found",
				Body:   []byte("<html>\r\n<body>\r\n<center>oops</center>\r\n</body>\r\n</html>"),
			},
		}},
		ActiveRow:     0,
		ContentWidth:  88,
		ContentHeight: 20,
	})

	content := ansi.Strip(data.Body)
	if strings.Contains(content, "\r") {
		t.Fatalf("expected carriage returns to be normalized, got %q", content)
	}
	for _, snippet := range []string{"<html>", "<body>", "<center>oops</center>", "</html>"} {
		if !strings.Contains(content, snippet) {
			t.Fatalf("expected normalized preview to include %q, got %q", snippet, content)
		}
	}
}
