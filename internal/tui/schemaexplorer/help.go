package schemaexplorer

import (
	"strings"

	"github.com/phergul/apiscope/internal/tui/widgets"
)

// BuildHelpView returns the contextual help for full-window schema exploration.
func BuildHelpView() widgets.HelpView {
	return widgets.HelpView{
		Hint:  "Help - ?",
		Title: "Schema explorer help",
		Body: strings.Join([]string{
			"j/k or up/down move selection",
			"Enter drill into selected schema",
			"h, left, or backspace go back",
			"Home / End jump to first or last row",
			"Ctrl+U / Ctrl+D scroll preview",
			"Esc close schema explorer",
			"t / T switch theme",
			"? or Esc close help",
		}, "\n"),
	}
}
