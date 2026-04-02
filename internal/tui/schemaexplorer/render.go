package schemaexplorer

import (
	"strings"

	"github.com/phergul/apiscope/internal/tui/widgets"
	"github.com/phergul/apiscope/internal/util"

	"github.com/charmbracelet/lipgloss"
)

// Render draws the two-column schema explorer body for the current shell mode.
func Render(data Data) string {
	if strings.TrimSpace(data.EmptyState) != "" {
		return data.EmptyState
	}

	left := lipgloss.NewStyle().
		Width(max(data.LeftWidth, 1)).
		MaxWidth(max(data.LeftWidth, 1)).
		Render(strings.Join([]string{
			widgets.MutedTextStyle().Bold(true).Render(data.LeftTitle),
			"",
			data.LeftBody,
		}, "\n"))

	right := lipgloss.NewStyle().
		Width(max(data.RightWidth, 1)).
		MaxWidth(max(data.RightWidth, 1)).
		Render(strings.Join([]string{
			widgets.MutedTextStyle().Bold(true).Render(data.RightTitle),
			"",
			data.RightBody,
		}, "\n"))

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		left,
		" "+widgets.MutedTextStyle().Render("│")+" ",
		right,
	)
}

func renderRows(rows []row, activeRow, scrollOffset, visibleRows, width int) string {
	if len(rows) == 0 {
		return widgets.MutedTextStyle().Render("No nested schemas.")
	}

	start := util.Clamp(scrollOffset, 0, max(len(rows)-visibleRows, 0))
	end := min(start+visibleRows, len(rows))
	items := make([]widgets.ListItem, 0, end-start)
	for index := start; index < end; index++ {
		content := rows[index].Label
		if strings.TrimSpace(rows[index].Meta) != "" {
			content += " (" + rows[index].Meta + ")"
		}
		if strings.TrimSpace(rows[index].Note) != "" {
			content += " - " + rows[index].Note
		}

		items = append(items, widgets.ListItem{
			Content:  content,
			Selected: index == util.Clamp(activeRow, 0, len(rows)-1),
			Muted:    !rows[index].Drillable,
			Width:    max(width, 1),
		})
	}

	return widgets.RenderList(items)
}

func columnWidths(contentWidth int) (int, int) {
	contentWidth = max(contentWidth, 24)
	leftWidth := min(max(int(float64(contentWidth)*0.34), 28), 42)
	if leftWidth > contentWidth-20 {
		leftWidth = max(contentWidth/3, 20)
	}
	rightWidth := max(contentWidth-leftWidth-3, 20)
	return leftWidth, rightWidth
}

func scrollLines(text string) int {
	if text == "" {
		return 1
	}

	return len(strings.Split(text, "\n"))
}
