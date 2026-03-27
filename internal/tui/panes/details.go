package panes

import (
	"fmt"
	"strings"

	"api-tui/internal/model"
)

const (
	DetailsSectionSummary  = "Summary"
	DetailsSectionSecurity = "Security"
	DetailsSectionWarnings = "Warnings"
)

type DetailsData struct {
	LoadInFlight  bool
	LoadErrorBody string
	Selected      *model.Operation
	FilterText    string
	Sections      []string
	ActiveSection string
	Security      *model.SecurityRequirement
	Warnings      []model.SpecWarning
}

func RenderDetails(data DetailsData) string {
	switch {
	case data.LoadInFlight:
		return "Loading spec..."
	case strings.TrimSpace(data.LoadErrorBody) != "":
		return data.LoadErrorBody
	}

	if data.Selected == nil {
		lines := []string{
			"No operation selected.",
			"Choose an operation in pane 1 to inspect its details.",
		}
		if strings.TrimSpace(data.FilterText) != "" {
			lines = append(lines, "If the list is empty, press Esc to clear the filter.")
		}
		return strings.Join(lines, "\n")
	}

	return strings.Join([]string{
		renderDetailsSectionStrip(data.Sections, data.ActiveSection),
		"",
		renderActiveDetailsSectionContent(data),
	}, "\n")
}

func renderActiveDetailsSectionContent(data DetailsData) string {
	switch data.ActiveSection {
	case DetailsSectionSecurity:
		return formatSecurityRequirement(data.Security)
	case DetailsSectionWarnings:
		return formatWarnings(data.Warnings)
	default:
		return strings.Join([]string{
			fmt.Sprintf("Operation: %s %s", strings.ToUpper(data.Selected.Method), data.Selected.Path),
			fmt.Sprintf("Summary: %s", fallbackText(data.Selected.Summary, "None")),
			fmt.Sprintf("Description: %s", fallbackText(data.Selected.Description, "None")),
			fmt.Sprintf("Tags: %s", formatTags(data.Selected.Tags)),
			fmt.Sprintf("Deprecated: %s", yesNo(data.Selected.Deprecated)),
		}, "\n")
	}
}

func renderDetailsSectionStrip(sections []string, active string) string {
	parts := make([]string, 0, len(sections))
	for _, section := range sections {
		label := section
		if section == active {
			label = "[" + label + "]"
		}
		parts = append(parts, label)
	}

	return strings.Join(parts, "  ")
}
