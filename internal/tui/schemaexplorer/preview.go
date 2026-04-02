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

func previewBody(node *treeNode, width int) string {
	lines := previewLines(node)
	return strings.Join(widgets.WrapLines(lines, width), "\n")
}

func previewLines(node *treeNode) []string {
	if node == nil {
		return []string{"No schema selected."}
	}
	if node.Schema == nil {
		return groupPreviewLines(node)
	}

	schema := node.Schema
	lines := make([]string, 0, 24)
	if path := previewPath(node); strings.TrimSpace(path) != "" {
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
		lines = append(lines, "", renderMetaLine("Description", schema.Description))
	}
	if len(schema.Required) > 0 {
		lines = append(lines, "", renderMetaLine("Required", strings.Join(schema.Required, ", ")))
	}
	if len(schema.Enum) > 0 {
		lines = append(lines, "", renderMetaLine("Enum", fmt.Sprintf("%d values", len(schema.Enum))))
		for _, value := range schema.Enum {
			lines = append(lines, "- "+formatValue(value))
		}
	}
	if example := formatStructuredValue(schema.Example); strings.TrimSpace(example) != "" {
		lines = append(lines, "", renderMetaLine("Example", example))
	}
	if defaultValue := formatStructuredValue(schema.Default); strings.TrimSpace(defaultValue) != "" {
		lines = append(lines, "", renderMetaLine("Default", defaultValue))
	}

	children := childSummary(schema)
	if len(children) > 0 {
		lines = append(lines, "", renderMetaLine("Children", ""))
		lines = append(lines, children...)
	}

	if strings.TrimSpace(node.Note) != "" {
		lines = append(lines, "", renderMetaLine("Note", node.Note))
	}

	return lines
}

func groupPreviewLines(node *treeNode) []string {
	lines := []string{
		renderMetaLine("Path", previewPath(node)),
		renderMetaLine("Entries", fmt.Sprintf("%d", len(node.Children))),
	}
	if len(node.Children) == 0 {
		lines = append(lines, "", renderMetaLine("Children", "- none"))
		return lines
	}

	lines = append(lines, "", renderMetaLine("Children", ""))
	for _, child := range node.Children {
		lines = append(lines, "- "+previewLabel(child))
	}

	return lines
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
