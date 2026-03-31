package widgets

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// RenderPaneFrame draws a pane border with optional left, centered, and right
// header titles above the content area.
//
// The center title is always anchored to the pane midpoint. Side titles are
// then clipped into the remaining space so they cannot push the center title
// away from its visual center.
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
	mutedTitleStyle := MutedTextStyle()

	width = max(width, 3)
	innerBorderWidth := max(width-2, 1)
	topLine := borderStyle.Render(paneBorder.TopLeft + strings.Repeat(paneBorder.Top, innerBorderWidth) + paneBorder.TopRight)

	titleText := fitPaneTitle(title, innerBorderWidth)
	titleWidth := lipgloss.Width(titleText)
	centerStart := 1
	centerEnd := 1
	if titleWidth > 0 {
		// Place the main title against the true pane midpoint, not in the
		// leftover space after laying out the side labels.
		centerStart = 1 + max((innerBorderWidth-titleWidth)/2, 0)
		centerEnd = centerStart + titleWidth
		topLine = overlayPaneTitle(topLine, titleStyle.Render(titleText), centerStart)
	}

	leftLimit := innerBorderWidth
	if titleWidth > 0 {
		// Reserve everything up to the centered title for the left label so it
		// can be clipped without ever overlapping the center anchor.
		leftLimit = max(centerStart-1, 0)
	}
	leftText := fitPaneTitle(titleLeft, leftLimit)
	if leftText != "" {
		topLine = overlayPaneTitle(topLine, mutedTitleStyle.Render(leftText), 1)
	}

	rightLimit := innerBorderWidth
	if titleWidth > 0 {
		// Mirror the left-side reservation on the right so action labels stay
		// pinned to the border without disturbing the centered title.
		rightLimit = max(width-1-centerEnd, 0)
	} else {
		rightLimit = max(innerBorderWidth/2, 0)
	}
	rightText := fitPaneTitle(titleRight, rightLimit)
	rightWidth := lipgloss.Width(rightText)
	if rightWidth > 0 {
		topLine = overlayPaneTitle(topLine, mutedTitleStyle.Render(rightText), width-1-rightWidth)
	}

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

// fitPaneTitle pads a title and clips it to the space available on the border.
func fitPaneTitle(title string, width int) string {
	if strings.TrimSpace(title) == "" || width <= 0 {
		return ""
	}

	return lipgloss.NewStyle().MaxWidth(width).Render(" " + title + " ")
}

// overlayPaneTitle replaces a segment of an ANSI-styled border line at the
// requested display-column offset.
//
// ANSI-aware width and slicing are required here because border and title
// styling add escape sequences that do not count toward visible width.
func overlayPaneTitle(line, segment string, start int) string {
	lineWidth := ansi.StringWidth(line)
	segmentWidth := ansi.StringWidth(segment)
	if start < 0 || segmentWidth == 0 || start >= lineWidth {
		return line
	}

	left := ansi.Cut(line, 0, start)
	rightStart := min(start+segmentWidth, lineWidth)
	right := ""
	if rightStart < lineWidth {
		right = ansi.Cut(line, rightStart, lineWidth)
	}

	return left + segment + right
}
