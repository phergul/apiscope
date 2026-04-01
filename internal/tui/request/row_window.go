package request

import "strings"

// WindowVisibleRows slices the request rows to the visible window and rebases the active row.
func WindowVisibleRows(data Data, scrollOffset, visibleLines int) Data {
	// body editing renders a popup over the full section, so row windowing does not apply there.
	if len(data.Rows) == 0 || data.Edit.Kind == "body" {
		return data
	}
	if visibleLines <= 0 {
		visibleLines = 1
	}

	offset := scrollOffset
	if offset < 0 {
		offset = 0
	}
	if offset >= len(data.Rows) {
		offset = len(data.Rows) - 1
	}

	end := offset + visibleLines
	if end > len(data.Rows) {
		end = len(data.Rows)
	}

	if data.ActiveRow >= 0 {
		data.ActiveRow = ClampActiveRow(data.ActiveRow, len(data.Rows)) - offset
	}
	data.Rows = data.Rows[offset:end]
	return data
}

// MaxActiveSectionScrollOffset returns the maximum vertical scroll offset for the active request section body.
func MaxActiveSectionScrollOffset(data Data, visibleLines int) int {
	lines := len(splitRenderedLines(RenderActiveSection(data)))
	if lines <= visibleLines {
		return 0
	}

	return lines - visibleLines
}

// splitRenderedLines splits rendered content into display lines while preserving empty content.
func splitRenderedLines(text string) []string {
	if text == "" {
		return []string{""}
	}

	return strings.Split(text, "\n")
}
