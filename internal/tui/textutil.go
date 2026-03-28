package tui

import (
	"strings"
)

func splitLines(text string) []string {
	if text == "" {
		return []string{""}
	}

	return strings.Split(text, "\n")
}

func fallbackText(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}

	return value
}
