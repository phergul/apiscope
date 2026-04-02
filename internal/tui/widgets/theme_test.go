package widgets

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestDefaultThemeMatchesCurrentThemeBaseline(t *testing.T) {
	t.Parallel()

	theme := DefaultTheme()
	if theme.Name != defaultThemeName {
		t.Fatalf("expected default theme name %q, got %q", defaultThemeName, theme.Name)
	}
	if theme.Palette.Border != color("#4D4E5C") {
		t.Fatalf("expected default border color to stay stable, got %q", theme.Palette.Border)
	}
	if theme.Palette.InputBackground != color("#3E4A2A") {
		t.Fatalf("expected default input background to stay stable, got %q", theme.Palette.InputBackground)
	}
}

func TestThemeByNameReturnsBuiltInTheme(t *testing.T) {
	t.Parallel()

	theme, ok := ThemeByName(harborThemeName)
	if !ok {
		t.Fatal("expected built-in theme lookup to succeed")
	}
	if theme.Name != harborThemeName {
		t.Fatalf("expected built-in theme name %q, got %q", harborThemeName, theme.Name)
	}
	if theme.Palette.Border == DefaultTheme().Palette.Border {
		t.Fatal("expected built-in theme border color to differ from default")
	}
}

func TestThemeByNameFallsBackToDefaultForUnknownTheme(t *testing.T) {
	t.Parallel()

	theme, ok := ThemeByName("missing")
	if ok {
		t.Fatal("expected missing theme lookup to report false")
	}
	if theme.Name != defaultThemeName {
		t.Fatalf("expected missing theme lookup to fall back to default, got %q", theme.Name)
	}
}

func TestSetThemeByNameChangesCurrentThemeAndFallsBack(t *testing.T) {
	original := CurrentTheme()
	t.Cleanup(func() {
		SetTheme(original)
	})

	if ok := SetThemeByName(emberThemeName); !ok {
		t.Fatal("expected built-in theme selection to succeed")
	}
	if CurrentTheme().Name != emberThemeName {
		t.Fatalf("expected current theme %q, got %q", emberThemeName, CurrentTheme().Name)
	}

	if ok := SetThemeByName("missing"); ok {
		t.Fatal("expected missing theme selection to report false")
	}
	if CurrentTheme().Name != defaultThemeName {
		t.Fatalf("expected current theme to fall back to default, got %q", CurrentTheme().Name)
	}
}

func TestAvailableThemesIncludesBuiltIns(t *testing.T) {
	t.Parallel()

	names := AvailableThemes()
	if len(names) != 4 {
		t.Fatalf("expected 4 available themes, got %#v", names)
	}
	if names[0] != defaultThemeName || names[1] != harborThemeName || names[2] != emberThemeName || names[3] != tundraThemeName {
		t.Fatalf("expected stable theme names, got %#v", names)
	}
}

func TestNextThemeNameWrapsThroughStableThemeOrder(t *testing.T) {
	t.Parallel()

	if got := NextThemeName(defaultThemeName); got != harborThemeName {
		t.Fatalf("expected next theme after %q to be %q, got %q", defaultThemeName, harborThemeName, got)
	}
	if got := NextThemeName(tundraThemeName); got != defaultThemeName {
		t.Fatalf("expected next theme after %q to wrap to %q, got %q", tundraThemeName, defaultThemeName, got)
	}
}

func TestPreviousThemeNameWrapsThroughStableThemeOrder(t *testing.T) {
	t.Parallel()

	if got := PreviousThemeName(harborThemeName); got != defaultThemeName {
		t.Fatalf("expected previous theme before %q to be %q, got %q", harborThemeName, defaultThemeName, got)
	}
	if got := PreviousThemeName(defaultThemeName); got != tundraThemeName {
		t.Fatalf("expected previous theme before %q to wrap to %q, got %q", defaultThemeName, tundraThemeName, got)
	}
}

func TestMethodColorRespectsActiveTheme(t *testing.T) {
	original := CurrentTheme()
	t.Cleanup(func() {
		SetTheme(original)
	})

	defaultGet := DefaultTheme().Palette.MethodGet
	if ok := SetThemeByName(tundraThemeName); !ok {
		t.Fatal("expected built-in theme selection to succeed")
	}
	if MethodColor("GET") == defaultGet {
		t.Fatal("expected GET method color to change with active theme")
	}
}

func TestMethodSelectedColorRespectsActiveTheme(t *testing.T) {
	original := CurrentTheme()
	t.Cleanup(func() {
		SetTheme(original)
	})

	defaultSelectedGet := DefaultTheme().Palette.MethodGetSelected
	if ok := SetThemeByName(tundraThemeName); !ok {
		t.Fatal("expected built-in theme selection to succeed")
	}
	if MethodSelectedColor("GET") == defaultSelectedGet {
		t.Fatal("expected selected GET method color to change with active theme")
	}
}

func TestBuiltInThemesKeepSelectionDistinctFromMethodColors(t *testing.T) {
	t.Parallel()

	for _, name := range AvailableThemes() {
		theme, ok := ThemeByName(name)
		if !ok {
			t.Fatalf("expected theme lookup for %q to succeed", name)
		}

		methodColors := []lipgloss.Color{
			theme.Palette.MethodGet,
			theme.Palette.MethodPost,
			theme.Palette.MethodPut,
			theme.Palette.MethodPatch,
			theme.Palette.MethodDelete,
			theme.Palette.MethodHead,
			theme.Palette.MethodOptions,
			theme.Palette.MethodDefault,
		}
		for _, methodColor := range methodColors {
			if theme.Palette.Selection == methodColor {
				t.Fatalf("expected selection color in theme %q to differ from method color %q", name, methodColor)
			}
		}
	}
}

func TestBuiltInThemesKeepSelectedMethodColorsDistinctFromSelectionText(t *testing.T) {
	t.Parallel()

	for _, name := range AvailableThemes() {
		theme, ok := ThemeByName(name)
		if !ok {
			t.Fatalf("expected theme lookup for %q to succeed", name)
		}

		selectedMethodColors := []lipgloss.Color{
			theme.Palette.MethodGetSelected,
			theme.Palette.MethodPostSelected,
			theme.Palette.MethodPutSelected,
			theme.Palette.MethodPatchSelected,
			theme.Palette.MethodDeleteSelected,
			theme.Palette.MethodHeadSelected,
			theme.Palette.MethodOptionsSelected,
			theme.Palette.MethodDefaultSelected,
		}
		for _, methodColor := range selectedMethodColors {
			if theme.Palette.Selection == methodColor {
				t.Fatalf("expected selected method color in theme %q to differ from selection background %q", name, methodColor)
			}
			if theme.Palette.SelectionText == methodColor {
				t.Fatalf("expected selected method color in theme %q to differ from selection text %q", name, methodColor)
			}
		}
	}
}
