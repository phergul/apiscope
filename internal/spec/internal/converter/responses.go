package converter

import (
	"fmt"
	"sort"
	"strings"

	"api-tui/internal/spec/internal/pipeline"

	"github.com/getkin/kin-openapi/openapi3"
)

func convertSwaggerResponses(source, pathName, method string, operationMap map[string]any, globalProduces []string) (*openapi3.Responses, error) {
	rawResponses, ok := getMap(operationMap, "responses")
	if !ok {
		return nil, unsupportedSwaggerConstruct(source, fmt.Sprintf("%s %s responses", strings.ToUpper(method), pathName), "responses are required")
	}

	produces := producesForOperation(operationMap, globalProduces)
	responses := openapi3.NewResponses()
	responses.Delete("default")

	keys := make([]string, 0, len(rawResponses))
	for key := range rawResponses {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, statusCode := range keys {
		responseMap, ok := rawResponses[statusCode].(map[string]any)
		if !ok {
			return nil, &pipeline.Error{
				Kind:   pipeline.ErrorKindSwaggerConversionFailure,
				Op:     "convert responses",
				Source: source,
				Err:    fmt.Errorf("response %q must be an object", statusCode),
			}
		}

		response, err := convertSwaggerResponse(source, fmt.Sprintf("%s %s response %s", strings.ToUpper(method), pathName, statusCode), responseMap, produces)
		if err != nil {
			return nil, err
		}

		if statusCode == "default" {
			responses.Set("default", response)
			continue
		}
		responses.Set(statusCode, response)
	}

	return responses, nil
}

func convertSwaggerResponse(source, location string, responseMap map[string]any, produces []string) (*openapi3.ResponseRef, error) {
	if ref, ok := responseMap["$ref"]; ok {
		return &openapi3.ResponseRef{Ref: convertSwaggerRef(fmt.Sprint(ref))}, nil
	}

	response := &openapi3.Response{
		Description: ptrString(getString(responseMap, "description")),
	}

	if headers, ok := getMap(responseMap, "headers"); ok && len(headers) > 0 {
		convertedHeaders, err := convertSwaggerHeaders(source, location+".headers", headers)
		if err != nil {
			return nil, err
		}
		response.Headers = convertedHeaders
	}

	if rawSchema, ok := responseMap["schema"]; ok {
		schema, err := convertSchemaRef(source, location+".schema", rawSchema)
		if err != nil {
			return nil, err
		}

		content := make(openapi3.Content)
		if len(produces) == 0 {
			produces = []string{"application/json"}
		}
		for _, mediaType := range produces {
			content[mediaType] = &openapi3.MediaType{Schema: schema}
		}
		response.Content = content
	}

	return &openapi3.ResponseRef{Value: response}, nil
}

func convertSwaggerResponseDefinitions(parsed *pipeline.ParsedDocument) (openapi3.ResponseBodies, error) {
	rawDefinitions, ok := getMap(parsed.SwaggerDoc, "responses")
	if !ok || len(rawDefinitions) == 0 {
		return nil, nil
	}

	responses := make(openapi3.ResponseBodies, len(rawDefinitions))
	globalProduces := getStringSlice(parsed.SwaggerDoc, "produces")
	for name, rawDefinition := range rawDefinitions {
		definitionMap, ok := rawDefinition.(map[string]any)
		if !ok {
			return nil, &pipeline.Error{
				Kind:   pipeline.ErrorKindSwaggerConversionFailure,
				Op:     "convert response definitions",
				Source: parsed.Document.CanonicalLocation,
				Err:    fmt.Errorf("response definition %q must be an object", name),
			}
		}

		response, err := convertSwaggerResponse(parsed.Document.CanonicalLocation, "responses."+name, definitionMap, globalProduces)
		if err != nil {
			return nil, err
		}
		responses[name] = response
	}

	return responses, nil
}

func producesForOperation(operationMap map[string]any, global []string) []string {
	if local := getStringSlice(operationMap, "produces"); len(local) > 0 {
		return local
	}
	return global
}
