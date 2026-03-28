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

	for _, group := range data.Groups {
		lines = append(lines, widgets.RenderMutedHeading(strings.ToUpper(group.Name)))

		for _, row := range group.Rows {
			lines = append(lines, renderOperationRow(row, data.ContentWidth))
		}
		lines = append(lines, "")
	}
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	return strings.Join(lines, "\n")
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
