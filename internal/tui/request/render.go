package request

import (
	"strings"

	"github.com/phergul/apiscope/internal/tui/widgets"
)

type Row struct {
	Label    string
	Meta     string
	Value    string
	Editable bool
}

type EditView struct {
	Kind      string
	Buffer    string
	MediaType string
	View      string
}

type Data struct {
	LoadInFlight  bool
	Sections      []string
	ActiveSection string
	Rows          []Row
	ActiveRow     int
	Edit          EditView
	EmptyState    string
}

func Render(data Data) string {
	if data.LoadInFlight {
		return "Loading spec..."
	}
	if len(data.Sections) == 0 {
		return data.EmptyState
	}

	return strings.Join([]string{
		widgets.RenderSectionLabels(widgets.SectionStripData{
			Labels: data.Sections,
			Active: data.ActiveSection,
		}),
		"",
		RenderActiveSection(data),
	}, "\n")
}

func RenderActiveSection(data Data) string {
	if data.Edit.Kind == "body" {
		return renderBodyEditor(data.Edit)
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

func renderBodyEditor(edit EditView) string {
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

func requestActiveRowIndex(rows []Row, activeRow int) int {
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

func fallbackText(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}

	return value
}
