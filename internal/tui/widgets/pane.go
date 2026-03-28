package widgets

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func RenderPaneFrame(title, content string, width int, focused bool) string {
	theme := CurrentTheme()
	borderColor := theme.Palette.Border
	if focused {
		borderColor = theme.Palette.BorderFocused
	}

	borderStyle := lipgloss.NewStyle().Foreground(borderColor)
	titleStyle := BodyTextStyle()
	if focused {
		titleStyle = titleStyle.Bold(true)
	}

	width = max(width, 3)
	innerBorderWidth := max(width-2, 1)
	titleText := " " + title + " "
	titleWidth := lipgloss.Width(titleText)
	if titleWidth > innerBorderWidth {
		titleText = lipgloss.NewStyle().MaxWidth(innerBorderWidth).Render(titleText)
		titleWidth = lipgloss.Width(titleText)
	}

	leftWidth := max((innerBorderWidth-titleWidth)/2, 0)
	rightWidth := max(innerBorderWidth-titleWidth-leftWidth, 0)
	topLine := borderStyle.Render(paneBorder.TopLeft) +
		borderStyle.Render(strings.Repeat(paneBorder.Top, leftWidth)) +
		titleStyle.Render(titleText) +
		borderStyle.Render(strings.Repeat(paneBorder.Top, rightWidth)) +
		borderStyle.Render(paneBorder.TopRight)

	bodyWidth := max(width-4, 1)
	bodyLines := strings.Split(content, "\n")
	framed := make([]string, 0, len(bodyLines)+2)
	framed = append(framed, topLine)
	for _, line := range bodyLines {
		padded := lipgloss.NewStyle().Width(bodyWidth).MaxWidth(bodyWidth).Render(line)
		framed = append(framed,
			borderStyle.Render(paneBorder.Left)+
				" "+padded+" "+
				borderStyle.Render(paneBorder.Right),
		)
	}
	bottomLine := borderStyle.Render(paneBorder.BottomLeft + strings.Repeat(paneBorder.Bottom, innerBorderWidth) + paneBorder.BottomRight)
	framed = append(framed, bottomLine)

	return strings.Join(framed, "\n")
}
