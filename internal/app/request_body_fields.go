package app

import (
	"fmt"
	"sort"
	"strings"

	"github.com/phergul/apiscope/internal/model"
)

// ProjectBodyFieldParameters returns synthetic form-style parameters for supported form-like request bodies.
func ProjectBodyFieldParameters(operation *model.Operation, draft *model.RequestDraft) []model.Parameter {
	mediaType, spec, ok := activeBodyMediaTypeSpec(operation, draft)
	if !ok || !supportsBodyFieldEditing(mediaType) || spec.Schema == nil {
		return nil
	}

	names := make([]string, 0, len(spec.Schema.Properties))
	for name, property := range spec.Schema.Properties {
		if property == nil {
			continue
		}
		names = append(names, name)
	}
	if len(names) == 0 {
		return nil
	}
	sort.Strings(names)

	required := make(map[string]struct{}, len(spec.Schema.Required))
	for _, name := range spec.Schema.Required {
		required[name] = struct{}{}
	}

	parameters := make([]model.Parameter, 0, len(names))
	for _, name := range names {
		property := spec.Schema.Properties[name]
		parameter := model.Parameter{
			Name:     name,
			In:       model.ParameterLocationForm,
			Required: hasRequired(required, name),
			Schema:   property,
			Example:  property.Example,
			Default:  property.Default,
		}
		if isBodyFileSchema(mediaType, property) {
			parameter.FormInputKind = model.FormInputKindFile
		}
		parameters = append(parameters, parameter)
	}

	return parameters
}

func supportsBodyFieldEditing(mediaType string) bool {
	switch strings.TrimSpace(mediaType) {
	case "multipart/form-data", "application/x-www-form-urlencoded":
		return true
	default:
		return false
	}
}

func bodyMediaType(operation *model.Operation, draft *model.RequestDraft) string {
	if draft != nil && strings.TrimSpace(draft.BodyMediaType) != "" {
		return strings.TrimSpace(draft.BodyMediaType)
	}
	if operation != nil && strings.TrimSpace(operation.SelectedContentType) != "" {
		return strings.TrimSpace(operation.SelectedContentType)
	}
	if operation != nil && operation.RequestBody != nil && len(operation.RequestBody.Content) > 0 {
		return strings.TrimSpace(operation.RequestBody.Content[0].MediaType)
	}

	return ""
}

func hasRequired(required map[string]struct{}, name string) bool {
	_, ok := required[name]
	return ok
}

func isBodyFileSchema(mediaType string, schema *model.Schema) bool {
	if schema == nil {
		return false
	}
	if strings.TrimSpace(mediaType) != "multipart/form-data" {
		return false
	}

	return strings.TrimSpace(schema.Type) == "string" && strings.TrimSpace(schema.Format) == "binary"
}

func seedDraftBodyFields(draft *model.RequestDraft, operation *model.Operation) {
	for _, parameter := range ProjectBodyFieldParameters(operation, draft) {
		seedDraftBodyField(draft, parameter)
	}
}

func seedDraftBodyField(draft *model.RequestDraft, parameter model.Parameter) {
	target := parameterValueMap(draft, parameter)
	if target == nil {
		return
	}
	if _, exists := target[parameter.Name]; exists {
		return
	}
	if parameter.FormInputKind == model.FormInputKindFile {
		return
	}

	value, ok := firstStructuredSeedValue(parameter.Example, parameter.Default, parameter.Schema)
	if !ok {
		return
	}

	formatted, ok := formatBodyFieldSeedValue(value)
	if !ok {
		return
	}
	target[parameter.Name] = formatted
}

func formatBodyFieldSeedValue(value any) (string, bool) {
	if value == nil {
		return "", false
	}

	switch seeded := value.(type) {
	case string:
		return seeded, true
	case bool:
		return fmt.Sprintf("%t", seeded), true
	case float32, float64, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprint(seeded), true
	default:
		return formatSeededBody(value)
	}
}
