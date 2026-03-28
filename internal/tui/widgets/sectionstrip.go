package widgets

import "github.com/charmbracelet/lipgloss"

func RenderSectionStrip(labels []string, active string) string {
	parts := make([]string, 0, len(labels))
	for _, label := range labels {
		renderLabel := label
		style := MutedTextStyle().Padding(0, 1)
		if label == active {
			style = SelectedTextStyle().Bold(true).Padding(0, 1)
		}
		parts = append(parts, style.Render(renderLabel))
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, parts...)
}
