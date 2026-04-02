package widgets

import "strings"

// NormalizeRenderedBody converts carriage returns into line breaks and strips
// control bytes that can corrupt terminal layout when body text is rendered.
func NormalizeRenderedBody(body string) string {
	body = strings.ReplaceAll(body, "\r\n", "\n")
	body = strings.ReplaceAll(body, "\r", "\n")

	return strings.Map(func(r rune) rune {
		switch {
		case r == '\n' || r == '\t':
			return r
		case r < 0x20 || r == 0x7f:
			return ' '
		default:
			return r
		}
	}, body)
}
