package panes

import (
	"fmt"
	"strings"

	"github.com/phergul/apiscope/internal/model"
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

	sections := []Section{
		{Label: DetailsSectionSummary, Body: renderSummaryDetailsContent(data)},
	}
	if data.Security != nil && len(data.Security.Alternatives) > 0 {
		sections = append(sections, Section{
			Label: DetailsSectionSecurity,
			Body:  formatSecurityRequirement(data.Security),
		})
	}
	if len(data.Warnings) > 0 {
		sections = append(sections, Section{
			Label: DetailsSectionWarnings,
			Body:  formatWarnings(data.Warnings),
		})
	}

	return RenderSectionView(sections, data.ActiveSection, "")
}

func renderSummaryDetailsContent(data DetailsData) string {
	return strings.Join([]string{
		fmt.Sprintf("Summary: %s", fallbackText(data.Selected.Summary, "None")),
		fmt.Sprintf("Description: %s", fallbackText(data.Selected.Description, "None")),
		fmt.Sprintf("Tags: %s", formatTags(data.Selected.Tags)),
		fmt.Sprintf("Deprecated: %s", yesNo(data.Selected.Deprecated)),
	}, "\n")
}

func RenderActiveDetailsSectionForProjection(data DetailsData) string {
	switch data.ActiveSection {
	case DetailsSectionSecurity:
		return formatSecurityRequirement(data.Security)
	case DetailsSectionWarnings:
		return formatWarnings(data.Warnings)
	default:
		return renderSummaryDetailsContent(data)
	}
}

func BuildDetailsSectionsForProjection(data DetailsData) []Section {
	sections := []Section{
		{Label: DetailsSectionSummary, Body: RenderActiveDetailsSectionForProjection(DetailsData{
			Selected:      data.Selected,
			ActiveSection: DetailsSectionSummary,
		})},
	}
	if data.Security != nil && len(data.Security.Alternatives) > 0 {
		sections = append(sections, Section{
			Label: DetailsSectionSecurity,
			Body: RenderActiveDetailsSectionForProjection(DetailsData{
				Security:      data.Security,
				ActiveSection: DetailsSectionSecurity,
			}),
		})
	}
	if len(data.Warnings) > 0 {
		sections = append(sections, Section{
			Label: DetailsSectionWarnings,
			Body: RenderActiveDetailsSectionForProjection(DetailsData{
				Warnings:      data.Warnings,
				ActiveSection: DetailsSectionWarnings,
			}),
		})
	}

	return sections
}
