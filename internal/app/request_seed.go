package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/phergul/apiscope/internal/model"
)

func seedRequestDraft(draft *model.RequestDraft, operation *model.Operation) {
	if draft == nil || operation == nil {
		return
	}

	for _, parameter := range operation.Parameters {
		seedDraftParameter(draft, parameter)
	}

	if strings.TrimSpace(draft.BodyMediaType) == "" {
		draft.BodyMediaType = defaultDraftBodyMediaType(operation)
	}
	seedDraftBody(draft, operation)
}

func defaultDraftBodyMediaType(operation *model.Operation) string {
	if operation == nil {
		return ""
	}
	if strings.TrimSpace(operation.SelectedContentType) != "" {
		return strings.TrimSpace(operation.SelectedContentType)
	}
	if operation.RequestBody == nil || len(operation.RequestBody.Content) == 0 {
		return ""
	}
	return operation.RequestBody.Content[0].MediaType
}

func seedDraftParameter(draft *model.RequestDraft, parameter model.Parameter) {
	target := parameterValueMap(draft, parameter)
	if target == nil {
		return
	}
	if _, exists := target[parameter.Name]; exists {
		return
	}

	value, ok := seededParameterValue(parameter)
	if !ok {
		return
	}
	target[parameter.Name] = value
}

func seededParameterValue(parameter model.Parameter) (string, bool) {
	if len(parameter.Content) > 0 || parameter.FormInputKind == model.FormInputKindFile {
		return "", false
	}

	value, ok := firstStructuredSeedValue(parameter.Example, parameter.Default, parameter.Schema)
	if !ok {
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
		return "", false
	}
}

func seedDraftBody(draft *model.RequestDraft, operation *model.Operation) {
	if draft == nil || operation == nil || strings.TrimSpace(draft.BodyRaw) != "" {
		return
	}

	body, exampleName, ok := seededRequestBody(operation, draft.BodyMediaType)
	if !ok {
		return
	}

	draft.BodyRaw = body
	setSelectedBodyExample(draft, draft.BodyMediaType, exampleName)
}

func shouldReplaceSeededBody(operation *model.Operation, draft *model.RequestDraft) bool {
	if operation == nil || draft == nil || strings.TrimSpace(draft.BodyRaw) == "" {
		return true
	}

	seeded, _, ok := seededRequestBody(operation, draft.BodyMediaType)
	return ok && draft.BodyRaw == seeded
}

func seededRequestBody(operation *model.Operation, mediaType string) (string, string, bool) {
	spec, ok := requestBodyMediaType(operation, mediaType)
	if !ok {
		return "", "", false
	}

	value, exampleName, ok := seededMediaTypeValue(spec)
	if !ok {
		return "", "", false
	}

	body, ok := formatSeededBody(value)
	if !ok {
		return "", "", false
	}

	return body, exampleName, true
}

func requestBodyMediaType(operation *model.Operation, mediaType string) (model.MediaTypeSpec, bool) {
	if operation == nil || operation.RequestBody == nil || len(operation.RequestBody.Content) == 0 {
		return model.MediaTypeSpec{}, false
	}

	trimmed := strings.TrimSpace(mediaType)
	if trimmed == "" {
		return operation.RequestBody.Content[0], true
	}

	for _, content := range operation.RequestBody.Content {
		if content.MediaType == trimmed {
			return content, true
		}
	}

	return model.MediaTypeSpec{}, false
}

func seededMediaTypeValue(spec model.MediaTypeSpec) (any, string, bool) {
	if spec.Example != nil {
		return spec.Example, "", true
	}
	if len(spec.Examples) > 0 {
		names := make([]string, 0, len(spec.Examples))
		for name := range spec.Examples {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			example := spec.Examples[name]
			if example.Value != nil {
				return example.Value, name, true
			}
		}
	}

	value, ok := schemaSeedValue(spec.Schema)
	return value, "", ok
}

func firstStructuredSeedValue(example, defaultValue any, schema *model.Schema) (any, bool) {
	if example != nil {
		return example, true
	}
	if defaultValue != nil {
		return defaultValue, true
	}
	return schemaSeedValue(schema)
}

func schemaSeedValue(schema *model.Schema) (any, bool) {
	if schema == nil {
		return nil, false
	}
	if schema.Example != nil {
		return schema.Example, true
	}
	if schema.Default != nil {
		return schema.Default, true
	}

	if value, ok := mergedSchemaSeedValues(schema.AllOf); ok {
		return value, true
	}
	if value, ok := firstSeededSchema(schema.OneOf); ok {
		return value, true
	}
	if value, ok := firstSeededSchema(schema.AnyOf); ok {
		return value, true
	}

	if schema.Items != nil && isArrayLikeSchema(schema) {
		item, ok := schemaSeedValue(schema.Items)
		if ok {
			return []any{item}, true
		}
	}

	if len(schema.Properties) > 0 || isObjectLikeSchema(schema) {
		names := make([]string, 0, len(schema.Properties))
		for name := range schema.Properties {
			names = append(names, name)
		}
		sort.Strings(names)

		seeded := make(map[string]any)
		for _, name := range names {
			value, ok := schemaSeedValue(schema.Properties[name])
			if ok {
				seeded[name] = value
			}
		}
		if len(seeded) > 0 {
			return seeded, true
		}
	}

	return nil, false
}

func firstSeededSchema(schemas []*model.Schema) (any, bool) {
	for _, schema := range schemas {
		value, ok := schemaSeedValue(schema)
		if ok {
			return value, true
		}
	}
	return nil, false
}

func mergedSchemaSeedValues(schemas []*model.Schema) (any, bool) {
	if len(schemas) == 0 {
		return nil, false
	}

	merged := make(map[string]any)
	for _, schema := range schemas {
		value, ok := schemaSeedValue(schema)
		if !ok {
			continue
		}
		object, ok := value.(map[string]any)
		if !ok {
			return value, true
		}
		for name, propertyValue := range object {
			merged[name] = propertyValue
		}
	}
	if len(merged) == 0 {
		return nil, false
	}
	return merged, true
}

func isArrayLikeSchema(schema *model.Schema) bool {
	return schema != nil && schema.Type == "array"
}

func isObjectLikeSchema(schema *model.Schema) bool {
	return schema != nil && (schema.Type == "object" || schema.Type == "")
}

func formatSeededBody(value any) (string, bool) {
	if value == nil {
		return "", false
	}

	switch seeded := value.(type) {
	case string:
		if json.Valid([]byte(seeded)) {
			var indented bytes.Buffer
			if err := json.Indent(&indented, []byte(seeded), "", "  "); err == nil {
				return indented.String(), true
			}
		}
		return seeded, true
	default:
		bytes, err := json.MarshalIndent(seeded, "", "  ")
		if err != nil {
			return "", false
		}
		return string(bytes), true
	}
}

func setSelectedBodyExample(draft *model.RequestDraft, mediaType, exampleName string) {
	if draft == nil || strings.TrimSpace(mediaType) == "" {
		return
	}
	if draft.SelectedExamples == nil {
		draft.SelectedExamples = make(map[string]string)
	}

	key := "body:" + mediaType
	if strings.TrimSpace(exampleName) == "" {
		delete(draft.SelectedExamples, key)
		return
	}
	draft.SelectedExamples[key] = exampleName
}
