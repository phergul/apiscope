package widgets

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func RenderHTTPMethod(method string, width int) string {
	if width < 1 {
		width = len(method)
	}

	return lipgloss.NewStyle().
		Width(width).
		Foreground(MethodColor(method)).
		Bold(true).
		Render(strings.ToUpper(method))
}
