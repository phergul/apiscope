package converter

import (
	"fmt"
	"strings"

	"api-tui/internal/spec/internal/pipeline"

	"github.com/getkin/kin-openapi/openapi3"
)

func convertSwaggerDefinitions(parsed *pipeline.ParsedDocument) (openapi3.Schemas, error) {
	rawDefinitions, ok := getMap(parsed.SwaggerDoc, "definitions")
	if !ok || len(rawDefinitions) == 0 {
		return nil, nil
	}

	schemas := make(openapi3.Schemas, len(rawDefinitions))
	for name, rawDefinition := range rawDefinitions {
		schema, err := convertSchemaRef(parsed.Document.CanonicalLocation, "definitions."+name, rawDefinition)
		if err != nil {
			return nil, err
		}
		schemas[name] = schema
	}

	return schemas, nil
}

func convertSchemaFromParameter(source, location string, rawParameter map[string]any) (*openapi3.SchemaRef, error) {
	if rawSchema, ok := rawParameter["schema"]; ok {
		return convertSchemaRef(source, location+".schema", rawSchema)
	}

	schemaMap := map[string]any{}
	for _, key := range []string{"type", "format", "description", "enum", "items", "default"} {
		if value, ok := rawParameter[key]; ok {
			schemaMap[key] = value
		}
	}

	if len(schemaMap) == 0 {
		schemaMap["type"] = "string"
	}

	return convertSchemaRef(source, location+".schema", schemaMap)
}

func convertSchemaRef(source, location string, raw any) (*openapi3.SchemaRef, error) {
	rawMap, ok := raw.(map[string]any)
	if !ok {
		return nil, &pipeline.Error{
			Kind:   pipeline.ErrorKindSwaggerConversionFailure,
			Op:     "convert schema",
			Source: source,
			Err:    fmt.Errorf("%s must be an object", location),
		}
	}

	if ref, ok := rawMap["$ref"]; ok {
		refString := fmt.Sprint(ref)
		return &openapi3.SchemaRef{
			Ref: convertSwaggerRef(refString),
		}, nil
	}

	schema := &openapi3.Schema{
		Type:        schemaTypes(getString(rawMap, "type")),
		Format:      getString(rawMap, "format"),
		Description: getString(rawMap, "description"),
	}

	if enumValues, ok := rawMap["enum"].([]any); ok {
		schema.Enum = enumValues
	}

	if items, ok := rawMap["items"]; ok {
		itemsSchema, err := convertSchemaRef(source, location+".items", items)
		if err != nil {
			return nil, err
		}
		schema.Items = itemsSchema
	}

	if properties, ok := getMap(rawMap, "properties"); ok {
		schema.Properties = make(map[string]*openapi3.SchemaRef, len(properties))
		for name, property := range properties {
			propertySchema, err := convertSchemaRef(source, location+".properties."+name, property)
			if err != nil {
				return nil, err
			}
			schema.Properties[name] = propertySchema
		}
	}

	if required := stringSliceFromAny(rawMap["required"]); len(required) > 0 {
		schema.Required = required
	}

	return &openapi3.SchemaRef{Value: schema}, nil
}

func convertSwaggerRef(ref string) string {
	fragmentIndex := strings.Index(ref, "#")
	if fragmentIndex == -1 {
		return ref
	}

	prefix := ref[:fragmentIndex]
	fragment := ref[fragmentIndex:]

	replacements := map[string]string{
		"#/definitions/":         "#/components/schemas/",
		"#/parameters/":          "#/components/parameters/",
		"#/responses/":           "#/components/responses/",
		"#/securityDefinitions/": "#/components/securitySchemes/",
	}
	for oldPrefix, newPrefix := range replacements {
		if strings.HasPrefix(fragment, oldPrefix) {
			return prefix + strings.Replace(fragment, oldPrefix, newPrefix, 1)
		}
	}

	return ref
}

func schemaTypes(value string) *openapi3.Types {
	if value == "" {
		return nil
	}
	types := openapi3.Types{value}
	return &types
}
