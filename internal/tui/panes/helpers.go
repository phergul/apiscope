package panes

import (
	"fmt"
	"strings"

	"api-tui/internal/model"
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

func formatParameterSections(parameters []model.Parameter) string {
	locations := []model.ParameterLocation{
		model.ParameterLocationPath,
		model.ParameterLocationQuery,
		model.ParameterLocationHeader,
		model.ParameterLocationCookie,
	}

	lines := make([]string, 0, len(locations)*2)
	for _, location := range locations {
		lines = append(lines, strings.ToUpper(string(location))+":")

		count := 0
		for _, parameter := range parameters {
			if parameter.In != location {
				continue
			}

			count++
			lines = append(lines, fmt.Sprintf(
				"- %s (%s, %s)",
				parameter.Name,
				requiredLabel(parameter.Required),
				formatParameterTypeHint(parameter),
			))
		}
		if count == 0 {
			lines = append(lines, "- none")
		}
	}

	return strings.Join(lines, "\n")
}

func requiredLabel(required bool) string {
	if required {
		return "required"
	}

	return "optional"
}

func formatParameterTypeHint(parameter model.Parameter) string {
	if parameter.Schema != nil {
		return formatSchemaType(parameter.Schema)
	}
	if len(parameter.Content) > 0 {
		return "content"
	}

	return "unknown"
}

func formatSchemaType(schema *model.Schema) string {
	if schema == nil {
		return "unknown"
	}

	parts := make([]string, 0, 2)
	if schema.Type != "" {
		parts = append(parts, schema.Type)
	}
	if schema.Format != "" {
		parts = append(parts, schema.Format)
	}
	if len(parts) > 0 {
		return strings.Join(parts, "/")
	}
	if schema.Ref != "" {
		return schema.Ref
	}
	if len(schema.OneOf) > 0 {
		return "oneOf"
	}
	if len(schema.AnyOf) > 0 {
		return "anyOf"
	}
	if len(schema.AllOf) > 0 {
		return "allOf"
	}

	return "object"
}

func formatRequestBody(body *model.RequestBodySpec) string {
	if body == nil {
		return "None"
	}

	required := "optional"
	if body.Required {
		required = "required"
	}

	mediaTypes := make([]string, 0, len(body.Content))
	for _, content := range body.Content {
		mediaTypes = append(mediaTypes, content.MediaType)
	}
	if len(mediaTypes) == 0 {
		mediaTypes = append(mediaTypes, "none")
	}

	lines := []string{
		fmt.Sprintf("Required: %s", required),
		fmt.Sprintf("Media types: %s", strings.Join(mediaTypes, ", ")),
	}
	if description := strings.TrimSpace(body.Description); description != "" {
		lines = append(lines, fmt.Sprintf("Description: %s", description))
	}

	return strings.Join(lines, "\n")
}

func formatResponses(responses []model.ResponseSpec) string {
	if len(responses) == 0 {
		return "None"
	}

	lines := make([]string, 0, len(responses))
	for _, response := range responses {
		mediaTypes := make([]string, 0, len(response.Content))
		for _, content := range response.Content {
			mediaTypes = append(mediaTypes, content.MediaType)
		}
		if len(mediaTypes) == 0 {
			mediaTypes = append(mediaTypes, "none")
		}

		description := fallbackText(response.Description, "None")
		lines = append(lines, fmt.Sprintf("- %s: %s [%s]", response.StatusCode, description, strings.Join(mediaTypes, ", ")))
	}

	return strings.Join(lines, "\n")
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
