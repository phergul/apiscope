package response

import (
	"fmt"
	"strings"

	"github.com/phergul/apiscope/internal/model"
	"github.com/phergul/apiscope/internal/tui/describe"
	"github.com/phergul/apiscope/internal/tui/widgets"
)

const SectionLive = "Live"

type Data struct {
	LoadInFlight  bool
	Sections      []widgets.Section
	ActiveSection string
	EmptyState    string
}

func AvailableSections(responses []model.ResponseSpec) []string {
	sections := make([]string, 0, len(responses)+1)
	sections = append(sections, SectionLive)
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

func LiveSection(response *model.HTTPResponse, selected *model.Operation) widgets.Section {
	body := "No request has been sent for this operation yet."
	if response != nil && selected != nil && response.OperationKey == selected.Key {
		body = liveSectionBody(response)
	}

	return widgets.Section{
		Label: SectionLive,
		Body:  body,
	}
}

func ActiveSectionBody(sections []widgets.Section, active string) string {
	if len(sections) == 0 {
		return ""
	}

	if active != "" {
		for _, section := range sections {
			if section.Label == active {
				return section.Body
			}
		}
	}

	return sections[0].Body
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

func liveSectionBody(response *model.HTTPResponse) string {
	if response == nil {
		return "No request has been sent for this operation yet."
	}

	status := response.Status
	if strings.TrimSpace(status) == "" {
		status = "No HTTP status"
	}

	lines := []string{
		fmt.Sprintf("Status: %s", status),
		fmt.Sprintf("Duration: %s", response.Duration),
	}
	if strings.TrimSpace(response.ContentType) != "" {
		lines = append(lines, fmt.Sprintf("Content type: %s", response.ContentType))
	}
	if response.ContentLength > 0 {
		lines = append(lines, fmt.Sprintf("Content length: %d", response.ContentLength))
	}
	if strings.TrimSpace(response.TransportError) != "" {
		lines = append(lines, widgets.ErrorTextStyle().Bold(true).Render("Transport error: "+response.TransportError))
	}

	lines = append(lines, "Headers:")
	if len(response.Headers) == 0 {
		lines = append(lines, "- none")
	} else {
		for name, values := range response.Headers {
			lines = append(lines, fmt.Sprintf("- %s: %s", name, strings.Join(values, ", ")))
		}
	}

	lines = append(lines, "Body:")
	body := strings.TrimSpace(response.PrettyBody)
	if body == "" {
		body = strings.TrimSpace(string(response.Body))
	}
	if body == "" {
		body = "<empty>"
	}
	lines = append(lines, body)

	return strings.Join(lines, "\n")
}

func fallbackText(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}

	return value
}
