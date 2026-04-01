package widgets

import (
	"slices"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	defaultThemeName   = "default"
	alternateThemeName = "harbor"
)

type Theme struct {
	Name    string
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
	paneBorder = lipgloss.NormalBorder()

	themeNames = []string{
		defaultThemeName,
		alternateThemeName,
	}

	themeRegistry = map[string]Theme{
		defaultThemeName: {
			Name: defaultThemeName,
			Palette: Palette{
				Border:           color("#48506B"),
				BorderFocused:    color("#fcb86a"),
				Text:             color("#E8ECF5"),
				TextMuted:        color("#98A3C2"),
				TextSuccess:      color("#63f542"),
				TextError:        color("#ff7a7a"),
				TextWarning:      color("#f5bc42"),
				TextInverse:      color("#0F1322"),
				Selection:        color("#fcb86a"),
				SelectionText:    color("#0F1322"),
				InputBackground:  color("#633B00"),
				InputBorder:      color("#7CC7FF"),
				InputText:        color("#F5F7FB"),
				InputPlaceholder: color("#8089A8"),
				MethodGet:        color("#4287f5"),
				MethodPost:       color("#63f542"),
				MethodPut:        color("#f5bc42"),
				MethodPatch:      color("#f542f5"),
				MethodDelete:     color("#d61806"),
				MethodHead:       color("#1b1fe3"),
				MethodOptions:    color("#1be3d9"),
				MethodDefault:    color("#aab2c8"),
			},
		},
		alternateThemeName: {
			Name: alternateThemeName,
			Palette: Palette{
				Border:           color("#4D4E5C"),
				BorderFocused:    color("#8FE0C7"),
				Text:             color("#F2F2EE"),
				TextMuted:        color("#B9BAAF"),
				TextSuccess:      color("#9CE06D"),
				TextError:        color("#F18A7A"),
				TextWarning:      color("#E6C15A"),
				TextInverse:      color("#14211F"),
				Selection:        color("#8FE0C7"),
				SelectionText:    color("#14211F"),
				InputBackground:  color("#3E4A2A"),
				InputBorder:      color("#A7D8FF"),
				InputText:        color("#F6F8F2"),
				InputPlaceholder: color("#A5AE9D"),
				MethodGet:        color("#73B7FF"),
				MethodPost:       color("#89D98A"),
				MethodPut:        color("#E9C86C"),
				MethodPatch:      color("#D89BFF"),
				MethodDelete:     color("#F17A6B"),
				MethodHead:       color("#6F90FF"),
				MethodOptions:    color("#79E0D7"),
				MethodDefault:    color("#C0C7D6"),
			},
		},
	}

	activeTheme = defaultTheme()
)

func DefaultTheme() Theme {
	return defaultTheme()
}

func CurrentTheme() Theme {
	return activeTheme
}

func AvailableThemes() []string {
	return append([]string(nil), themeNames...)
}

func ThemeByName(name string) (Theme, bool) {
	normalized := normalizeThemeName(name)
	theme, ok := themeRegistry[normalized]
	if !ok {
		return defaultTheme(), false
	}

	return theme, true
}

func SetTheme(theme Theme) {
	if normalized, ok := ThemeByName(theme.Name); ok {
		activeTheme = normalized
		return
	}

	activeTheme = defaultTheme()
}

func SetThemeByName(name string) bool {
	theme, ok := ThemeByName(name)
	activeTheme = theme
	return ok
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

func defaultTheme() Theme {
	return themeRegistry[defaultThemeName]
}

func normalizeThemeName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return defaultThemeName
	}
	if slices.Contains(themeNames, name) {
		return name
	}

	return name
}

func color(value string) lipgloss.Color {
	return lipgloss.Color(value)
}
