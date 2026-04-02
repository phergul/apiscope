package schemaexplorer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/phergul/apiscope/internal/model"
	"github.com/phergul/apiscope/internal/tui/describe"
	"github.com/phergul/apiscope/internal/tui/widgets"
)

func previewBody(state State, schema *model.Schema, label, note string) string {
	if schema == nil {
		return "No schema selected."
	}

	lines := make([]string, 0, 24)
	if path := breadcrumbPath(state, label); strings.TrimSpace(path) != "" {
		lines = append(lines, renderMetaLine("Path", path))
	}
	if strings.TrimSpace(schema.Ref) != "" {
		lines = append(lines, renderMetaLine("Ref", schema.Ref))
	}
	if strings.TrimSpace(schema.Title) != "" {
		lines = append(lines, renderMetaLine("Title", schema.Title))
	}
	lines = append(lines, renderMetaLine("Type", describe.SchemaTypeHint(schema)))
	if strings.TrimSpace(schema.Description) != "" {
		lines = append(lines, "", "Description:", schema.Description)
	}
	if len(schema.Required) > 0 {
		lines = append(lines, "", renderMetaLine("Required", strings.Join(schema.Required, ", ")))
	}
	if len(schema.Enum) > 0 {
		lines = append(lines, "", "Enum:")
		for _, value := range schema.Enum {
			lines = append(lines, "- "+formatValue(value))
		}
	}
	if example := formatStructuredValue(schema.Example); strings.TrimSpace(example) != "" {
		lines = append(lines, "", "Example:", example)
	}
	if defaultValue := formatStructuredValue(schema.Default); strings.TrimSpace(defaultValue) != "" {
		lines = append(lines, "", "Default:", defaultValue)
	}

	lines = append(lines, "", "Children:")
	children := childSummary(schema)
	if len(children) == 0 {
		lines = append(lines, "- none")
	} else {
		lines = append(lines, children...)
	}

	if strings.TrimSpace(note) != "" {
		lines = append(lines, "", renderMetaLine("Note", note))
	}

	return strings.Join(lines, "\n")
}

func childSummary(schema *model.Schema) []string {
	if schema == nil {
		return nil
	}

	lines := make([]string, 0, 5)
	if len(schema.Properties) > 0 {
		lines = append(lines, fmt.Sprintf("- properties: %d", len(schema.Properties)))
	}
	if schema.Items != nil {
		lines = append(lines, "- items: present")
	}
	if len(schema.OneOf) > 0 {
		lines = append(lines, fmt.Sprintf("- oneOf: %d", len(schema.OneOf)))
	}
	if len(schema.AnyOf) > 0 {
		lines = append(lines, fmt.Sprintf("- anyOf: %d", len(schema.AnyOf)))
	}
	if len(schema.AllOf) > 0 {
		lines = append(lines, fmt.Sprintf("- allOf: %d", len(schema.AllOf)))
	}

	return lines
}

func breadcrumbPath(state State, label string) string {
	parts := make([]string, 0, len(state.Breadcrumbs)+1)
	for _, breadcrumb := range state.Breadcrumbs {
		parts = append(parts, breadcrumb.Label)
	}
	if strings.TrimSpace(label) != "" && (len(parts) == 0 || parts[len(parts)-1] != label) {
		parts = append(parts, label)
	}

	return strings.Join(parts, " > ")
}

func renderMetaLine(label, value string) string {
	return widgets.MutedTextStyle().Render(label+": ") + widgets.BodyTextStyle().Render(value)
}

func formatStructuredValue(value any) string {
	if value == nil {
		return ""
	}

	switch typed := value.(type) {
	case string:
		if json.Valid([]byte(typed)) {
			var buffer bytes.Buffer
			if err := json.Indent(&buffer, []byte(typed), "", "  "); err == nil {
				return buffer.String()
			}
		}
		return typed
	default:
		bytes, err := json.MarshalIndent(typed, "", "  ")
		if err != nil {
			return fmt.Sprint(value)
		}
		return string(bytes)
	}
}

func formatValue(value any) string {
	if value == nil {
		return "null"
	}
	if formatted := formatStructuredValue(value); strings.TrimSpace(formatted) != "" {
		return formatted
	}
	return fmt.Sprint(value)
}
