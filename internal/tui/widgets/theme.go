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
	TextSuccess      lipgloss.Color
	TextError        lipgloss.Color
	TextWarning      lipgloss.Color
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
			TextSuccess:      lipgloss.Color("#63f542"),
			TextError:        lipgloss.Color("#ff7a7a"),
			TextWarning:      lipgloss.Color("#f5bc42"),
			TextInverse:      lipgloss.Color("#0F1322"),
			Selection:        lipgloss.Color("#fcb86a"),
			SelectionText:    lipgloss.Color("#0F1322"),
			InputBackground:  lipgloss.Color("#633B00"),
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
