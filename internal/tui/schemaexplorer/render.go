package schemaexplorer

import (
	"strings"

	"github.com/phergul/apiscope/internal/tui/widgets"
	"github.com/phergul/apiscope/internal/util"

	"github.com/charmbracelet/lipgloss"
)

const (
	openMarker         = "[-]"
	closedMarker       = "[+]"
	leafMarker         = "   "
	treeGuideSegment   = "│  "
	treeBlankSegment   = "   "
	treeBranchSegment  = "├─ "
	treeLastBranchLine = "└─ "
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

func renderRows(rows []visibleRow, activeRow, scrollOffset, viewportRows, width int) string {
	if len(rows) == 0 {
		return widgets.MutedTextStyle().Render("No schemas available.")
	}

	start := util.Clamp(scrollOffset, 0, max(len(rows)-viewportRows, 0))
	end := min(start+viewportRows, len(rows))
	lines := make([]string, 0, end-start)
	for index := start; index < end; index++ {
		lines = append(lines, renderRow(rows[index], index == util.Clamp(activeRow, 0, len(rows)-1), width))
	}

	return strings.Join(lines, "\n")
}

func renderRow(row visibleRow, selected bool, width int) string {
	content := renderTreeLine(row, selected)
	return widgets.FitLine(content, max(width, 1))
}

func renderTreeLine(row visibleRow, selected bool) string {
	if row.Node == nil {
		return ""
	}

	var builder strings.Builder
	builder.WriteString(renderTreePrefix(row, selected))
	builder.WriteString(renderMarker(row, selected))
	builder.WriteString(" ")

	if row.Depth == 0 {
		builder.WriteString(groupLabelStyle(selected).Render(row.Node.Label.Name))
		return builder.String()
	}

	if prefix := strings.TrimSpace(row.Node.Label.Prefix); prefix != "" {
		builder.WriteString(mutedRowStyle(selected).Render(prefix + " "))
	}
	if name := strings.TrimSpace(row.Node.Label.Name); name != "" {
		builder.WriteString(nameRowStyle(selected).Render(name))
	}
	if meta := strings.TrimSpace(row.Node.Label.Meta); meta != "" {
		builder.WriteString(mutedRowStyle(selected).Render(" (" + meta + ")"))
	}
	if note := strings.TrimSpace(row.Node.Note); note != "" {
		builder.WriteString(noteRowStyle(selected).Render(" - " + note))
	}

	return builder.String()
}

func renderTreePrefix(row visibleRow, selected bool) string {
	if row.Depth == 0 {
		return ""
	}

	chrome := mutedRowStyle(selected)
	var builder strings.Builder
	for _, ancestorHasNext := range row.AncestorHasNext {
		if ancestorHasNext {
			builder.WriteString(chrome.Render(treeGuideSegment))
			continue
		}
		builder.WriteString(treeBlankSegment)
	}
	if row.HasNextSibling {
		builder.WriteString(chrome.Render(treeBranchSegment))
	} else {
		builder.WriteString(chrome.Render(treeLastBranchLine))
	}

	return builder.String()
}

func renderMarker(row visibleRow, selected bool) string {
	style := mutedRowStyle(selected)
	switch {
	case expandable(row.Node) && row.Expanded:
		return style.Render(openMarker)
	case expandable(row.Node):
		return style.Render(closedMarker)
	default:
		return style.Render(leafMarker)
	}
}

func groupLabelStyle(selected bool) lipgloss.Style {
	if selected {
		return widgets.SelectedTextStyle().Bold(true)
	}

	return widgets.BodyTextStyle().Bold(true)
}

func nameRowStyle(selected bool) lipgloss.Style {
	if selected {
		return widgets.SelectedTextStyle().Bold(true)
	}

	return widgets.BodyTextStyle()
}

func mutedRowStyle(selected bool) lipgloss.Style {
	if selected {
		return widgets.SelectedTextStyle()
	}

	return widgets.MutedTextStyle()
}

func noteRowStyle(selected bool) lipgloss.Style {
	if selected {
		return widgets.SelectedTextStyle().Italic(true)
	}

	return widgets.WarningTextStyle()
}

func columnWidths(contentWidth int) (int, int) {
	contentWidth = max(contentWidth, 24)
	leftWidth := min(max(int(float64(contentWidth)*0.38), 32), 50)
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
