package operations

import (
	"strings"

	"github.com/phergul/apiscope/internal/tui/widgets"
)

// BuildBrowseHelpView returns the contextual help for operations browsing.
func BuildBrowseHelpView() widgets.HelpView {
	return widgets.HelpView{
		Hint:  "Help - ?",
		Title: "Operations help",
		Body: strings.Join([]string{
			"j/k or up/down move selection",
			"[ or ] or h/l jump groups",
			"/ filter operations",
			"1-4 or Tab / Shift+Tab focus panes",
			"z zoom focused pane",
			"? or Esc close help",
		}, "\n"),
	}
}

// BuildFilterHelpView returns the contextual help for filter editing.
func BuildFilterHelpView() widgets.HelpView {
	return widgets.HelpView{
		Hint:  "Help - ?",
		Title: "Filter help",
		Body: strings.Join([]string{
			"Type to filter operations",
			"Backspace/Delete edit filter",
			"Enter apply filter",
			"Esc clear and close filter",
			"? or Esc close help",
		}, "\n"),
	}
}
