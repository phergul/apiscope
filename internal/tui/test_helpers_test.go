package tui

import "github.com/charmbracelet/x/ansi"

func stripANSI(value string) string {
	return ansi.Strip(value)
}
