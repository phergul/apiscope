package converter

import (
	"fmt"
	"strings"

	"github.com/phergul/apiscope/internal/spec/internal/pipeline"

	"github.com/getkin/kin-openapi/openapi3"
)

func convertSwaggerParameters(source, location string, rawParameters []map[string]any) (openapi3.Parameters, error) {
	parameters := make(openapi3.Parameters, 0, len(rawParameters))
	for index, rawParameter := range rawParameters {
		parameter, err := convertSwaggerParameter(source, fmt.Sprintf("%s[%d]", location, index), rawParameter)
		if err != nil {
			return nil, err
		}
		parameters = append(parameters, parameter)
	}

	return parameters, nil
}

func convertSwaggerOperationInputs(source, pathName, method string, operationMap map[string]any, globalConsumes []string) (openapi3.Parameters, *openapi3.RequestBodyRef, error) {
	parameters := openapi3.Parameters{}
	var requestBody *openapi3.RequestBodyRef

	rawParameters := getSliceMap(operationMap, "parameters")
	for index, rawParameter := range rawParameters {
		location := fmt.Sprintf("%s %s parameter %d", strings.ToUpper(method), pathName, index)
		inValue := getString(rawParameter, "in")

		switch inValue {
		case "body":
			if requestBody != nil {
				return nil, nil, unsupportedSwaggerConstruct(source, location, "multiple body parameters are not supported")
			}
			body, err := convertSwaggerBodyParameter(source, location, rawParameter, consumesForOperation(operationMap, globalConsumes))
			if err != nil {
				return nil, nil, err
			}
			requestBody = body
		case "formData":
			return nil, nil, unsupportedSwaggerConstruct(source, location, "formData parameters are not supported")
		default:
			parameter, err := convertSwaggerParameter(source, location, rawParameter)
			if err != nil {
				return nil, nil, err
			}
			parameters = append(parameters, parameter)
		}
	}

	return parameters, requestBody, nil
}

func convertSwaggerBodyParameter(source, location string, rawParameter map[string]any, consumes []string) (*openapi3.RequestBodyRef, error) {
	if _, ok := rawParameter["schema"]; !ok {
		return nil, unsupportedSwaggerConstruct(source, location, "body parameters require a schema")
	}

	schema, err := convertSchemaRef(source, location+".schema", rawParameter["schema"])
	if err != nil {
		return nil, err
	}

	content := make(openapi3.Content)
	if len(consumes) == 0 {
		consumes = []string{"application/json"}
	}
	for _, mediaType := range consumes {
		content[mediaType] = &openapi3.MediaType{Schema: schema}
	}

	return &openapi3.RequestBodyRef{
		Value: &openapi3.RequestBody{
			Description: getString(rawParameter, "description"),
			Required:    getBool(rawParameter, "required"),
			Content:     content,
		},
	}, nil
}

func convertSwaggerParameter(source, location string, rawParameter map[string]any) (*openapi3.ParameterRef, error) {
	if ref, ok := rawParameter["$ref"]; ok {
		return &openapi3.ParameterRef{Ref: convertSwaggerRef(fmt.Sprint(ref))}, nil
	}

	inValue := getString(rawParameter, "in")
	switch inValue {
	case "path", "query", "header":
	default:
		return nil, unsupportedSwaggerConstruct(source, location, fmt.Sprintf("parameter location %q is not supported", inValue))
	}

	schema, err := convertSchemaFromParameter(source, location, rawParameter)
	if err != nil {
		return nil, err
	}

	parameter := &openapi3.Parameter{
		Name:        getString(rawParameter, "name"),
		In:          inValue,
		Description: getString(rawParameter, "description"),
		Required:    getBool(rawParameter, "required") || inValue == "path",
		Schema:      schema,
	}
	applySwaggerCollectionFormat(parameter, inValue, rawParameter)

	return &openapi3.ParameterRef{Value: parameter}, nil
}

func convertSwaggerParameterDefinitions(parsed *pipeline.ParsedDocument) (openapi3.ParametersMap, error) {
	rawDefinitions, ok := getMap(parsed.SwaggerDoc, "parameters")
	if !ok || len(rawDefinitions) == 0 {
		return nil, nil
	}

	parameters := make(openapi3.ParametersMap, len(rawDefinitions))
	for name, rawDefinition := range rawDefinitions {
		definitionMap, ok := rawDefinition.(map[string]any)
		if !ok {
			return nil, &pipeline.Error{
				Kind:   pipeline.ErrorKindSwaggerConversionFailure,
				Op:     "convert parameter definitions",
				Source: parsed.Document.CanonicalLocation,
				Err:    fmt.Errorf("parameter definition %q must be an object", name),
			}
		}

		parameter, err := convertSwaggerParameter(parsed.Document.CanonicalLocation, "parameters."+name, definitionMap)
		if err != nil {
			return nil, err
		}
		parameters[name] = parameter
	}

	return parameters, nil
}

func convertSwaggerHeaders(source, location string, rawHeaders map[string]any) (openapi3.Headers, error) {
	headers := make(openapi3.Headers, len(rawHeaders))
	for name, rawHeader := range rawHeaders {
		headerMap, ok := rawHeader.(map[string]any)
		if !ok {
			return nil, &pipeline.Error{
				Kind:   pipeline.ErrorKindSwaggerConversionFailure,
				Op:     "convert response headers",
				Source: source,
				Err:    fmt.Errorf("%s.%s must be an object", location, name),
			}
		}

		header, err := convertSwaggerHeader(source, location+"."+name, headerMap)
		if err != nil {
			return nil, err
		}
		headers[name] = header
	}

	return headers, nil
}

func convertSwaggerHeader(source, location string, rawHeader map[string]any) (*openapi3.HeaderRef, error) {
	if _, ok := rawHeader["$ref"]; ok {
		return nil, unsupportedSwaggerConstruct(source, location, "response header references are not supported")
	}

	schema, err := convertSchemaFromParameter(source, location, rawHeader)
	if err != nil {
		return nil, err
	}

	parameter := openapi3.Parameter{
		Description: getString(rawHeader, "description"),
		Required:    getBool(rawHeader, "required"),
		Schema:      schema,
	}
	applySwaggerCollectionFormat(&parameter, "header", rawHeader)

	return &openapi3.HeaderRef{
		Value: &openapi3.Header{
			Parameter: parameter,
		},
	}, nil
}

func consumesForOperation(operationMap map[string]any, global []string) []string {
	if local := getStringSlice(operationMap, "consumes"); len(local) > 0 {
		return local
	}
	return global
}

func applySwaggerCollectionFormat(parameter *openapi3.Parameter, inValue string, raw map[string]any) {
	if parameter == nil || !swaggerParameterHasArrayShape(raw) {
		return
	}

	collectionFormat := strings.ToLower(strings.TrimSpace(getString(raw, "collectionFormat")))
	switch collectionFormat {
	case "", "csv":
		switch inValue {
		case "query":
			parameter.Style = "form"
			parameter.Explode = boolPtr(false)
		case "path", "header":
			parameter.Style = "simple"
			parameter.Explode = boolPtr(false)
		}
	case "multi":
		if inValue == "query" {
			parameter.Style = "form"
			parameter.Explode = boolPtr(true)
			return
		}
		parameter.Extensions = withSwaggerCollectionFormat(parameter.Extensions, collectionFormat)
	case "ssv":
		if inValue == "query" {
			parameter.Style = "spaceDelimited"
			parameter.Explode = boolPtr(false)
			return
		}
		parameter.Extensions = withSwaggerCollectionFormat(parameter.Extensions, collectionFormat)
	case "pipes":
		if inValue == "query" {
			parameter.Style = "pipeDelimited"
			parameter.Explode = boolPtr(false)
			return
		}
		parameter.Extensions = withSwaggerCollectionFormat(parameter.Extensions, collectionFormat)
	case "tsv":
		parameter.Extensions = withSwaggerCollectionFormat(parameter.Extensions, collectionFormat)
	default:
		if collectionFormat != "" {
			parameter.Extensions = withSwaggerCollectionFormat(parameter.Extensions, collectionFormat)
		}
	}
}

func swaggerParameterHasArrayShape(raw map[string]any) bool {
	if raw == nil {
		return false
	}
	if getString(raw, "type") == "array" {
		return true
	}
	_, hasItems := raw["items"]
	return hasItems
}

func withSwaggerCollectionFormat(extensions map[string]any, collectionFormat string) map[string]any {
	if collectionFormat == "" {
		return extensions
	}
	if extensions == nil {
		extensions = make(map[string]any, 1)
	}
	extensions[pipeline.SwaggerCollectionFormatExtension] = collectionFormat
	return extensions
}
