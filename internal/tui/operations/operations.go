package operations

import (
	"fmt"
	"strings"

	"github.com/phergul/apiscope/internal/model"
	"github.com/phergul/apiscope/internal/tui/widgets"

	"github.com/charmbracelet/lipgloss"
)

// Row describes one rendered operations row.
type Row struct {
	Method   string
	Path     string
	Selected bool
}

// Group describes one rendered operations group.
type Group struct {
	Name string
	Rows []Row
}

// KeyGroup groups visible operation keys by rendered group name.
type KeyGroup struct {
	Name string
	Keys []model.OperationKey
}

// Data contains the render-ready state for the operations pane.
type Data struct {
	LoadInFlight    bool
	LoadFailed      bool
	HasSpec         bool
	ContentWidth    int
	ScrollOffset    int
	MaxLines        int
	TotalOperations int
	Groups          []Group
}

// MatchFilter reports whether an operation matches the current filter text.
func MatchFilter(operation model.Operation, filter string) bool {
	if filter == "" {
		return true
	}

	fields := []string{
		operation.Method,
		operation.Path,
		operation.Summary,
	}
	fields = append(fields, operation.Tags...)

	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), filter) {
			return true
		}
	}

	return false
}

// GroupKeys groups visible operation keys by their rendered group name.
func GroupKeys(keys []model.OperationKey, operations []model.Operation) []KeyGroup {
	if len(keys) == 0 {
		return nil
	}

	lookup := operationMap(operations)
	groups := make([]KeyGroup, 0)
	indexByName := make(map[string]int)
	for _, key := range keys {
		operation, ok := lookup[key]
		if !ok {
			continue
		}

		groupName := GroupName(&operation)
		groupIndex, ok := indexByName[groupName]
		if !ok {
			groupIndex = len(groups)
			indexByName[groupName] = groupIndex
			groups = append(groups, KeyGroup{Name: groupName})
		}

		groups[groupIndex].Keys = append(groups[groupIndex].Keys, key)
	}

	return groups
}

// FlattenKeys flattens grouped operation keys into their rendered order.
func FlattenKeys(groups []KeyGroup) []model.OperationKey {
	total := 0
	for _, group := range groups {
		total += len(group.Keys)
	}

	ordered := make([]model.OperationKey, 0, total)
	for _, group := range groups {
		ordered = append(ordered, group.Keys...)
	}

	return ordered
}

// GroupName returns the rendered group name for an operation.
func GroupName(operation *model.Operation) string {
	if operation == nil || len(operation.Tags) == 0 || strings.TrimSpace(operation.Tags[0]) == "" {
		return "Untagged"
	}

	return operation.Tags[0]
}

// Render renders the operations pane from its render-ready data.
func Render(data Data) string {
	switch {
	case data.LoadInFlight:
		return "Loading spec..."
	case data.LoadFailed:
		return "Spec load failed.\nSee pane 2 for details and recovery steps."
	case !data.HasSpec:
		return "No spec loaded."
	}

	if data.TotalOperations == 0 {
		return "This spec loaded successfully, but it does not define any operations."
	}
	if len(data.Groups) == 0 {
		return strings.Join([]string{
			"No operations match the current filter.",
			"Press Esc to clear the filter.",
		}, "\n")
	}

	lines, _ := collectLines(data)
	return strings.Join(lines, "\n")
}

// VisibleRowCount returns the number of visible operation rows for the projected pane data.
func VisibleRowCount(data Data) int {
	switch {
	case data.LoadInFlight, data.LoadFailed, !data.HasSpec, data.TotalOperations == 0, len(data.Groups) == 0:
		return 0
	}

	_, rowCount := collectLines(data)
	return rowCount
}

// collectLines renders the grouped operation rows into display lines.
func collectLines(data Data) ([]string, int) {
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

			rendered := renderRow(row, data.ContentWidth)
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

// renderRow renders one operation row with the current selection styling.
func renderRow(row Row, width int) string {
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
		rowStyle = rowStyle.MaxWidth(width)
	}

	return rowStyle.Render(content)
}

// operationMap builds a by-key lookup for operations projection helpers.
func operationMap(operations []model.Operation) map[model.OperationKey]model.Operation {
	lookup := make(map[model.OperationKey]model.Operation, len(operations))
	for _, operation := range operations {
		lookup[operation.Key] = operation
	}

	return lookup
}
