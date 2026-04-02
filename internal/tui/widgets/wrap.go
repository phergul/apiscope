package widgets

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// WrapLines wraps ANSI-aware lines to the available width before viewport clipping.
func WrapLines(lines []string, width int) []string {
	if width <= 0 {
		return append([]string(nil), lines...)
	}

	wrapped := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			wrapped = append(wrapped, "")
			continue
		}

		wordWrapped := ansi.Wordwrap(line, width, "")
		hardWrapped := ansi.Hardwrap(wordWrapped, width, true)
		wrapped = append(wrapped, strings.Split(hardWrapped, "\n")...)
	}

	return wrapped
}
