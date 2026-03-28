package widgets

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	HttpMethodGet     = "#4287f5"
	HttpMethodPost    = "#63f542"
	HttpMethodPut     = "#f5bc42"
	HttpMethodPatch   = "#f542f5"
	HttpMethodDelete  = "#d61806"
	HttpMethodHead    = "#1b1fe3"
	HttpMethodOptions = "#1be3d9"
	HttpMethodDefault = "#aab2c8"
)

type Theme struct {
	Palette Palette
}

type Palette struct {
	Border           lipgloss.Color
	BorderFocused    lipgloss.Color
	Text             lipgloss.Color
	TextMuted        lipgloss.Color
	TextInverse      lipgloss.Color
	Selection        lipgloss.Color
	SelectionText    lipgloss.Color
	InputBackground  lipgloss.Color
	InputBorder      lipgloss.Color
	InputText        lipgloss.Color
	InputPlaceholder lipgloss.Color
	MethodGet        lipgloss.Color
	MethodPost       lipgloss.Color
	MethodPut        lipgloss.Color
	MethodPatch      lipgloss.Color
	MethodDelete     lipgloss.Color
	MethodHead       lipgloss.Color
	MethodOptions    lipgloss.Color
	MethodDefault    lipgloss.Color
}

var (
	paneBorder  = lipgloss.NormalBorder()
	activeTheme = Theme{
		Palette: Palette{
			Border:           lipgloss.Color("#48506B"),
			BorderFocused:    lipgloss.Color("#fcb86a"),
			Text:             lipgloss.Color("#E8ECF5"),
			TextMuted:        lipgloss.Color("#98A3C2"),
			TextInverse:      lipgloss.Color("#0F1322"),
			Selection:        lipgloss.Color("#a9c8d4"),
			SelectionText:    lipgloss.Color("#0F1322"),
			InputBackground:  lipgloss.Color("#232A42"),
			InputBorder:      lipgloss.Color("#7CC7FF"),
			InputText:        lipgloss.Color("#F5F7FB"),
			InputPlaceholder: lipgloss.Color("#8089A8"),
			MethodGet:        lipgloss.Color(HttpMethodGet),
			MethodPost:       lipgloss.Color(HttpMethodPost),
			MethodPut:        lipgloss.Color(HttpMethodPut),
			MethodPatch:      lipgloss.Color(HttpMethodPatch),
			MethodDelete:     lipgloss.Color(HttpMethodDelete),
			MethodHead:       lipgloss.Color(HttpMethodHead),
			MethodOptions:    lipgloss.Color(HttpMethodOptions),
			MethodDefault:    lipgloss.Color(HttpMethodDefault),
		},
	}
)

func CurrentTheme() Theme {
	return activeTheme
}

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

func RenderMutedHeading(content string) string {
	return MutedTextStyle().Bold(true).Render(content)
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
		Foreground(theme.Palette.TextInverse).
		Background(theme.Palette.Selection)
}

func InputFrameStyle(focused bool) lipgloss.Style {
	theme := CurrentTheme()
	borderColor := theme.Palette.Border
	if focused {
		borderColor = theme.Palette.InputBorder
	}
	return InputTextStyle().
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

func PaneFrameStyle(focused bool) lipgloss.Style {
	theme := CurrentTheme()
	borderColor := theme.Palette.Border
	if focused {
		borderColor = theme.Palette.BorderFocused
	}
	return BodyTextStyle().
		Border(paneBorder).
		BorderForeground(borderColor).
		Padding(0, 1).
		Bold(focused)
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

func MethodColor(method string) lipgloss.Color {
	return activeTheme.methodColor(method)
}

func (t Theme) methodColor(method string) lipgloss.Color {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case "GET":
		return t.Palette.MethodGet
	case "POST":
		return t.Palette.MethodPost
	case "PUT":
		return t.Palette.MethodPut
	case "PATCH":
		return t.Palette.MethodPatch
	case "DELETE":
		return t.Palette.MethodDelete
	case "HEAD":
		return t.Palette.MethodHead
	case "OPTIONS":
		return t.Palette.MethodOptions
	default:
		return t.Palette.MethodDefault
	}
}
