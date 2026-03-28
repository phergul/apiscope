package details

import (
	"fmt"
	"strings"

	"github.com/phergul/apiscope/internal/model"
	"github.com/phergul/apiscope/internal/tui/widgets"
)

const (
	SectionSummary  = "Summary"
	SectionSecurity = "Security"
	SectionWarnings = "Warnings"
)

type Data struct {
	LoadInFlight  bool
	LoadErrorBody string
	Selected      *model.Operation
	FilterText    string
	ActiveSection string
	Security      *model.SecurityRequirement
	Warnings      []model.SpecWarning
}

func AvailableSections(selected *model.Operation, security *model.SecurityRequirement, warnings []model.SpecWarning) []string {
	if selected == nil {
		return []string{SectionSummary}
	}

	sections := []string{SectionSummary}
	if security != nil && len(security.Alternatives) > 0 {
		sections = append(sections, SectionSecurity)
	}
	if len(warnings) > 0 {
		sections = append(sections, SectionWarnings)
	}

	return sections
}

func Render(data Data) string {
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

	return widgets.RenderSectionView(widgets.SectionViewData{
		Sections:   Sections(data),
		Active:     data.ActiveSection,
		EmptyState: "",
	})
}

func RenderActiveSection(data Data) string {
	switch data.ActiveSection {
	case SectionSecurity:
		return formatSecurityRequirement(data.Security)
	case SectionWarnings:
		return formatWarnings(data.Warnings)
	default:
		return renderSummaryContent(data)
	}
}

func Sections(data Data) []widgets.Section {
	sections := []widgets.Section{
		{Label: SectionSummary, Body: RenderActiveSection(Data{
			Selected:      data.Selected,
			ActiveSection: SectionSummary,
		})},
	}
	if data.Security != nil && len(data.Security.Alternatives) > 0 {
		sections = append(sections, widgets.Section{
			Label: SectionSecurity,
			Body: RenderActiveSection(Data{
				Security:      data.Security,
				ActiveSection: SectionSecurity,
			}),
		})
	}
	if len(data.Warnings) > 0 {
		sections = append(sections, widgets.Section{
			Label: SectionWarnings,
			Body: RenderActiveSection(Data{
				Warnings:      data.Warnings,
				ActiveSection: SectionWarnings,
			}),
		})
	}

	return sections
}

func renderSummaryContent(data Data) string {
	return strings.Join([]string{
		fmt.Sprintf("Summary: %s", fallbackText(data.Selected.Summary, "None")),
		fmt.Sprintf("Description: %s", fallbackText(data.Selected.Description, "None")),
		fmt.Sprintf("Tags: %s", formatTags(data.Selected.Tags)),
		fmt.Sprintf("Deprecated: %s", yesNo(data.Selected.Deprecated)),
	}, "\n")
}

func fallbackText(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}

	return value
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}

	return "no"
}

func formatTags(tags []string) string {
	if len(tags) == 0 {
		return "None"
	}

	return strings.Join(tags, ", ")
}

func formatSecurityRequirement(requirement *model.SecurityRequirement) string {
	if requirement == nil || len(requirement.Alternatives) == 0 {
		return "None"
	}

	lines := make([]string, 0, len(requirement.Alternatives))
	for _, alternative := range requirement.Alternatives {
		parts := make([]string, 0, len(alternative.Schemes))
		for _, scheme := range alternative.Schemes {
			part := scheme.Name
			if len(scheme.Scopes) > 0 {
				part += " (" + strings.Join(scheme.Scopes, ", ") + ")"
			}
			parts = append(parts, part)
		}
		if len(parts) == 0 {
			continue
		}
		lines = append(lines, "- "+strings.Join(parts, " AND "))
	}
	if len(lines) == 0 {
		return "None"
	}

	return strings.Join(lines, "\nOR\n")
}

func formatWarnings(warnings []model.SpecWarning) string {
	if len(warnings) == 0 {
		return "No warnings."
	}

	lines := make([]string, 0, len(warnings)*3)
	for _, warning := range warnings {
		lines = append(lines, fmt.Sprintf("- %s: %s", warning.Code, warning.Message))
		if strings.TrimSpace(warning.Path) != "" {
			lines = append(lines, fmt.Sprintf("  path: %s", warning.Path))
		}
	}

	return strings.Join(lines, "\n")
}
