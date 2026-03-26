package spec

import (
	"context"
	"fmt"
	"strings"

	"api-tui/internal/model"

	"github.com/getkin/kin-openapi/openapi3"
)

type resolvedDocument struct {
	document      *loadedDocument
	sourceFamily  model.SourceFamily
	sourceVersion string
	openAPI3Doc   *openapi3.T
}

func (l *loader) resolveDocument(ctx context.Context, converted *convertedDocument) (*resolvedDocument, error) {
	if err := rejectExternalRefs(converted.openAPI3Doc); err != nil {
		return nil, err
	}

	refLoader := openapi3.NewLoader()
	refLoader.IsExternalRefsAllowed = false
	refLoader.Context = ctx

	if err := refLoader.ResolveRefsIn(converted.openAPI3Doc, nil); err != nil {
		return nil, &Error{
			Kind:   ErrorKindRefResolutionFailure,
			Op:     "resolve refs",
			Source: converted.document.CanonicalLocation,
			Err:    err,
		}
	}

	return &resolvedDocument{
		document:      converted.document,
		sourceFamily:  converted.sourceFamily,
		sourceVersion: converted.sourceVersion,
		openAPI3Doc:   converted.openAPI3Doc,
	}, nil
}

func rejectExternalRefs(doc *openapi3.T) error {
	if doc == nil {
		return nil
	}

	if doc.Paths != nil {
		for pathName, pathItem := range doc.Paths.Map() {
			if pathItem == nil {
				continue
			}
			if err := rejectPathItemRefs(pathName, pathItem); err != nil {
				return err
			}
		}
	}

	if doc.Components == nil {
		return nil
	}

	for name, schema := range doc.Components.Schemas {
		if err := rejectRef(schema.Ref, "components.schemas."+name); err != nil {
			return err
		}
	}
	for name, parameter := range doc.Components.Parameters {
		if err := rejectRef(parameter.Ref, "components.parameters."+name); err != nil {
			return err
		}
	}
	for name, response := range doc.Components.Responses {
		if err := rejectRef(response.Ref, "components.responses."+name); err != nil {
			return err
		}
	}
	for name, requestBody := range doc.Components.RequestBodies {
		if err := rejectRef(requestBody.Ref, "components.requestBodies."+name); err != nil {
			return err
		}
	}
	for name, securityScheme := range doc.Components.SecuritySchemes {
		if err := rejectRef(securityScheme.Ref, "components.securitySchemes."+name); err != nil {
			return err
		}
	}

	return nil
}

func rejectPathItemRefs(pathName string, pathItem *openapi3.PathItem) error {
	if pathItem == nil {
		return nil
	}

	for index, parameter := range pathItem.Parameters {
		if err := rejectRef(parameter.Ref, fmt.Sprintf("paths.%s.parameters[%d]", pathName, index)); err != nil {
			return err
		}
	}

	operations := map[string]*openapi3.Operation{
		"get":     pathItem.Get,
		"put":     pathItem.Put,
		"post":    pathItem.Post,
		"delete":  pathItem.Delete,
		"options": pathItem.Options,
		"head":    pathItem.Head,
		"patch":   pathItem.Patch,
	}

	for method, operation := range operations {
		if err := rejectOperationRefs(pathName, method, operation); err != nil {
			return err
		}
	}

	return nil
}

func rejectOperationRefs(pathName, method string, operation *openapi3.Operation) error {
	if operation == nil {
		return nil
	}

	for index, parameter := range operation.Parameters {
		if err := rejectRef(parameter.Ref, fmt.Sprintf("paths.%s.%s.parameters[%d]", pathName, method, index)); err != nil {
			return err
		}
	}
	if operation.RequestBody != nil {
		if err := rejectRef(operation.RequestBody.Ref, fmt.Sprintf("paths.%s.%s.requestBody", pathName, method)); err != nil {
			return err
		}
		if operation.RequestBody.Value != nil {
			if err := rejectContentRefs(operation.RequestBody.Value.Content, fmt.Sprintf("paths.%s.%s.requestBody.content", pathName, method)); err != nil {
				return err
			}
		}
	}
	if operation.Responses != nil {
		for code, response := range operation.Responses.Map() {
			if err := rejectRef(response.Ref, fmt.Sprintf("paths.%s.%s.responses.%s", pathName, method, code)); err != nil {
				return err
			}
			if response != nil && response.Value != nil {
				if err := rejectContentRefs(response.Value.Content, fmt.Sprintf("paths.%s.%s.responses.%s.content", pathName, method, code)); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func rejectContentRefs(content openapi3.Content, location string) error {
	for mediaType, mediaTypeValue := range content {
		if mediaTypeValue == nil || mediaTypeValue.Schema == nil {
			continue
		}
		if err := rejectSchemaRefs(mediaTypeValue.Schema, location+"."+mediaType); err != nil {
			return err
		}
	}

	return nil
}

func rejectSchemaRefs(schema *openapi3.SchemaRef, location string) error {
	if schema == nil {
		return nil
	}
	if err := rejectRef(schema.Ref, location); err != nil {
		return err
	}
	if schema.Value == nil {
		return nil
	}
	if schema.Value.Items != nil {
		if err := rejectSchemaRefs(schema.Value.Items, location+".items"); err != nil {
			return err
		}
	}
	for name, property := range schema.Value.Properties {
		if err := rejectSchemaRefs(property, location+".properties."+name); err != nil {
			return err
		}
	}

	return nil
}

func rejectRef(ref, location string) error {
	if ref == "" || strings.HasPrefix(ref, "#/") {
		return nil
	}

	return &Error{
		Kind:   ErrorKindUnsupportedExternalRef,
		Op:     "resolve refs",
		Source: location,
		Err:    fmt.Errorf("external ref %q is not supported in m1.5", ref),
	}
}
