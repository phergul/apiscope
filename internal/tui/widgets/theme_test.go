package widgets

import "testing"

func TestDefaultThemeMatchesCurrentThemeBaseline(t *testing.T) {
	t.Parallel()

	theme := DefaultTheme()
	if theme.Name != defaultThemeName {
		t.Fatalf("expected default theme name %q, got %q", defaultThemeName, theme.Name)
	}
	if theme.Palette.Border != color("#48506B") {
		t.Fatalf("expected default border color to stay stable, got %q", theme.Palette.Border)
	}
	if theme.Palette.InputBackground != color("#633B00") {
		t.Fatalf("expected default input background to stay stable, got %q", theme.Palette.InputBackground)
	}
}

func TestThemeByNameReturnsAlternateTheme(t *testing.T) {
	t.Parallel()

	theme, ok := ThemeByName(alternateThemeName)
	if !ok {
		t.Fatal("expected alternate theme lookup to succeed")
	}
	if theme.Name != alternateThemeName {
		t.Fatalf("expected alternate theme name %q, got %q", alternateThemeName, theme.Name)
	}
	if theme.Palette.Border == DefaultTheme().Palette.Border {
		t.Fatal("expected alternate theme border color to differ from default")
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

	if ok := SetThemeByName(alternateThemeName); !ok {
		t.Fatal("expected alternate theme selection to succeed")
	}
	if CurrentTheme().Name != alternateThemeName {
		t.Fatalf("expected current theme %q, got %q", alternateThemeName, CurrentTheme().Name)
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
	if len(names) != 2 {
		t.Fatalf("expected 2 available themes, got %#v", names)
	}
	if names[0] != defaultThemeName || names[1] != alternateThemeName {
		t.Fatalf("expected stable theme names, got %#v", names)
	}
}

func TestMethodColorRespectsActiveTheme(t *testing.T) {
	original := CurrentTheme()
	t.Cleanup(func() {
		SetTheme(original)
	})

	defaultGet := DefaultTheme().Palette.MethodGet
	if ok := SetThemeByName(alternateThemeName); !ok {
		t.Fatal("expected alternate theme selection to succeed")
	}
	if MethodColor("GET") == defaultGet {
		t.Fatal("expected GET method color to change with active theme")
	}
}
