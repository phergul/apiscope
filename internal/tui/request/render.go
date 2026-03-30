package request

import (
	"strings"

	"github.com/phergul/apiscope/internal/tui/widgets"

	"github.com/charmbracelet/lipgloss"
)

type Row struct {
	Label    string
	Meta     string
	Value    string
	Editable bool
	Error    string
}

type EditView struct {
	Kind      string
	Buffer    string
	MediaType string
	View      string
	Title     string
	Context   string
	Meta      string
	Help      string
	ShowHelp  bool
}

type Data struct {
	LoadInFlight     bool
	Sections         []string
	ActiveSection    string
	Rows             []Row
	ActiveRow        int
	Edit             EditView
	EmptyState       string
	ValidationNotice []string
	ContentWidth     int
	ContentHeight    int
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
	parts := make([]string, 0, 3)

	if len(data.Rows) == 0 {
		if summary := renderValidationSummary(data.ValidationNotice); summary != "" {
			parts = append(parts, summary)
		}
		base := strings.Join(append(parts, "No inputs available."), "\n\n")
		if data.Edit.Kind == "" {
			return base
		}
		return renderEditorOverlay(base, data)
	}

	activeIndex := requestActiveRowIndex(data.Rows, data.ActiveRow)
	lines := make([]string, 0, len(data.Rows)*2)
	for index, row := range data.Rows {
		label := row.Label
		if row.Meta != "" {
			label += " (" + row.Meta + ")"
		}

		value := row.Value
		if !row.Editable && value != "" {
			value += " [read-only]"
		}

		line := label
		if value != "" {
			line += " = " + value
		}
		lines = append(lines, widgets.RenderList([]widgets.ListItem{{
			Content:  line,
			Selected: index == activeIndex,
			Muted:    !row.Editable,
		}}))
		if strings.TrimSpace(row.Error) != "" {
			lines = append(lines, "  "+widgets.RenderValidationMessage(row.Error))
		}
	}

	if summary := renderValidationSummary(data.ValidationNotice); summary != "" {
		parts = append(parts, summary)
	}
	base := strings.Join(append(parts, strings.Join(lines, "\n")), "\n\n")
	if data.Edit.Kind == "" {
		return base
	}

	return renderEditorOverlay(base, data)
}

func renderEditorOverlay(base string, data Data) string {
	editorPopup := widgets.RenderPopup(widgets.PopupData{
		Title:   data.Edit.Title,
		Body:    editorPopupBody(data),
		Width:   popupWidth(data),
		Focused: true,
	})

	return widgets.Overlay(base, editorPopup, popupX(data, editorPopup), popupY(data, editorPopup))
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

func renderValidationSummary(messages []string) string {
	if len(messages) == 0 {
		return ""
	}

	lines := []string{widgets.RenderValidationMessage("Validation:")}
	for _, message := range messages {
		lines = append(lines, widgets.RenderValidationMessage("- "+message))
	}

	return strings.Join(lines, "\n")
}

func editorPopupBody(data Data) string {
	lines := make([]string, 0, 5)
	if strings.TrimSpace(data.Edit.Context) != "" {
		lines = append(lines, data.Edit.Context, "")
	}

	editor := strings.TrimSpace(data.Edit.View)
	if editor == "" {
		editor = data.Edit.Buffer
	}
	if strings.TrimSpace(editor) == "" {
		editor = "<empty>"
	}

	lines = append(lines, editor)
	return strings.Join(lines, "\n")
}

func popupWidth(data Data) int {
	if data.Edit.Kind == "body" {
		return min(max(data.ContentWidth-8, 28), 84)
	}

	return min(max(data.ContentWidth-10, 24), 64)
}

func popupX(data Data, popup string) int {
	popupWidth := lipgloss.Width(popup)
	return max((max(data.ContentWidth, popupWidth)-popupWidth)/2, 0)
}

func popupY(data Data, popup string) int {
	popupHeight := lipgloss.Height(popup)
	if data.Edit.Kind == "body" {
		return max((max(data.ContentHeight, popupHeight)-popupHeight)/2, 0)
	}

	summaryHeight := 0
	if len(data.ValidationNotice) > 0 {
		summaryHeight = lipgloss.Height(renderValidationSummary(data.ValidationNotice)) + 2
	}
	target := summaryHeight + data.ActiveRow + 1
	maxY := max(data.ContentHeight-popupHeight, 0)
	if target > maxY {
		target = maxY
	}
	return max(target, 0)
}
