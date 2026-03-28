package panes

import (
	"fmt"
	"strings"

	"github.com/phergul/apiscope/internal/model"
)

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

func FormatSecurityRequirementForProjection(requirement *model.SecurityRequirement) string {
	return formatSecurityRequirement(requirement)
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
