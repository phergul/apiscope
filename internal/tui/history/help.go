package history

import (
	"strings"

	"github.com/phergul/apiscope/internal/tui/widgets"
)

// BuildHelpView returns the contextual help for the previous-requests popup.
func BuildHelpView() widgets.HelpView {
	return widgets.HelpView{
		Hint:  "Help - ?",
		Title: "History help",
		Body: strings.Join([]string{
			"j/k or up/down move selection",
			"Ctrl+U / Ctrl+D scroll preview",
			"t / T switch theme",
			"Enter load response",
			"r restore request",
			"Esc or p close history",
			"? or Esc close help",
		}, "\n"),
	}
}
