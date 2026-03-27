package panes

import "strings"

type Section struct {
	Label string
	Body  string
}

func RenderSectionView(sections []Section, active, emptyState string) string {
	if len(sections) == 0 {
		return emptyState
	}

	return strings.Join([]string{
		RenderSectionStrip(sections, active),
		"",
		activeSectionBody(sections, active),
	}, "\n")
}

func RenderSectionStrip(sections []Section, active string) string {
	parts := make([]string, 0, len(sections))
	for _, section := range sections {
		label := section.Label
		if label == activeSectionLabel(sections, active) {
			label = "[" + label + "]"
		}
		parts = append(parts, label)
	}

	return strings.Join(parts, "  ")
}

func activeSectionBody(sections []Section, active string) string {
	active = activeSectionLabel(sections, active)
	for _, section := range sections {
		if section.Label == active {
			return section.Body
		}
	}

	return sections[0].Body
}

func activeSectionLabel(sections []Section, active string) string {
	if active != "" {
		for _, section := range sections {
			if section.Label == active {
				return active
			}
		}
	}

	if len(sections) == 0 {
		return ""
	}

	return sections[0].Label
}
