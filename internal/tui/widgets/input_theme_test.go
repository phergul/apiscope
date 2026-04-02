package widgets

import "testing"

func TestTextInputPlaceholderUsesCurrentThemeBackgroundAfterThemeSwitch(t *testing.T) {
	t.Parallel()

	original := CurrentTheme()
	t.Cleanup(func() {
		SetTheme(original)
	})

	input := NewTextInput()
	input.SetPlaceholder("Enter value")
	input.SetWidth(16)

	SetThemeByName(harborThemeName)
	input.refreshTheme()

	if input.model.PlaceholderStyle.GetBackground() != CurrentTheme().Palette.InputBackground {
		t.Fatalf("expected placeholder background %q, got %q", CurrentTheme().Palette.InputBackground, input.model.PlaceholderStyle.GetBackground())
	}
	if input.model.PlaceholderStyle.GetForeground() != CurrentTheme().Palette.InputPlaceholder {
		t.Fatalf("expected placeholder foreground %q, got %q", CurrentTheme().Palette.InputPlaceholder, input.model.PlaceholderStyle.GetForeground())
	}
}

func TestTextAreaPlaceholderUsesCurrentThemeBackgroundAfterThemeSwitch(t *testing.T) {
	t.Parallel()

	original := CurrentTheme()
	t.Cleanup(func() {
		SetTheme(original)
	})

	area := NewTextArea()
	area.SetPlaceholder("Enter raw request body")
	area.SetSize(24, 4)

	SetThemeByName(emberThemeName)
	area.refreshTheme()

	if area.model.FocusedStyle.Placeholder.GetBackground() != CurrentTheme().Palette.InputBackground {
		t.Fatalf("expected placeholder background %q, got %q", CurrentTheme().Palette.InputBackground, area.model.FocusedStyle.Placeholder.GetBackground())
	}
	if area.model.FocusedStyle.Placeholder.GetForeground() != CurrentTheme().Palette.InputPlaceholder {
		t.Fatalf("expected placeholder foreground %q, got %q", CurrentTheme().Palette.InputPlaceholder, area.model.FocusedStyle.Placeholder.GetForeground())
	}
}
