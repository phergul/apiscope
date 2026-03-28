package response

import (
	"strings"

	"github.com/phergul/apiscope/internal/model"
	"github.com/phergul/apiscope/internal/tui/describe"
	"github.com/phergul/apiscope/internal/tui/widgets"
)

type Data struct {
	LoadInFlight  bool
	Sections      []widgets.Section
	ActiveSection string
	EmptyState    string
}

func AvailableSections(responses []model.ResponseSpec) []string {
	sections := make([]string, 0, len(responses))
	for _, response := range responses {
		sections = append(sections, response.StatusCode)
	}

	return sections
}

func Sections(responses []model.ResponseSpec) []widgets.Section {
	sections := make([]widgets.Section, 0, len(responses))
	for _, response := range responses {
		sections = append(sections, widgets.Section{
			Label: response.StatusCode,
			Body:  sectionBody(response),
		})
	}

	return sections
}

func Render(data Data) string {
	if data.LoadInFlight {
		return "Loading spec..."
	}

	return widgets.RenderSectionView(widgets.SectionViewData{
		Sections:   data.Sections,
		Active:     data.ActiveSection,
		EmptyState: data.EmptyState,
	})
}

func sectionBody(response model.ResponseSpec) string {
	lines := []string{
		"Description: " + describe.NormaliseInlineText(fallbackText(response.Description, "None")),
		"Headers:",
	}
	if len(response.Headers) == 0 {
		lines = append(lines, "- none")
	} else {
		for _, header := range response.Headers {
			lines = append(lines, "- "+header.Name+" ("+describe.ParameterTypeHint(header)+")")
		}
	}
	lines = append(lines, "Media types: "+strings.Join(describe.DefaultStrings(describe.MediaTypesForContent(response.Content), "none"), ", "))

	return strings.Join(lines, "\n")
}

func fallbackText(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}

	return value
}
