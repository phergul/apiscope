package tui

import "strings"

func splitLines(text string) []string {
	if text == "" {
		return []string{""}
	}

	return strings.Split(text, "\n")
}

func clampLines(lines []string, offset, height int) []string {
	if len(lines) == 0 {
		return []string{""}
	}
	if offset < 0 {
		offset = 0
	}
	if offset >= len(lines) {
		offset = len(lines) - 1
	}
	if height <= 0 {
		height = 1
	}

	end := offset + height
	if end > len(lines) {
		end = len(lines)
	}

	return lines[offset:end]
}
