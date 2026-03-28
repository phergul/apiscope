package widgets

import (
	"slices"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type Section struct {
	Label string
	Body  string
}

type SectionStripData struct {
	Labels []string
	Active string
}

type SectionViewData struct {
	Sections   []Section
	Active     string
	EmptyState string
}

func ResolveActiveSection(current string, available []string, empty string) string {
	if len(available) == 0 {
		return empty
	}

	if slices.Contains(available, current) {
		return current
	}

	return available[0]
}

func MoveActiveSection(current string, available []string, direction int, empty string) string {
	if len(available) == 0 {
		return empty
	}

	current = ResolveActiveSection(current, available, empty)
	currentIndex := 0
	for index, section := range available {
		if section == current {
			currentIndex = index
			break
		}
	}

	targetIndex := currentIndex + direction
	if targetIndex < 0 || targetIndex >= len(available) {
		return current
	}

	return available[targetIndex]
}

func BoundaryActiveSection(available []string, last bool, empty string) string {
	if len(available) == 0 {
		return empty
	}
	if last {
		return available[len(available)-1]
	}

	return available[0]
}

func RenderSectionLabels(data SectionStripData) string {
	return RenderSectionStrip(data)
}

func RenderSectionView(data SectionViewData) string {
	if len(data.Sections) == 0 {
		return data.EmptyState
	}

	return strings.Join([]string{
		RenderSectionStrip(SectionStripData{
			Labels: sectionLabels(data.Sections),
			Active: activeSectionLabel(data.Sections, data.Active),
		}),
		"",
		activeSectionBody(data.Sections, data.Active),
	}, "\n")
}

func RenderSectionStrip(data SectionStripData) string {
	parts := make([]string, 0, len(data.Labels))
	for _, label := range data.Labels {
		renderLabel := label
		style := MutedTextStyle().Padding(0, 1)
		if label == data.Active {
			style = SelectedTextStyle().Bold(true).Padding(0, 1)
		}
		parts = append(parts, style.Render(renderLabel))
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, parts...)
}

func sectionLabels(sections []Section) []string {
	labels := make([]string, 0, len(sections))
	for _, section := range sections {
		labels = append(labels, section.Label)
	}

	return labels
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
