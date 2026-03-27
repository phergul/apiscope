package panes

import (
	"fmt"
	"strings"
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

	filterValue := fallbackText(data.FilterText, "None")
	if data.FilterEditing {
		filterValue += " (editing)"
	}

	lines := []string{
		fmt.Sprintf("Filter: %s", filterValue),
		"",
	}

	if data.TotalOperations == 0 {
		lines = append(lines, "This spec loaded successfully, but it does not define any operations.")
		return strings.Join(lines, "\n")
	}
	if len(data.Groups) == 0 {
		lines = append(lines, "No operations match the current filter.", "Press Esc to clear the filter.")
		return strings.Join(lines, "\n")
	}

	for _, group := range data.Groups {
		lines = append(lines, strings.ToUpper(group.Name))
		for _, row := range group.Rows {
			prefix := "  "
			if row.Selected {
				prefix = "> "
			}

			lines = append(lines, fmt.Sprintf("%s%-6s %s", prefix, strings.ToUpper(row.Method), row.Path))
		}
		lines = append(lines, "")
	}
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	return strings.Join(lines, "\n")
}
