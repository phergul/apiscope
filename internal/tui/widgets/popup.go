package widgets

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type PopupData struct {
	Title       string
	Meta        string
	Body        string
	Help        string
	HelpVisible bool
	Width       int
	Focused     bool
}

func RenderPopup(data PopupData) string {
	width := max(data.Width, 18)
	innerWidth := max(width-4, 1)

	lines := make([]string, 0, 6)
	header := renderPopupHeader(data.Title, data.Meta, innerWidth)
	if header != "" {
		lines = append(lines, header, "")
	}

	body := strings.TrimRight(data.Body, "\n")
	if body != "" {
		for line := range strings.SplitSeq(body, "\n") {
			lines = append(lines, fitPopupLine(line, innerWidth))
		}
	}

	if data.HelpVisible && strings.TrimSpace(data.Help) != "" {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		for line := range strings.SplitSeq(data.Help, "\n") {
			lines = append(lines, fitPopupLine(MutedTextStyle().Render(line), innerWidth))
		}
	}

	if len(lines) == 0 {
		lines = append(lines, "")
	}

	return renderPopupFrame(lines, width, data.Focused)
}

func Overlay(base, popup string, x, y int) string {
	baseLines := strings.Split(base, "\n")
	popupLines := strings.Split(popup, "\n")
	if len(baseLines) == 0 {
		baseLines = []string{""}
	}
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	for len(baseLines) < y+len(popupLines) {
		baseLines = append(baseLines, "")
	}

	for index, line := range popupLines {
		baseLine := baseLines[y+index]
		baseWidth := ansi.StringWidth(baseLine)
		if baseWidth < x {
			baseLine += strings.Repeat(" ", x-baseWidth)
		}

		left := ansi.Cut(baseLine, 0, x)
		popupWidth := ansi.StringWidth(line)
		right := ""
		baseWidth = ansi.StringWidth(baseLine)
		if baseWidth > x+popupWidth {
			right = ansi.Cut(baseLine, x+popupWidth, baseWidth)
		}

		baseLines[y+index] = left + line + right
	}

	return strings.Join(baseLines, "\n")
}

func renderPopupHeader(title, meta string, width int) string {
	title = strings.TrimSpace(title)
	meta = strings.TrimSpace(meta)
	if title == "" && meta == "" {
		return ""
	}

	left := BodyTextStyle().Bold(true).Render(title)
	right := MutedTextStyle().Render(meta)
	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	if leftWidth+rightWidth > width {
		if title != "" {
			left = BodyTextStyle().Bold(true).Render(lipgloss.NewStyle().MaxWidth(max(width-rightWidth-1, 0)).Render(title))
			leftWidth = lipgloss.Width(left)
		}
	}

	gapWidth := max(width-leftWidth-rightWidth, 0)
	return left + strings.Repeat(" ", gapWidth) + right
}

func fitPopupLine(line string, width int) string {
	if width <= 0 {
		return line
	}

	return lipgloss.NewStyle().Width(width).MaxWidth(width).Render(line)
}

func renderPopupFrame(lines []string, width int, focused bool) string {
	theme := CurrentTheme()
	borderColor := theme.Palette.Border
	if focused {
		borderColor = theme.Palette.BorderFocused
	}

	borderStyle := lipgloss.NewStyle().Foreground(borderColor)
	innerWidth := max(width-2, 1)
	contentWidth := max(width-4, 1)

	framed := make([]string, 0, len(lines)+2)
	framed = append(framed, borderStyle.Render(paneBorder.TopLeft+strings.Repeat(paneBorder.Top, innerWidth)+paneBorder.TopRight))
	for _, line := range lines {
		padded := lipgloss.NewStyle().Width(contentWidth).MaxWidth(contentWidth).Render(line)
		framed = append(framed,
			borderStyle.Render(paneBorder.Left)+
				" "+padded+" "+
				borderStyle.Render(paneBorder.Right),
		)
	}
	framed = append(framed, borderStyle.Render(paneBorder.BottomLeft+strings.Repeat(paneBorder.Bottom, innerWidth)+paneBorder.BottomRight))

	return strings.Join(framed, "\n")
}
