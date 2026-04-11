package response

import (
	"strings"

	"github.com/phergul/apiscope/internal/tui/widgets"
)

// BuildBrowseHelpView returns the contextual help for response browsing.
func BuildBrowseHelpView() widgets.HelpView {
	return widgets.HelpView{
		Hint:  "Help - ?",
		Title: "Response help",
		Body: strings.Join([]string{
			"j/k or up/down scroll",
			"[ or ] or h/l switch responses",
			"R or Ctrl+L reload spec",
			"d open spec diff",
			"t / T switch theme",
			"1-4 or Tab / Shift+Tab focus panes",
			"z zoom focused pane",
			"? or Esc close help",
		}, "\n"),
	}
}
