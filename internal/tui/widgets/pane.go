package widgets

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func RenderPaneFrame(titleLeft, title, titleRight, content string, width int, focused bool) string {
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
	leftText := ""
	leftTextWidth := 0
	if strings.TrimSpace(titleLeft) != "" {
		leftText = " " + titleLeft + " "
		leftTextWidth = lipgloss.Width(leftText)
	}
	titleText := " " + title + " "
	titleWidth := lipgloss.Width(titleText)
	if titleWidth > innerBorderWidth {
		titleText = lipgloss.NewStyle().MaxWidth(innerBorderWidth).Render(titleText)
		titleWidth = lipgloss.Width(titleText)
	}
	rightText := ""
	rightWidth := 0
	if strings.TrimSpace(titleRight) != "" {
		rightText = " " + titleRight + " "
		rightWidth = lipgloss.Width(rightText)
		if rightWidth > max(innerBorderWidth-titleWidth-leftTextWidth, 0) {
			available := max(innerBorderWidth-titleWidth-leftTextWidth-1, 0)
			rightText = lipgloss.NewStyle().MaxWidth(available).Render(rightText)
			rightWidth = lipgloss.Width(rightText)
		}
	}
	if leftTextWidth > max(innerBorderWidth-titleWidth-rightWidth, 0) {
		available := max(innerBorderWidth-titleWidth-rightWidth-1, 0)
		leftText = lipgloss.NewStyle().MaxWidth(available).Render(leftText)
		leftTextWidth = lipgloss.Width(leftText)
	}

	leftGapWidth := max((innerBorderWidth-titleWidth-rightWidth-leftTextWidth)/2, 0)
	middleWidth := max(innerBorderWidth-titleWidth-rightWidth-leftTextWidth-leftGapWidth, 0)
	topLine := borderStyle.Render(paneBorder.TopLeft) +
		MutedTextStyle().Render(leftText) +
		borderStyle.Render(strings.Repeat(paneBorder.Top, leftGapWidth)) +
		titleStyle.Render(titleText) +
		borderStyle.Render(strings.Repeat(paneBorder.Top, middleWidth)) +
		MutedTextStyle().Render(rightText) +
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
