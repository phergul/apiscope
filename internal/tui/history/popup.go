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
	Selected      *model.Operation
	Entries       []model.HistoryEntry
	ActiveRow     int
	ContentWidth  int
	ContentHeight int
}

// PopupData contains render-ready popup content plus selection metadata.
type PopupData struct {
	Title     string
	Meta      string
	Body      string
	Help      string
	ActiveRow int
}

// ProjectPopup projects operation-scoped history entries into popup render data.
func ProjectPopup(input PopupInput) PopupData {
	data := PopupData{
		Title: "Previous requests",
		Help:  "Enter load response\nr restore request\nEsc close",
	}
	if input.Selected != nil {
		data.Meta = strings.ToUpper(input.Selected.Method) + " " + input.Selected.Path
	}
	if input.Selected == nil {
		data.Body = "No operation selected.\nChoose an operation in pane 1 to browse previous requests."
		return data
	}
	if len(input.Entries) == 0 {
		data.Body = "No previous requests for this operation yet.\nRun Ctrl+R to create the first history entry."
		return data
	}

	activeRow := ClampActiveRow(input.Entries, input.ActiveRow)
	data.ActiveRow = activeRow
	data.Body = renderPopupBody(input.Entries, activeRow, input.ContentWidth, input.ContentHeight)
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

func renderPopupBody(entries []model.HistoryEntry, activeRow, contentWidth, contentHeight int) string {
	activeRow = ClampActiveRow(entries, activeRow)
	listBody := renderEntryList(entries, activeRow, contentWidth, contentHeight)
	entry, _ := ActiveEntry(entries, activeRow)
	detailBody := renderEntryDetails(entry, contentWidth)
	if detailBody == "" {
		return "Requests\n" + listBody
	}

	return strings.Join([]string{
		"Requests",
		listBody,
		"",
		"Selected request",
		detailBody,
	}, "\n")
}

func renderEntryList(entries []model.HistoryEntry, activeRow, contentWidth, contentHeight int) string {
	listWidth := contentWidth
	if listWidth <= 0 {
		listWidth = 28
	}
	items := make([]widgets.ListItem, 0, len(entries))
	for _, row := range visibleRows(entries, activeRow, listHeight(contentHeight)) {
		items = append(items, widgets.ListItem{
			Content:  historyRowLabel(row.Entry, listWidth),
			Selected: row.Index == activeRow,
			Muted:    row.Index != activeRow,
			Width:    listWidth,
		})
	}

	return widgets.RenderList(items)
}

func listHeight(contentHeight int) int {
	if contentHeight <= 0 {
		return 8
	}

	return max(contentHeight-10, 3)
}

type indexedEntry struct {
	Index int
	Entry model.HistoryEntry
}

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

func renderEntryDetails(entry model.HistoryEntry, contentWidth int) string {
	width := contentWidth
	if width <= 0 {
		width = 24
	}

	lines := []string{
		"Server: " + fallback(entry.Request.ServerURL, entry.ServerURL),
	}

	for _, section := range requestSections(entry.Request.Draft) {
		lines = append(lines, section...)
	}

	authLine := authSummary(entry.Request.AuthState)
	if authLine != "" {
		lines = append(lines, "Auth: "+authLine)
	}

	return joinClippedLines(lines, width, 8)
}

func requestSections(draft model.RequestDraft) [][]string {
	sections := make([][]string, 0, 6)
	if line := formatParamLine("Path", draft.PathParams); line != "" {
		sections = append(sections, []string{line})
	}
	if line := formatParamLine("Query", draft.QueryParams); line != "" {
		sections = append(sections, []string{line})
	}
	if line := formatParamLine("Header", draft.HeaderParams); line != "" {
		sections = append(sections, []string{line})
	}
	if line := formatParamLine("Cookie", draft.CookieParams); line != "" {
		sections = append(sections, []string{line})
	}
	if line := formatParamLine("Form", draft.FormParams); line != "" {
		sections = append(sections, []string{line})
	}
	if strings.TrimSpace(draft.BodyMediaType) != "" {
		sections = append(sections, []string{"Body media type: " + draft.BodyMediaType})
	}
	if preview := bodyPreview(draft.BodyRaw); preview != "" {
		lines := []string{"Body preview:"}
		for _, line := range strings.Split(preview, "\n") {
			lines = append(lines, "  "+line)
		}
		sections = append(sections, lines)
	}

	return sections
}

func formatParamLine(label string, values map[string]string) string {
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

	return label + ": " + strings.Join(parts, ", ")
}

func bodyPreview(body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}

	lines := strings.Split(body, "\n")
	if len(lines) > 2 {
		lines = append(lines[:2], "...")
	}

	return strings.Join(lines, "\n")
}

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

func joinClippedLines(lines []string, width, limit int) string {
	clipped := make([]string, 0, min(len(lines), limit))
	for _, line := range lines {
		clipped = append(clipped, fitLine(line, width))
		if len(clipped) == limit {
			break
		}
	}

	return strings.Join(clipped, "\n")
}

func fitLine(line string, width int) string {
	if width <= 0 {
		return line
	}

	return lipgloss.NewStyle().Width(width).MaxWidth(width).Render(line)
}

func fallback(primary, secondary string) string {
	if strings.TrimSpace(primary) != "" {
		return primary
	}

	return secondary
}
