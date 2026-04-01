package widgets

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func BodyTextStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(CurrentTheme().Palette.Text)
}

func MutedTextStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(CurrentTheme().Palette.TextMuted)
}

func SelectedTextStyle() lipgloss.Style {
	theme := CurrentTheme()
	return lipgloss.NewStyle().
		Foreground(theme.Palette.SelectionText).
		Background(theme.Palette.Selection)
}

func SuccessTextStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(CurrentTheme().Palette.TextSuccess)
}

func ErrorTextStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(CurrentTheme().Palette.TextError)
}

func WarningTextStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(CurrentTheme().Palette.TextWarning)
}

func RenderMutedHeading(content string) string {
	return MutedTextStyle().Bold(true).Render(content)
}

func RenderValidationMessage(content string) string {
	return WarningTextStyle().Bold(true).Render(content)
}

func InputTextStyle() lipgloss.Style {
	theme := CurrentTheme()
	return lipgloss.NewStyle().
		Foreground(theme.Palette.InputText).
		Background(theme.Palette.InputBackground)
}

func InputPlaceholderStyle() lipgloss.Style {
	theme := CurrentTheme()
	return lipgloss.NewStyle().
		Foreground(theme.Palette.InputPlaceholder).
		Background(theme.Palette.InputBackground)
}

func InputCursorStyle() lipgloss.Style {
	theme := CurrentTheme()
	return lipgloss.NewStyle().
		Foreground(theme.Palette.InputText).
		Background(theme.Palette.InputBackground)
}

// InputAreaStyle returns the filled background style for an input area of the requested size.
func InputAreaStyle(width, height int) lipgloss.Style {
	theme := CurrentTheme()
	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		MaxWidth(width).
		MaxHeight(height).
		Background(theme.Palette.InputBackground)
}

// RenderFilledInputArea renders a full-size input background block and overlays content onto it.
func RenderFilledInputArea(content string, width, height int) string {
	baseLine := InputAreaStyle(width, 1).Render("")
	var base strings.Builder
	for row := 0; row < max(height, 1); row++ {
		if row > 0 {
			base.WriteString("\n")
		}
		base.WriteString(baseLine)
	}

	return Overlay(base.String(), content, 0, 0)
}

func InputFrameStyle(focused bool) lipgloss.Style {
	theme := CurrentTheme()
	borderColor := theme.Palette.Border
	if focused {
		borderColor = theme.Palette.InputBorder
	}
	return lipgloss.NewStyle().
		Foreground(theme.Palette.InputText).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1)
}

func PaneBodyStyle(width, height int) lipgloss.Style {
	return BodyTextStyle().
		Width(width).
		Height(height).
		MaxWidth(width).
		MaxHeight(height)
}

func PaneFooterStyle(width int) lipgloss.Style {
	theme := CurrentTheme()
	return BodyTextStyle().
		BorderTop(true).
		BorderStyle(paneBorder).
		BorderForeground(theme.Palette.Border).
		Width(width)
}

func StatusBarStyle(width int) lipgloss.Style {
	return BodyTextStyle().
		Width(width)
}

func ModalStyle(width int) lipgloss.Style {
	theme := CurrentTheme()
	return BodyTextStyle().
		Width(width).
		Border(paneBorder).
		BorderForeground(theme.Palette.BorderFocused).
		Padding(1, 2)
}

func ViewportStyle() lipgloss.Style {
	return BodyTextStyle()
}

func PopupFrameStyle(width int, focused bool) lipgloss.Style {
	theme := CurrentTheme()
	borderColor := theme.Palette.Border
	if focused {
		borderColor = theme.Palette.BorderFocused
	}

	return lipgloss.NewStyle().
		Width(width).
		MaxWidth(width).
		Border(lipgloss.NormalBorder()).
		BorderForeground(borderColor).
		Padding(0, 1)
}
