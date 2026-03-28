package widgets

import "strings"

type ListItem struct {
	Content  string
	Selected bool
	Muted    bool
	Width    int
}

func RenderList(items []ListItem) string {
	lines := make([]string, 0, len(items))
	for _, item := range items {
		style := BodyTextStyle()
		padding := " "
		if item.Muted {
			style = MutedTextStyle()
		}
		if item.Selected {
			style = SelectedTextStyle()
		}

		content := padding + item.Content + padding
		if item.Width > 0 {
			style = style.Width(item.Width).MaxWidth(item.Width)
		}

		lines = append(lines, style.Render(content))
	}

	return strings.Join(lines, "\n")
}
