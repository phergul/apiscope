package converter

import (
	"fmt"
	"strings"

	"api-tui/internal/spec/internal/pipeline"

	"github.com/getkin/kin-openapi/openapi3"
)

func unsupportedSwaggerConstruct(source, location, message string) error {
	return &pipeline.Error{
		Kind:   pipeline.ErrorKindUnsupportedSwaggerConstruct,
		Op:     "convert swagger",
		Source: source,
		Err:    fmt.Errorf("%s: %s", location, message),
	}
}

func getMap(m map[string]any, key string) (map[string]any, bool) {
	raw, ok := m[key]
	if !ok {
		return nil, false
	}
	value, ok := raw.(map[string]any)
	return value, ok
}

func getSliceMap(m map[string]any, key string) []map[string]any {
	raw, ok := m[key]
	if !ok {
		return nil
	}

	items, ok := raw.([]any)
	if !ok {
		return nil
	}

	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if itemMap, ok := item.(map[string]any); ok {
			result = append(result, itemMap)
		}
	}

	return result
}

func getString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	value, ok := m[key]
	if !ok || value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func getStringSlice(m map[string]any, key string) []string {
	if m == nil {
		return nil
	}
	return stringSliceFromAny(m[key])
}

func stringSliceFromAny(raw any) []string {
	items, ok := raw.([]any)
	if !ok {
		return nil
	}

	result := make([]string, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		result = append(result, strings.TrimSpace(fmt.Sprint(item)))
	}

	return result
}

func stringMapFromAny(raw any) openapi3.StringMap {
	items, ok := raw.(map[string]any)
	if !ok {
		return nil
	}

	result := make(openapi3.StringMap, len(items))
	for key, value := range items {
		result[key] = strings.TrimSpace(fmt.Sprint(value))
	}

	return result
}

func getBool(m map[string]any, key string) bool {
	if m == nil {
		return false
	}
	value, ok := m[key]
	if !ok {
		return false
	}
	boolValue, ok := value.(bool)
	return ok && boolValue
}

func ptrString(value string) *string {
	return &value
}

func boolPtr(value bool) *bool {
	return &value
}
