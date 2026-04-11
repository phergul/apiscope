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

// Data contains the render-ready state for the details pane.
type Data struct {
	LoadInFlight  bool
	LoadErrorBody string
	Selected      *model.Operation
	FilterText    string
	ActiveSection string
	Security      *model.SecurityRequirement
	Warnings      []model.SpecWarning
	Sections      []widgets.Section
}

// AvailableSections returns the visible details sections for the selected operation.
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

// ResolveActiveSection returns the active details section, falling back when needed.
func ResolveActiveSection(current string, selected *model.Operation, security *model.SecurityRequirement, warnings []model.SpecWarning) string {
	return widgets.ResolveActiveSection(current, AvailableSections(selected, security, warnings), SectionSummary)
}

// MoveActiveSection moves the active details section by the requested direction.
func MoveActiveSection(current string, direction int, selected *model.Operation, security *model.SecurityRequirement, warnings []model.SpecWarning) string {
	return widgets.MoveActiveSection(current, AvailableSections(selected, security, warnings), direction, SectionSummary)
}

// BoundaryActiveSection returns the first or last available details section.
func BoundaryActiveSection(last bool, selected *model.Operation, security *model.SecurityRequirement, warnings []model.SpecWarning) string {
	return widgets.BoundaryActiveSection(AvailableSections(selected, security, warnings), last, SectionSummary)
}

// Render renders the details pane from its render-ready data.
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
		Sections:   dataSections(data),
		Active:     data.ActiveSection,
		EmptyState: "",
	})
}

// RenderActiveSection renders the currently active details section body.
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

// Sections builds the unwindowed details sections for the supplied data.
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

// dataSections returns the projected sections when present or builds them on demand.
func dataSections(data Data) []widgets.Section {
	if len(data.Sections) > 0 {
		return append([]widgets.Section(nil), data.Sections...)
	}

	return Sections(data)
}

// renderSummaryContent renders the summary section for the selected operation.
func renderSummaryContent(data Data) string {
	return strings.Join([]string{
		renderSummaryField("Summary", fallbackText(data.Selected.Summary, "None")),
		renderSummaryField("Description", fallbackText(data.Selected.Description, "None")),
		renderSummaryField("Tags", formatTags(data.Selected.Tags)),
		renderSummaryField("Deprecated", yesNo(data.Selected.Deprecated)),
	}, "\n")
}

func renderSummaryField(label, value string) string {
	lines := strings.Split(value, "\n")
	body := make([]string, 0, len(lines))
	body = append(body, fmt.Sprintf("%s %s", widgets.MutedTextStyle().Render(label+":"), lines[0]))
	for _, line := range lines[1:] {
		body = append(body, "  "+line)
	}

	return strings.Join(body, "\n")
}

// fallbackText returns a trimmed value or the provided fallback string.
func fallbackText(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}

	return value
}

// yesNo formats a boolean as an explicit yes or no string.
func yesNo(value bool) string {
	if value {
		return "yes"
	}

	return "no"
}

// formatTags joins operation tags or returns an explicit empty-state label.
func formatTags(tags []string) string {
	if len(tags) == 0 {
		return "None"
	}

	return strings.Join(tags, ", ")
}

// formatSecurityRequirement renders the effective security requirement for the pane.
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

// formatWarnings renders spec warnings for the warnings details section.
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
