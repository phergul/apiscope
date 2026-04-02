package history

import (
	"fmt"
	"sort"
	"strings"

	"github.com/phergul/apiscope/internal/model"
	"github.com/phergul/apiscope/internal/tui/widgets"
	"github.com/phergul/apiscope/internal/util"

	"github.com/charmbracelet/lipgloss"
)

// PopupInput contains the root-owned state needed to project the previous-requests popup.
type PopupInput struct {
	Selected            *model.Operation
	Entries             []model.HistoryEntry
	ActiveRow           int
	PreviewScrollOffset int
	ContentWidth        int
	ContentHeight       int
}

// PopupData contains render-ready popup content plus selection metadata.
type PopupData struct {
	Title            string
	Meta             string
	Body             string
	ActiveRow        int
	MaxPreviewScroll int
}

// ProjectPopup projects operation-scoped history entries into popup render data.
func ProjectPopup(input PopupInput) PopupData {
	data := PopupData{
		Title: "Previous requests",
	}
	if input.Selected != nil {
		data.Meta = strings.ToUpper(input.Selected.Method) + " " + input.Selected.Path
	}

	switch {
	case input.Selected == nil:
		data.Body = fixedBodyBlock("No operation selected.\nChoose an operation in pane 1 to browse previous requests.", input.ContentWidth, input.ContentHeight)
		return data
	case len(input.Entries) == 0:
		data.Body = fixedBodyBlock("No previous requests for this operation yet.\nRun Ctrl+R to create the first history entry.", input.ContentWidth, input.ContentHeight)
		return data
	}

	activeRow := ClampActiveRow(input.Entries, input.ActiveRow)
	data.ActiveRow = activeRow
	data.Body, data.MaxPreviewScroll = renderPopupBody(input.Entries, activeRow, input.PreviewScrollOffset, input.ContentWidth, input.ContentHeight)
	return data
}

// ClampActiveRow keeps the popup selection inside the available entry range.
func ClampActiveRow(entries []model.HistoryEntry, activeRow int) int {
	if len(entries) == 0 {
		return 0
	}

	return util.Clamp(activeRow, 0, len(entries)-1)
}

// MoveActiveRow moves the popup selection by the provided delta.
func MoveActiveRow(entries []model.HistoryEntry, activeRow, delta int) int {
	return ClampActiveRow(entries, activeRow+delta)
}

// BoundaryActiveRow moves the popup selection to the first or last row.
func BoundaryActiveRow(entries []model.HistoryEntry, last bool) int {
	if last {
		return ClampActiveRow(entries, len(entries)-1)
	}

	return ClampActiveRow(entries, 0)
}

// ActiveEntry returns the selected history entry, if any.
func ActiveEntry(entries []model.HistoryEntry, activeRow int) (model.HistoryEntry, bool) {
	if len(entries) == 0 {
		return model.HistoryEntry{}, false
	}

	return entries[ClampActiveRow(entries, activeRow)], true
}

// renderPopupBody renders the split history picker body and returns its preview scroll bound.
func renderPopupBody(entries []model.HistoryEntry, activeRow, previewScrollOffset, contentWidth, contentHeight int) (string, int) {
	if contentWidth < 48 {
		return renderStackedPopupBody(entries, activeRow, previewScrollOffset, contentWidth, contentHeight)
	}

	leftWidth := max((contentWidth-1)/2, 20)
	rightWidth := max(contentWidth-leftWidth-1, 20)
	entry, _ := ActiveEntry(entries, activeRow)

	listBody := renderEntryList(entries, activeRow, leftWidth, contentHeight)
	previewBody, maxPreviewScroll := renderPreviewPane(entry, previewScrollOffset, rightWidth, contentHeight)

	leftBlock := widgets.PopupPreviewColumnStyle(leftWidth, contentHeight).Render(listBody)
	divider := widgets.PopupPreviewDividerStyle(contentHeight).Render(strings.Repeat("│\n", max(contentHeight-1, 0)) + "│")
	rightBlock := widgets.PopupPreviewColumnStyle(rightWidth, contentHeight).Render(previewBody)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftBlock, divider, rightBlock), maxPreviewScroll
}

// renderStackedPopupBody falls back to a stacked layout when the popup is too narrow to split.
func renderStackedPopupBody(entries []model.HistoryEntry, activeRow, previewScrollOffset, contentWidth, contentHeight int) (string, int) {
	entry, _ := ActiveEntry(entries, activeRow)
	listHeight := max(contentHeight/2, 3)
	previewHeight := max(contentHeight-listHeight-1, 1)
	listBody := renderEntryList(entries, activeRow, contentWidth, listHeight)
	previewBody, maxPreviewScroll := renderPreviewPane(entry, previewScrollOffset, contentWidth, previewHeight)

	body := lipgloss.JoinVertical(
		lipgloss.Left,
		widgets.PopupPreviewColumnStyle(contentWidth, listHeight).Render(listBody),
		widgets.PopupPreviewColumnStyle(contentWidth, 1).Render(""),
		widgets.PopupPreviewColumnStyle(contentWidth, previewHeight).Render(previewBody),
	)

	return body, maxPreviewScroll
}

// renderEntryList renders the selectable left-side history rows for the active operation.
func renderEntryList(entries []model.HistoryEntry, activeRow, contentWidth, contentHeight int) string {
	listWidth := contentWidth
	if listWidth <= 0 {
		listWidth = 28
	}

	items := make([]widgets.ListItem, 0, len(entries))
	for _, row := range visibleRows(entries, activeRow, contentHeight) {
		items = append(items, widgets.ListItem{
			Content:  historyRowLabel(row.Entry, listWidth-2),
			Selected: row.Index == activeRow,
			Muted:    row.Index != activeRow,
			Width:    listWidth,
		})
	}

	return lipgloss.NewStyle().Width(listWidth).MaxWidth(listWidth).Height(max(contentHeight, 1)).MaxHeight(max(contentHeight, 1)).Render(widgets.RenderList(items))
}

type indexedEntry struct {
	Index int
	Entry model.HistoryEntry
}

// visibleRows keeps the selected history row centered within the rendered list window when possible.
func visibleRows(entries []model.HistoryEntry, activeRow, maxRows int) []indexedEntry {
	if len(entries) <= maxRows {
		rows := make([]indexedEntry, 0, len(entries))
		for index, entry := range entries {
			rows = append(rows, indexedEntry{Index: index, Entry: entry})
		}
		return rows
	}

	activeRow = ClampActiveRow(entries, activeRow)
	start := util.Clamp(activeRow-maxRows/2, 0, len(entries)-maxRows)
	rows := make([]indexedEntry, 0, maxRows)
	for index := start; index < start+maxRows; index++ {
		rows = append(rows, indexedEntry{Index: index, Entry: entries[index]})
	}

	return rows
}

// historyRowLabel formats one compact picker row with request id, status, and server URL.
func historyRowLabel(entry model.HistoryEntry, width int) string {
	status := historyStatus(entry)
	server := strings.TrimSpace(entry.ServerURL)
	if server == "" {
		server = entry.Request.ServerURL
	}

	label := fmt.Sprintf("#%d  %s", entry.RequestID, status)
	if server == "" {
		return fitLine(label, width)
	}

	return fitLine(label+"  "+server, width)
}

// historyStatus returns the most useful short status label for one history row.
func historyStatus(entry model.HistoryEntry) string {
	if strings.TrimSpace(entry.TransportNote) != "" {
		return "transport failed"
	}
	if entry.Response == nil {
		return "no response"
	}
	if strings.TrimSpace(entry.Response.Status) != "" {
		return entry.Response.Status
	}
	if entry.Response.StatusCode > 0 {
		return fmt.Sprintf("%d", entry.Response.StatusCode)
	}

	return "completed"
}

// renderPreviewPane renders the right-side request or response preview through a clipped viewport.
func renderPreviewPane(entry model.HistoryEntry, scrollOffset, contentWidth, contentHeight int) (string, int) {
	lines := previewLines(entry, contentWidth)
	content := strings.Join(lines, "\n")
	maxScroll := max(len(lines)-max(contentHeight, 1), 0)

	viewport := widgets.NewViewport(max(contentWidth, 1), max(contentHeight, 1))
	viewport.SetContent(content)
	viewport.SetYOffset(util.Clamp(scrollOffset, 0, maxScroll))
	return viewport.View(), maxScroll
}

// previewLines builds the stacked request and response preview lines for the selected entry.
func previewLines(entry model.HistoryEntry, contentWidth int) []string {
	lines := make([]string, 0, 32)
	lines = append(lines, renderPreviewHeading("Request"), "")
	lines = append(lines, wrapLines([]string{
		renderPreviewMetaLine("Request ID", fmt.Sprintf("%d", entry.RequestID)),
		renderPreviewMetaLine("Server", util.FallbackText(entry.Request.ServerURL, entry.ServerURL)),
	}, contentWidth)...)
	for _, section := range requestSections(entry.Request.Draft) {
		lines = append(lines, "")
		lines = append(lines, wrapLines(section, contentWidth)...)
	}
	if authLine := authSummary(entry.Request.AuthState); authLine != "" {
		lines = append(lines, "")
		lines = append(lines, wrapLines([]string{renderPreviewMetaLine("Auth", authLine)}, contentWidth)...)
	}

	lines = append(lines, "", renderPreviewHeading("Response"), "")
	lines = append(lines, wrapLines(responseLines(entry), contentWidth)...)
	return lines
}

// responseLines formats the stored response summary, headers, and body preview for one entry.
func responseLines(entry model.HistoryEntry) []string {
	lines := make([]string, 0, 20)
	transport := strings.TrimSpace(entry.TransportNote)
	if transport == "" && entry.Response != nil {
		transport = strings.TrimSpace(entry.Response.TransportError)
	}
	if transport != "" {
		lines = append(lines, renderPreviewMetaLine("Transport", transport))
	}
	if entry.Response == nil {
		if transport == "" {
			lines = append(lines, widgets.BodyTextStyle().Render("No stored response."))
		}
		return lines
	}

	response := entry.Response
	status := strings.TrimSpace(response.Status)
	if status == "" && response.StatusCode > 0 {
		status = fmt.Sprintf("%d", response.StatusCode)
	}
	if status != "" {
		lines = append(lines, renderPreviewMetaLine("Status", status))
	}
	if response.Duration > 0 {
		lines = append(lines, renderPreviewMetaLine("Duration", response.Duration.String()))
	}
	if strings.TrimSpace(response.ContentType) != "" {
		lines = append(lines, renderPreviewMetaLine("Content type", response.ContentType))
	}
	if response.ContentLength > 0 {
		lines = append(lines, renderPreviewMetaLine("Content length", fmt.Sprintf("%d", response.ContentLength)))
	}
	if len(response.Headers) > 0 {
		lines = append(lines, "", renderPreviewHeading("Headers:"))
		lines = append(lines, headerLines(response.Headers)...)
	}
	if preview := responseBodyPreview(response); preview != "" {
		lines = append(lines, "", renderPreviewHeading("Body preview:"))
		for line := range strings.SplitSeq(preview, "\n") {
			lines = append(lines, "  "+widgets.BodyTextStyle().Render(line))
		}
	}

	return lines
}

// headerLines renders stored response headers in a stable alphabetical order.
func headerLines(headers map[string][]string) []string {
	names := make([]string, 0, len(headers))
	for name := range headers {
		names = append(names, name)
	}
	sort.Strings(names)

	lines := make([]string, 0, len(names))
	for _, name := range names {
		lines = append(lines, "  "+renderPreviewMetaLine(name, strings.Join(headers[name], ", ")))
	}
	return lines
}

// responseBodyPreview returns a short body preview for the stored response, when available.
func responseBodyPreview(response *model.HTTPResponse) string {
	if response == nil {
		return ""
	}

	body := strings.TrimSpace(response.PrettyBody)
	if body == "" {
		body = strings.TrimSpace(string(response.Body))
	}
	return bodyPreview(body)
}

// requestSections formats grouped request-input sections for the selected history entry.
func requestSections(draft model.RequestDraft) [][]string {
	sections := make([][]string, 0, 6)
	if line := formatParamValues(draft.PathParams); line != "" {
		sections = append(sections, []string{renderPreviewMetaLine("Path", line)})
	}
	if line := formatParamValues(draft.QueryParams); line != "" {
		sections = append(sections, []string{renderPreviewMetaLine("Query", line)})
	}
	if line := formatParamValues(draft.HeaderParams); line != "" {
		sections = append(sections, []string{renderPreviewMetaLine("Header", line)})
	}
	if line := formatParamValues(draft.CookieParams); line != "" {
		sections = append(sections, []string{renderPreviewMetaLine("Cookie", line)})
	}
	if line := formatParamValues(draft.FormParams); line != "" {
		sections = append(sections, []string{renderPreviewMetaLine("Form", line)})
	}
	if line := formatParamValues(draft.FormFileParams); line != "" {
		sections = append(sections, []string{renderPreviewMetaLine("Files", line)})
	}
	if strings.TrimSpace(draft.BodyMediaType) != "" {
		sections = append(sections, []string{renderPreviewMetaLine("Body media type", draft.BodyMediaType)})
	}
	if preview := bodyPreview(draft.BodyRaw); preview != "" {
		lines := []string{renderPreviewHeading("Body preview:")}
		for line := range strings.SplitSeq(preview, "\n") {
			lines = append(lines, "  "+widgets.BodyTextStyle().Render(line))
		}
		sections = append(sections, lines)
	}

	return sections
}

// formatParamValues joins one request-input map into a stable user-facing summary value.
func formatParamValues(values map[string]string) string {
	if len(values) == 0 {
		return ""
	}

	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+values[key])
	}

	return strings.Join(parts, ", ")
}

// bodyPreview trims a multiline body down to a short preview block for popup rendering.
func bodyPreview(body string) string {
	body = widgets.NormalizeRenderedBody(body)
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}

	lines := strings.Split(body, "\n")
	// if len(lines) > 2 {
	// 	lines = append(lines[:2], "...")
	// }

	return strings.Join(lines, "\n")
}

// authSummary returns one concise summary of the configured auth inputs for an executed request.
func authSummary(state map[string]model.AuthValue) string {
	if len(state) == 0 {
		return ""
	}

	names := make([]string, 0, len(state))
	for name := range state {
		names = append(names, name)
	}
	sort.Strings(names)

	parts := make([]string, 0, len(names))
	for _, name := range names {
		value := state[name]
		switch {
		case strings.TrimSpace(value.APIKey) != "":
			parts = append(parts, name+" key set")
		case strings.TrimSpace(value.BearerToken) != "":
			parts = append(parts, name+" token set")
		case strings.TrimSpace(value.Username) != "" || strings.TrimSpace(value.Password) != "":
			parts = append(parts, name+" credentials set")
		default:
			parts = append(parts, name+" configured")
		}
	}

	return strings.Join(parts, ", ")
}

// fixedBodyBlock keeps popup empty states inside the requested body box when size is known.
func fixedBodyBlock(body string, width, height int) string {
	if width <= 0 || height <= 0 {
		return body
	}

	return lipgloss.NewStyle().Width(max(width, 1)).MaxWidth(max(width, 1)).Height(max(height, 1)).MaxHeight(max(height, 1)).Render(body)
}

// wrapLines wraps preview lines to the available pane width before viewport clipping.
func wrapLines(lines []string, width int) []string {
	return widgets.WrapLines(lines, width)
}

// fitLine clips one list row to the requested width while preserving the surrounding style.
func fitLine(line string, width int) string {
	return widgets.FitLine(line, width)
}

// renderPreviewHeading styles one preview heading to match the muted response-pane headings.
func renderPreviewHeading(content string) string {
	return widgets.MutedTextStyle().Render(content)
}

// renderPreviewMetaLine styles one preview label and value pair like the response pane.
func renderPreviewMetaLine(label, value string) string {
	return widgets.MutedTextStyle().Render(label+": ") + widgets.BodyTextStyle().Render(value)
}
