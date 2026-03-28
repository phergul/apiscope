package widgets

import "strings"

type ListItem struct {
	Content  string
	Selected bool
	Muted    bool
}

func RenderList(items []ListItem) string {
	lines := make([]string, 0, len(items))
	for _, item := range items {
		style := BodyTextStyle()
		prefix := "  "
		if item.Muted {
			style = MutedTextStyle()
		}
		if item.Selected {
			style = SelectedTextStyle()
			prefix = "> "
		}

		lines = append(lines, prefix+style.Render(item.Content))
	}

	return strings.Join(lines, "\n")
}
