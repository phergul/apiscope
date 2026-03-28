package panes

import (
	"fmt"
	"strings"

	"github.com/phergul/apiscope/internal/tui/widgets"

	"github.com/charmbracelet/lipgloss"
)

type OperationRow struct {
	Method   string
	Path     string
	Selected bool
}

type OperationsGroup struct {
	Name string
	Rows []OperationRow
}

type OperationsData struct {
	LoadInFlight    bool
	LoadFailed      bool
	HasSpec         bool
	ContentWidth    int
	ScrollOffset    int
	MaxLines        int
	TotalOperations int
	Groups          []OperationsGroup
}

func RenderOperations(data OperationsData) string {
	switch {
	case data.LoadInFlight:
		return "Loading spec..."
	case data.LoadFailed:
		return "Spec load failed.\nSee pane 2 for details and recovery steps."
	case !data.HasSpec:
		return "No spec loaded."
	}

	lines := []string{}

	if data.TotalOperations == 0 {
		lines = append(lines, "This spec loaded successfully, but it does not define any operations.")
		return strings.Join(lines, "\n")
	}
	if len(data.Groups) == 0 {
		lines = append(lines, "No operations match the current filter.", "Press Esc to clear the filter.")
		return strings.Join(lines, "\n")
	}

	lines, _ = collectOperationLines(data)
	return strings.Join(lines, "\n")
}

func VisibleOperationRowCount(data OperationsData) int {
	switch {
	case data.LoadInFlight, data.LoadFailed, !data.HasSpec, data.TotalOperations == 0, len(data.Groups) == 0:
		return 0
	}

	_, rowCount := collectOperationLines(data)
	return rowCount
}

func collectOperationLines(data OperationsData) ([]string, int) {
	lines := []string{}
	skippedRows := 0
	usedLines := 0
	renderedRows := 0
	stop := false
	for _, group := range data.Groups {
		groupLines := []string{}
		groupLineCount := 0

		for _, row := range group.Rows {
			if skippedRows < data.ScrollOffset {
				skippedRows++
				continue
			}

			rendered := renderOperationRow(row, data.ContentWidth)
			rowHeight := lipgloss.Height(rendered)
			additionalLines := rowHeight
			if len(groupLines) == 0 {
				additionalLines++
			}
			if usedLines > 0 && len(groupLines) == 0 {
				additionalLines++
			}
			if data.MaxLines > 0 && renderedRows > 0 && usedLines+groupLineCount+additionalLines > data.MaxLines {
				stop = true
				break
			}

			if len(groupLines) == 0 {
				if usedLines > 0 {
					groupLines = append(groupLines, "")
					groupLineCount++
				}
				groupLines = append(groupLines, widgets.RenderMutedHeading(strings.ToUpper(group.Name)))
				groupLineCount++
			}

			groupLines = append(groupLines, rendered)
			groupLineCount += rowHeight
			renderedRows++
		}
		if len(groupLines) > 0 {
			lines = append(lines, groupLines...)
			usedLines += groupLineCount
		}
		if stop {
			break
		}
	}

	return lines, renderedRows
}

func renderOperationRow(row OperationRow, width int) string {
	methodLabel := fmt.Sprintf(" %-6s ", strings.ToUpper(row.Method))
	methodStyle := lipgloss.NewStyle().Foreground(widgets.MethodColor(row.Method)).Bold(true)
	pathStyle := widgets.BodyTextStyle()
	rowStyle := widgets.BodyTextStyle()

	if row.Selected {
		methodStyle = methodStyle.Background(widgets.CurrentTheme().Palette.Selection)
		pathStyle = widgets.SelectedTextStyle()
		rowStyle = widgets.SelectedTextStyle()
	}

	content := fmt.Sprintf("%s%s", methodStyle.Render(methodLabel), pathStyle.Render(row.Path))
	if width > 0 {
		rowStyle = rowStyle.Width(width).MaxWidth(width)
	}

	return rowStyle.Render(content)
}
