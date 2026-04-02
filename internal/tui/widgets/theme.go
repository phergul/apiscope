package widgets

import (
	"slices"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	defaultThemeName = "default"
	harborThemeName  = "harbor"
	emberThemeName   = "ember"
	tundraThemeName  = "tundra"
)

type Theme struct {
	Name    string
	Palette Palette
}

type Palette struct {
	Border                lipgloss.Color
	BorderFocused         lipgloss.Color
	Text                  lipgloss.Color
	TextMuted             lipgloss.Color
	TextSuccess           lipgloss.Color
	TextError             lipgloss.Color
	TextWarning           lipgloss.Color
	TextInverse           lipgloss.Color
	Selection             lipgloss.Color
	SelectionText         lipgloss.Color
	InputBackground       lipgloss.Color
	InputBorder           lipgloss.Color
	InputText             lipgloss.Color
	InputPlaceholder      lipgloss.Color
	MethodGet             lipgloss.Color
	MethodPost            lipgloss.Color
	MethodPut             lipgloss.Color
	MethodPatch           lipgloss.Color
	MethodDelete          lipgloss.Color
	MethodHead            lipgloss.Color
	MethodOptions         lipgloss.Color
	MethodDefault         lipgloss.Color
	MethodGetSelected     lipgloss.Color
	MethodPostSelected    lipgloss.Color
	MethodPutSelected     lipgloss.Color
	MethodPatchSelected   lipgloss.Color
	MethodDeleteSelected  lipgloss.Color
	MethodHeadSelected    lipgloss.Color
	MethodOptionsSelected lipgloss.Color
	MethodDefaultSelected lipgloss.Color
}

var (
	paneBorder = lipgloss.NormalBorder()

	themeNames = []string{
		defaultThemeName,
		harborThemeName,
		emberThemeName,
		tundraThemeName,
	}

	themeRegistry = map[string]Theme{
		defaultThemeName: {
			Name: defaultThemeName,
			Palette: Palette{
				Border:                color("#4D4E5C"),
				BorderFocused:         color("#8FE0C7"),
				Text:                  color("#F2F2EE"),
				TextMuted:             color("#B9BAAF"),
				TextSuccess:           color("#9CE06D"),
				TextError:             color("#F18A7A"),
				TextWarning:           color("#E6C15A"),
				TextInverse:           color("#14211F"),
				Selection:             color("#8FE0C7"),
				SelectionText:         color("#14211F"),
				InputBackground:       color("#3E4A2A"),
				InputBorder:           color("#A7D8FF"),
				InputText:             color("#F6F8F2"),
				InputPlaceholder:      color("#A5AE9D"),
				MethodGet:             color("#73B7FF"),
				MethodPost:            color("#89D98A"),
				MethodPut:             color("#E9C86C"),
				MethodPatch:           color("#D89BFF"),
				MethodDelete:          color("#F17A6B"),
				MethodHead:            color("#6F90FF"),
				MethodOptions:         color("#79E0D7"),
				MethodDefault:         color("#C0C7D6"),
				MethodGetSelected:     color("#1F5AA8"),
				MethodPostSelected:    color("#1F7A43"),
				MethodPutSelected:     color("#8A5C00"),
				MethodPatchSelected:   color("#7A38A3"),
				MethodDeleteSelected:  color("#A12B2B"),
				MethodHeadSelected:    color("#3049B8"),
				MethodOptionsSelected: color("#0B7C72"),
				MethodDefaultSelected: color("#38404F"),
			},
		},
		harborThemeName: {
			Name: harborThemeName,
			Palette: Palette{
				Border:                color("#45536F"),
				BorderFocused:         color("#F4A261"),
				Text:                  color("#EFF3FB"),
				TextMuted:             color("#A2B0C9"),
				TextSuccess:           color("#8AE7A1"),
				TextError:             color("#F28B82"),
				TextWarning:           color("#F2C46D"),
				TextInverse:           color("#151C2E"),
				Selection:             color("#F4A261"),
				SelectionText:         color("#151C2E"),
				InputBackground:       color("#5E3816"),
				InputBorder:           color("#7AC7FF"),
				InputText:             color("#F9FBFF"),
				InputPlaceholder:      color("#A7B4CB"),
				MethodGet:             color("#65B9FF"),
				MethodPost:            color("#63DA95"),
				MethodPut:             color("#E9D46A"),
				MethodPatch:           color("#D8A8FF"),
				MethodDelete:          color("#F07178"),
				MethodHead:            color("#7B9CFF"),
				MethodOptions:         color("#66E3D4"),
				MethodDefault:         color("#C8D0E0"),
				MethodGetSelected:     color("#123C74"),
				MethodPostSelected:    color("#165C37"),
				MethodPutSelected:     color("#7B5A12"),
				MethodPatchSelected:   color("#6A3D8F"),
				MethodDeleteSelected:  color("#8E2433"),
				MethodHeadSelected:    color("#3048A8"),
				MethodOptionsSelected: color("#146F68"),
				MethodDefaultSelected: color("#394556"),
			},
		},
		emberThemeName: {
			Name: emberThemeName,
			Palette: Palette{
				Border:                color("#5B4650"),
				BorderFocused:         color("#E56B6F"),
				Text:                  color("#FBF4F1"),
				TextMuted:             color("#C5B6B2"),
				TextSuccess:           color("#8ED082"),
				TextError:             color("#F28B82"),
				TextWarning:           color("#F2B96D"),
				TextInverse:           color("#23181D"),
				Selection:             color("#E56B6F"),
				SelectionText:         color("#23181D"),
				InputBackground:       color("#5A2E27"),
				InputBorder:           color("#FFC8A2"),
				InputText:             color("#FFF8F4"),
				InputPlaceholder:      color("#C9B4AD"),
				MethodGet:             color("#6AAFFF"),
				MethodPost:            color("#77D78A"),
				MethodPut:             color("#E9C46A"),
				MethodPatch:           color("#C792EA"),
				MethodDelete:          color("#FF7F6A"),
				MethodHead:            color("#78A6FF"),
				MethodOptions:         color("#59D9C5"),
				MethodDefault:         color("#D9C9C2"),
				MethodGetSelected:     color("#163F78"),
				MethodPostSelected:    color("#215C2F"),
				MethodPutSelected:     color("#7A590F"),
				MethodPatchSelected:   color("#6B418A"),
				MethodDeleteSelected:  color("#972D1F"),
				MethodHeadSelected:    color("#3651AE"),
				MethodOptionsSelected: color("#116C62"),
				MethodDefaultSelected: color("#4A3A3A"),
			},
		},
		tundraThemeName: {
			Name: tundraThemeName,
			Palette: Palette{
				Border:                color("#4E5962"),
				BorderFocused:         color("#DDBB5A"),
				Text:                  color("#F3F6F4"),
				TextMuted:             color("#B0BAB6"),
				TextSuccess:           color("#8BD49C"),
				TextError:             color("#EF8C7D"),
				TextWarning:           color("#DDBB5A"),
				TextInverse:           color("#1C2421"),
				Selection:             color("#DDBB5A"),
				SelectionText:         color("#1C2421"),
				InputBackground:       color("#4E4631"),
				InputBorder:           color("#A7D7FF"),
				InputText:             color("#FBFDFB"),
				InputPlaceholder:      color("#AAB5AF"),
				MethodGet:             color("#6DB6FF"),
				MethodPost:            color("#73D39B"),
				MethodPut:             color("#FF9F5A"),
				MethodPatch:           color("#C39BFF"),
				MethodDelete:          color("#F56F6F"),
				MethodHead:            color("#7B98FF"),
				MethodOptions:         color("#62D8E3"),
				MethodDefault:         color("#CAD3CF"),
				MethodGetSelected:     color("#164678"),
				MethodPostSelected:    color("#1E5F3B"),
				MethodPutSelected:     color("#8A4F0E"),
				MethodPatchSelected:   color("#69418A"),
				MethodDeleteSelected:  color("#9B2E2E"),
				MethodHeadSelected:    color("#334DAE"),
				MethodOptionsSelected: color("#116A73"),
				MethodDefaultSelected: color("#3E4A46"),
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

// NextThemeName returns the next built-in theme name, wrapping to the start.
func NextThemeName(current string) string {
	index := themeIndex(current)
	return themeNames[(index+1)%len(themeNames)]
}

// PreviousThemeName returns the previous built-in theme name, wrapping to the end.
func PreviousThemeName(current string) string {
	index := themeIndex(current)
	return themeNames[(index-1+len(themeNames))%len(themeNames)]
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

// MethodSelectedColor returns the selected-row method color for the active theme.
func MethodSelectedColor(method string) lipgloss.Color {
	return activeTheme.methodSelectedColor(method)
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

func (t Theme) methodSelectedColor(method string) lipgloss.Color {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case "GET":
		return t.Palette.MethodGetSelected
	case "POST":
		return t.Palette.MethodPostSelected
	case "PUT":
		return t.Palette.MethodPutSelected
	case "PATCH":
		return t.Palette.MethodPatchSelected
	case "DELETE":
		return t.Palette.MethodDeleteSelected
	case "HEAD":
		return t.Palette.MethodHeadSelected
	case "OPTIONS":
		return t.Palette.MethodOptionsSelected
	default:
		return t.Palette.MethodDefaultSelected
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

func themeIndex(name string) int {
	normalized := normalizeThemeName(name)
	index := slices.Index(themeNames, normalized)
	if index < 0 {
		return 0
	}

	return index
}

func color(value string) lipgloss.Color {
	return lipgloss.Color(value)
}
