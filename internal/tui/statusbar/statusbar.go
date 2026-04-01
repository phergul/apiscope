package statusbar

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type Data struct {
	Status   string
	HelpHint string
}

func Render(data Data, width int) string {
	left := strings.TrimSpace(data.Status)
	right := strings.TrimSpace(data.HelpHint)
	if right == "" {
		right = "Help - ?"
	}

	if width <= 0 {
		if left == "" {
			return right
		}
		return left + " " + right
	}

	if lipgloss.Width(right) >= width {
		return lipgloss.NewStyle().MaxWidth(width).Render(right)
	}

	leftWidth := max(width - lipgloss.Width(right) - 1, 0)
	if left != "" && lipgloss.Width(left) > leftWidth {
		left = lipgloss.NewStyle().MaxWidth(leftWidth).Render(left)
	}

	if left == "" {
		return strings.Repeat(" ", width-lipgloss.Width(right)) + right
	}

	gapWidth := max(width - lipgloss.Width(left) - lipgloss.Width(right), 1)

	return left + strings.Repeat(" ", gapWidth) + right
}
