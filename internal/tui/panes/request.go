package panes

import (
	"strings"

	"github.com/phergul/apiscope/internal/tui/widgets"
)

type RequestRow struct {
	Label    string
	Meta     string
	Value    string
	Editable bool
}

type RequestEditView struct {
	Kind      string
	Buffer    string
	MediaType string
	View      string
}

type RequestData struct {
	LoadInFlight  bool
	Sections      []string
	ActiveSection string
	Rows          []RequestRow
	ActiveRow     int
	Edit          RequestEditView
	EmptyState    string
}

func RenderRequest(data RequestData) string {
	if data.LoadInFlight {
		return "Loading spec..."
	}
	if len(data.Sections) == 0 {
		return data.EmptyState
	}

	return strings.Join([]string{
		RenderSectionLabels(data.Sections, data.ActiveSection),
		"",
		RenderActiveRequestSection(data),
	}, "\n")
}

func RenderActiveRequestSection(data RequestData) string {
	if data.Edit.Kind == "body" {
		return renderRequestBodyEditor(data.Edit)
	}

	if len(data.Rows) == 0 {
		return "No inputs available."
	}

	activeIndex := requestActiveRowIndex(data.Rows, data.ActiveRow)
	items := make([]widgets.ListItem, 0, len(data.Rows))
	for index, row := range data.Rows {
		label := row.Label
		if row.Meta != "" {
			label += " (" + row.Meta + ")"
		}

		value := row.Value
		if index == activeIndex && data.Edit.Kind == "field" {
			editorView := data.Edit.View
			if strings.TrimSpace(editorView) == "" {
				editorView = data.Edit.Buffer
			}
			value = editorView + " [Enter save, Esc cancel]"
		}
		if !row.Editable && value != "" {
			value += " [read-only]"
		}

		line := label
		if value != "" {
			line += " = " + value
		}
		items = append(items, widgets.ListItem{
			Content:  line,
			Selected: index == activeIndex,
			Muted:    !row.Editable,
		})
	}

	return widgets.RenderList(items)
}

func renderRequestBodyEditor(edit RequestEditView) string {
	lines := []string{
		"Media type: " + fallbackText(edit.MediaType, "none"),
		"Ctrl+S save | Esc cancel",
		"",
	}

	body := edit.View
	if strings.TrimSpace(body) == "" {
		body = edit.Buffer
	}
	if body == "" {
		body = "<empty>"
	}

	lines = append(lines, strings.Split(body, "\n")...)
	return strings.Join(lines, "\n")
}

func requestActiveRowIndex(rows []RequestRow, activeRow int) int {
	if len(rows) == 0 {
		return 0
	}
	if activeRow < 0 {
		return 0
	}
	if activeRow >= len(rows) {
		return len(rows) - 1
	}

	return activeRow
}
