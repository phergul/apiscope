package specdiff

import (
	"strings"

	"github.com/phergul/apiscope/internal/tui/widgets"
)

// BuildHelpView returns the contextual help for the spec-diff popup.
func BuildHelpView() widgets.HelpView {
	return widgets.HelpView{
		Hint:  "Help - ?",
		Title: "Spec diff help",
		Body: strings.Join([]string{
			"d open or close spec diff",
			"Esc or q close spec diff",
			"R or Ctrl+L reload spec",
			"t / T switch theme",
			"? or Esc close help",
		}, "\n"),
	}
}
