package request

import (
	"strings"

	"github.com/phergul/apiscope/internal/tui/widgets"
	"github.com/phergul/apiscope/internal/util"

	"github.com/charmbracelet/lipgloss"
)

type Row struct {
	Kind     RowKind
	Label    string
	Meta     string
	Value    string
	Editable bool
	Error    string
	Support  []SupportNote
}

type EditView struct {
	Kind      string
	Buffer    string
	MediaType string
	View      string
	Title     string
	Context   string
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
	SupportNotice    []SupportNote
	ContentWidth     int
	ContentHeight    int
}

// Render renders the full request pane body from the projected request data.
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

// RenderActiveSection renders the active request section content, including any active editor overlay.
func RenderActiveSection(data Data) string {
	parts := make([]string, 0, 3)

	if len(data.Rows) == 0 {
		if summary := renderValidationSummary(data.ValidationNotice); summary != "" {
			parts = append(parts, summary)
		}
		if summary := renderSupportSummary(data.SupportNotice); summary != "" {
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
		if row.Kind == RowKindBodyText {
			lines = append(lines, renderBodyPreviewRow(row, index == activeIndex, data.ContentWidth))
		} else {
			lines = append(lines, renderRequestRow(row, index == activeIndex))
		}
		if strings.TrimSpace(row.Error) != "" {
			lines = append(lines, "  "+widgets.RenderValidationMessage(row.Error))
		}
		for _, note := range row.Support {
			lines = append(lines, "  "+renderSupportMessage(note))
		}
	}

	if summary := renderValidationSummary(data.ValidationNotice); summary != "" {
		parts = append(parts, summary)
	}
	if summary := renderSupportSummary(data.SupportNotice); summary != "" {
		parts = append(parts, summary)
	}
	base := strings.Join(append(parts, strings.Join(lines, "\n")), "\n\n")
	if data.Edit.Kind == "" {
		return base
	}

	return renderEditorOverlay(base, data)
}

// renderRequestRow renders a standard single-line request row.
func renderRequestRow(row Row, selected bool) string {
	label := row.Label
	if row.Meta != "" {
		label += " (" + row.Meta + ")"
	}

	value := row.Value
	if !row.Editable && value != "" && row.Kind != RowKindAuthOption {
		value += " [read-only]"
	}

	line := label
	if value != "" {
		line += " = " + value
	}

	return widgets.RenderList([]widgets.ListItem{{
		Content:  line,
		Selected: selected,
		Muted:    !row.Editable,
	}})
}

// renderBodyPreviewRow renders the request body row with a multiline preview block.
func renderBodyPreviewRow(row Row, selected bool, contentWidth int) string {
	label := row.Label
	if row.Meta != "" {
		label += " (" + row.Meta + ")"
	}

	lines := []string{renderBodyPreviewHeader(label, row.Editable, selected, contentWidth), "\n"}
	for _, previewLine := range renderBodyPreviewLines(row.Value, !row.Editable) {
		lines = append(lines, previewLine)
	}

	return strings.Join(lines, "\n")
}

// renderBodyPreviewHeader renders the selectable body preview header row.
func renderBodyPreviewHeader(label string, editable, selected bool, contentWidth int) string {
	content := label + " ="
	if editable {
		hint := "Enter edit"
		content += " " + hint
	}

	return widgets.RenderList([]widgets.ListItem{{
		Content:  content,
		Selected: selected,
		Muted:    !editable,
		Width:    0,
	}})
}

// renderBodyPreviewLines renders the visible body preview lines for the body request row.
func renderBodyPreviewLines(value string, muted bool) []string {
	if strings.TrimSpace(value) == "" {
		value = "<empty>"
	}

	previewLines := strings.Split(value, "\n")
	rendered := make([]string, 0, len(previewLines))
	for _, line := range previewLines {
		rendered = append(rendered,
			lipgloss.JoinHorizontal(
				lipgloss.Top,
				widgets.MutedTextStyle().Render("│ "),
				bodyPreviewTextStyle(muted).Render(line),
			),
		)
	}

	return rendered
}

// bodyPreviewTextStyle returns the preview text style for body preview lines.
func bodyPreviewTextStyle(muted bool) lipgloss.Style {
	if muted {
		return widgets.MutedTextStyle()
	}

	return widgets.BodyTextStyle()
}

// renderEditorOverlay renders the active request editor popup over the base pane content.
func renderEditorOverlay(base string, data Data) string {
	editorPopup := widgets.RenderPopup(widgets.PopupData{
		Title:   data.Edit.Title,
		Body:    editorPopupBody(data),
		Width:   popupWidth(data),
		Focused: true,
	})

	return widgets.Overlay(base, editorPopup, popupX(data, editorPopup), popupY(data, editorPopup))
}

// requestActiveRowIndex clamps the rendered request row selection into the visible row list.
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

// fallbackText returns the trimmed value or the provided fallback when the value is blank.
func fallbackText(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}

	return value
}

// renderValidationSummary renders any request validation messages above the active section body.
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

// renderSupportSummary renders non-blocking request support notes above the active section body.
func renderSupportSummary(notes []SupportNote) string {
	if len(notes) == 0 {
		return ""
	}

	lines := []string{widgets.RenderMutedHeading("Support notes:")}
	for _, note := range notes {
		lines = append(lines, renderSupportMessage(note))
	}

	return strings.Join(lines, "\n")
}

// renderSupportMessage renders one non-blocking unsupported or downgraded support note.
func renderSupportMessage(note SupportNote) string {
	label := string(note.Severity)
	if label == "" {
		label = "note"
	}

	content := label + ": " + note.Summary
	if strings.TrimSpace(note.Detail) != "" {
		content += " " + note.Detail
	}

	return widgets.WarningTextStyle().Render(content)
}

// editorPopupBody renders the request editor popup body and optional context line.
func editorPopupBody(data Data) string {
	lines := make([]string, 0, 5)
	if strings.TrimSpace(data.Edit.Context) != "" {
		lines = append(lines, data.Edit.Context, "")
	}

	// preserve the widget's padded view so textarea backgrounds fill the full editor area.
	editor := data.Edit.View
	if editor == "" {
		editor = data.Edit.Buffer
	}
	if strings.TrimSpace(editor) == "" {
		editor = "<empty>"
	}

	lines = append(lines, editor)
	return strings.Join(lines, "\n")
}

// popupWidth returns the request editor popup width for the active editor kind.
func popupWidth(data Data) int {
	if data.Edit.Kind == "body" {
		// leave extra side padding for the larger body editor so the popup does not hug the pane frame.
		// return min(max(data.ContentWidth-8, 28), 84)
		return util.Clamp(data.ContentWidth-8, 28, 84)
	}

	// use slightly tighter padding for field editors so the popup stays close to the selected row.
	return util.Clamp(data.ContentWidth-10, 24, 64)
}

// popupX places the editor under the selected row.
func popupX(data Data, popup string) int {
	return 0
}

// popupY returns the request editor popup anchor inside the pane body.
func popupY(data Data, popup string) int {
	popupHeight := lipgloss.Height(popup)
	if data.Edit.Kind == "body" {
		// always anchor to the top of the pane body for the body editor.
		return 0
	}

	summaryHeight := 0
	if len(data.ValidationNotice) > 0 {
		summaryHeight = lipgloss.Height(renderValidationSummary(data.ValidationNotice)) + 2
	}
	if len(data.SupportNotice) > 0 {
		summaryHeight += lipgloss.Height(renderSupportSummary(data.SupportNotice)) + 2
	}
	target := summaryHeight + data.ActiveRow + 1
	maxY := max(data.ContentHeight-popupHeight, 0)
	if target > maxY {
		target = maxY
	}
	return max(target, 0)
}
