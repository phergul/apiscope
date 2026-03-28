package panes

import (
	"fmt"
	"strings"

	"github.com/phergul/apiscope/internal/tui/widgets"
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
	FilterText      string
	FilterEditing   bool
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

		items := make([]widgets.ListItem, 0, len(group.Rows))
		for _, row := range group.Rows {
			items = append(items, widgets.ListItem{
				Content:  fmt.Sprintf("%s %s", widgets.RenderHTTPMethod(row.Method, 6), row.Path),
				Selected: row.Selected,
			})
		}
		lines = append(lines, strings.Split(widgets.RenderList(items), "\n")...)
		lines = append(lines, "")
	}
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	return strings.Join(lines, "\n")
}
