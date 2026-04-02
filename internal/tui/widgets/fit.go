package widgets

import "github.com/charmbracelet/x/ansi"

// FitLine clips one ANSI-aware line to the requested width.
func FitLine(line string, width int) string {
	if width <= 0 {
		return line
	}

	return ansi.Truncate(line, width, "")
}
