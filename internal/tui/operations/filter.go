package operations

import (
	"strings"
	"unicode/utf8"

	"github.com/phergul/apiscope/internal/model"
	"github.com/phergul/apiscope/internal/tui/widgets"

	tea "github.com/charmbracelet/bubbletea"
)

// FilterFooterInput contains the state needed to render the operations footer.
type FilterFooterInput struct {
	Editing    bool
	FilterText string
	EditorView string
}

// FilterEditorUpdate describes the result of handling a filter-editor keypress.
type FilterEditorUpdate struct {
	FilterText      string
	Editing         bool
	Quit            bool
	UseWidgetUpdate bool
	RefreshVisible  bool
}

// FilterVisibleKeys returns the grouped visible operation keys for the current filter.
func FilterVisibleKeys(operations []model.Operation, filter string) []model.OperationKey {
	filter = strings.TrimSpace(strings.ToLower(filter))
	visible := make([]model.OperationKey, 0, len(operations))
	for _, operation := range operations {
		if filter == "" || MatchFilter(operation, filter) {
			visible = append(visible, operation.Key)
		}
	}

	return FlattenKeys(GroupKeys(visible, operations))
}

// RenderFooter renders the operations footer for the current filter state.
func RenderFooter(input FilterFooterInput) string {
	if !input.Editing && strings.TrimSpace(input.FilterText) == "" {
		return ""
	}
	if input.Editing {
		return input.EditorView
	}

	return widgets.InputTextStyle().Render("Filter: " + input.FilterText)
}

// UpdateFilterEditor handles one keypress in the operations filter editor.
func UpdateFilterEditor(msg tea.KeyMsg, filterText string) FilterEditorUpdate {
	result := FilterEditorUpdate{
		FilterText: filterText,
		Editing:    true,
	}

	switch msg.String() {
	case "ctrl+c":
		result.Quit = true
		return result
	case "enter":
		result.Editing = false
		return result
	case "esc":
		result.FilterText = ""
		result.Editing = false
		result.RefreshVisible = true
		return result
	case "backspace", "ctrl+h", "delete":
		result.FilterText = trimLastRune(filterText)
		result.RefreshVisible = true
		return result
	default:
		result.UseWidgetUpdate = true
		result.RefreshVisible = true
		return result
	}
}

// trimLastRune removes the trailing rune from a filter string.
func trimLastRune(value string) string {
	if value == "" {
		return ""
	}

	_, size := utf8.DecodeLastRuneInString(value)
	if size <= 0 {
		return ""
	}

	return value[:len(value)-size]
}
