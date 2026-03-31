package util

import "strings"

// fallbackText returns the value if it is not empty, otherwise returns the fallback.
func FallbackText(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}

	return value
}
