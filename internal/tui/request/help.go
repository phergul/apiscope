package request

import (
	"strings"

	"github.com/phergul/apiscope/internal/tui/widgets"
)

// BuildBrowseHelpView returns the contextual help for request browsing.
func BuildBrowseHelpView() widgets.HelpView {
	return widgets.HelpView{
		Hint:  "Help - ?",
		Title: "Request help",
		Body: strings.Join([]string{
			"j/k or up/down move row",
			"Home/End jump to first or last row",
			"[ or ] or h/l switch sections",
			"Enter edit, apply, or cycle option",
			"Ctrl+R send request",
			"R or Ctrl+L reload spec",
			"d open spec diff",
			"c export curl",
			"t / T switch theme",
			"1-4 or Tab / Shift+Tab focus panes",
			"z zoom focused pane",
			"? or Esc close help",
		}, "\n"),
	}
}

// BuildEditHelpView builds the contextual help for the active request editor.
func BuildEditHelpView(state EditorState) widgets.HelpView {
	if strings.TrimSpace(state.Kind) == "" {
		return widgets.HelpView{}
	}

	return widgets.HelpView{
		Hint:  "Help - ?",
		Title: "Help",
		Body:  editHelpBody(state.Kind),
	}
}
