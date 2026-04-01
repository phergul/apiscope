package details

import (
	"strings"

	"github.com/phergul/apiscope/internal/tui/widgets"
)

// BuildBrowseHelpView returns the contextual help for details browsing.
func BuildBrowseHelpView() widgets.HelpView {
	return widgets.HelpView{
		Hint:  "Help - ?",
		Title: "Details help",
		Body: strings.Join([]string{
			"j/k or up/down scroll",
			"[ or ] or h/l switch sections",
			"1-4 or Tab / Shift+Tab focus panes",
			"z zoom focused pane",
			"? or Esc close help",
		}, "\n"),
	}
}
