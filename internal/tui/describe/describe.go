package describe

import (
	"strings"

	"github.com/phergul/apiscope/internal/model"
)

func MediaTypesForContent(content []model.MediaTypeSpec) []string {
	mediaTypes := make([]string, 0, len(content))
	for _, item := range content {
		mediaTypes = append(mediaTypes, item.MediaType)
	}

	return mediaTypes
}

func ParametersInLocation(parameters []model.Parameter, location model.ParameterLocation) []model.Parameter {
	filtered := make([]model.Parameter, 0, len(parameters))
	for _, parameter := range parameters {
		if parameter.In == location {
			filtered = append(filtered, parameter)
		}
	}

	return filtered
}

func ParameterTypeHint(parameter model.Parameter) string {
	if parameter.Schema != nil {
		return SchemaTypeHint(parameter.Schema)
	}
	if len(parameter.Content) > 0 {
		return "content"
	}

	return "unknown"
}

func SchemaTypeHint(schema *model.Schema) string {
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

func NormaliseInlineText(value string) string {
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return "None"
	}

	return strings.Join(fields, " ")
}

func DefaultStrings(values []string, fallback string) []string {
	if len(values) > 0 {
		return values
	}

	return []string{fallback}
}

func BooleanRequirementLabel(required bool) string {
	if required {
		return "required"
	}

	return "optional"
}
