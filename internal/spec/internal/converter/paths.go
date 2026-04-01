package converter

import (
	"fmt"
	"strings"

	"github.com/phergul/apiscope/internal/spec/internal/pipeline"

	"github.com/getkin/kin-openapi/openapi3"
)

func convertSwaggerPaths(source string, swaggerDoc map[string]any, globalConsumes, globalProduces []string) (*openapi3.Paths, error) {
	rawPaths, _ := getMap(swaggerDoc, "paths")
	converted := &openapi3.Paths{}

	for pathName, rawPathItem := range rawPaths {
		pathMap, ok := rawPathItem.(map[string]any)
		if !ok {
			return nil, &pipeline.Error{
				Kind:   pipeline.ErrorKindSwaggerConversionFailure,
				Op:     "convert paths",
				Source: source,
				Err:    fmt.Errorf("path %q must be an object", pathName),
			}
		}

		pathItem, err := convertSwaggerPathItem(source, pathName, pathMap, globalConsumes, globalProduces)
		if err != nil {
			return nil, err
		}

		converted.Set(pathName, pathItem)
	}

	return converted, nil
}

func convertSwaggerPathItem(source, pathName string, pathMap map[string]any, globalConsumes, globalProduces []string) (*openapi3.PathItem, error) {
	if ref, ok := pathMap["$ref"]; ok {
		return &openapi3.PathItem{
			Ref: convertSwaggerRef(fmt.Sprint(ref)),
		}, nil
	}

	pathParameters, err := convertSwaggerParameters(source, fmt.Sprintf("paths.%s.parameters", pathName), getSliceMap(pathMap, "parameters"))
	if err != nil {
		return nil, err
	}

	item := &openapi3.PathItem{
		Parameters: pathParameters,
	}

	for _, method := range []string{"get", "put", "post", "delete", "options", "head", "patch"} {
		rawOperation, ok := pathMap[method]
		if !ok {
			continue
		}

		operationMap, ok := rawOperation.(map[string]any)
		if !ok {
			return nil, &pipeline.Error{
				Kind:   pipeline.ErrorKindSwaggerConversionFailure,
				Op:     "convert operation",
				Source: source,
				Err:    fmt.Errorf("%s %s must be an object", strings.ToUpper(method), pathName),
			}
		}

		operation, err := convertSwaggerOperation(source, pathName, method, operationMap, globalConsumes, globalProduces)
		if err != nil {
			return nil, err
		}

		switch method {
		case "get":
			item.Get = operation
		case "put":
			item.Put = operation
		case "post":
			item.Post = operation
		case "delete":
			item.Delete = operation
		case "options":
			item.Options = operation
		case "head":
			item.Head = operation
		case "patch":
			item.Patch = operation
		}
	}

	return item, nil
}

func convertSwaggerOperation(source, pathName, method string, operationMap map[string]any, globalConsumes, globalProduces []string) (*openapi3.Operation, error) {
	parameters, requestBody, extensions, err := convertSwaggerOperationInputs(source, pathName, method, operationMap, globalConsumes)
	if err != nil {
		return nil, err
	}

	responses, err := convertSwaggerResponses(source, pathName, method, operationMap, globalProduces)
	if err != nil {
		return nil, err
	}

	security, err := convertSecurityRequirementList(source, fmt.Sprintf("%s %s security", strings.ToUpper(method), pathName), getSliceMap(operationMap, "security"))
	if err != nil {
		return nil, err
	}

	return &openapi3.Operation{
		OperationID: getString(operationMap, "operationId"),
		Summary:     getString(operationMap, "summary"),
		Description: getString(operationMap, "description"),
		Deprecated:  getBool(operationMap, "deprecated"),
		Tags:        getStringSlice(operationMap, "tags"),
		Parameters:  parameters,
		RequestBody: requestBody,
		Responses:   responses,
		Security:    securityRequirementPtr(security),
		Extensions:  extensions,
	}, nil
}
