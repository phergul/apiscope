package normalise

import (
	"api-tui/internal/model"

	"github.com/getkin/kin-openapi/openapi3"
)

func normaliseSchema(schemaRef *openapi3.SchemaRef) *model.Schema {
	if schemaRef == nil {
		return nil
	}

	schema := &model.Schema{
		Ref: schemaRef.Ref,
	}
	if schemaRef.Value == nil {
		return schema
	}

	value := schemaRef.Value
	schema.Type = firstSchemaType(value.Type)
	schema.Format = value.Format
	schema.Title = value.Title
	schema.Description = value.Description
	schema.Nullable = value.Nullable
	schema.Required = append([]string{}, value.Required...)
	schema.Enum = append([]any{}, value.Enum...)

	if value.Items != nil {
		schema.Items = normaliseSchema(value.Items)
	}
	if len(value.Properties) > 0 {
		schema.Properties = make(map[string]*model.Schema, len(value.Properties))
		for name, property := range value.Properties {
			schema.Properties[name] = normaliseSchema(property)
		}
	}
	if len(value.OneOf) > 0 {
		schema.OneOf = normaliseSchemaRefs(value.OneOf)
	}
	if len(value.AnyOf) > 0 {
		schema.AnyOf = normaliseSchemaRefs(value.AnyOf)
	}
	if len(value.AllOf) > 0 {
		schema.AllOf = normaliseSchemaRefs(value.AllOf)
	}

	return schema
}

func normaliseSchemaRefs(refs openapi3.SchemaRefs) []*model.Schema {
	result := make([]*model.Schema, 0, len(refs))
	for _, ref := range refs {
		result = append(result, normaliseSchema(ref))
	}
	return result
}

func firstSchemaType(types *openapi3.Types) string {
	if types == nil || len(*types) == 0 {
		return ""
	}
	return (*types)[0]
}

func schemaDefault(schemaRef *openapi3.SchemaRef) any {
	if schemaRef == nil || schemaRef.Value == nil {
		return nil
	}
	return schemaRef.Value.Default
}
