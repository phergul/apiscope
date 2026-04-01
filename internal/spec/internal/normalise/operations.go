package normalise

import (
	"fmt"
	"sort"
	"strings"

	"github.com/phergul/apiscope/internal/model"
	"github.com/phergul/apiscope/internal/spec/internal/pipeline"

	"github.com/getkin/kin-openapi/openapi3"
)

func normaliseOperations(resolved *pipeline.ResolvedDocument, state *normalisationState) ([]model.Operation, error) {
	if resolved.OpenAPI3Doc.Paths == nil {
		return nil, nil
	}

	pathNames := make([]string, 0, len(resolved.OpenAPI3Doc.Paths.Map()))
	for pathName := range resolved.OpenAPI3Doc.Paths.Map() {
		pathNames = append(pathNames, pathName)
	}
	sort.Strings(pathNames)

	operations := make([]model.Operation, 0)
	for _, pathName := range pathNames {
		pathItem := resolved.OpenAPI3Doc.Paths.Value(pathName)
		if pathItem == nil {
			continue
		}

		pathParameters, warnings := normaliseParameters(pathItem.Parameters)
		state.warnings = append(state.warnings, warnings...)

		for _, item := range []struct {
			method    string
			operation *openapi3.Operation
		}{
			{"GET", pathItem.Get},
			{"PUT", pathItem.Put},
			{"POST", pathItem.Post},
			{"DELETE", pathItem.Delete},
			{"OPTIONS", pathItem.Options},
			{"HEAD", pathItem.Head},
			{"PATCH", pathItem.Patch},
		} {
			if item.operation == nil {
				continue
			}

			normalised, err := normaliseOperation(pathName, item.method, item.operation, pathItem, resolved.OpenAPI3Doc.Servers, pathParameters, state)
			if err != nil {
				return nil, err
			}
			operations = append(operations, normalised)
		}
	}

	return operations, nil
}

func normaliseOperation(pathName, method string, operation *openapi3.Operation, pathItem *openapi3.PathItem, specServers openapi3.Servers, pathParameters []model.Parameter, state *normalisationState) (model.Operation, error) {
	if strings.TrimSpace(pathName) == "" {
		return model.Operation{}, &pipeline.Error{
			Kind:   pipeline.ErrorKindNormalisationFailure,
			Op:     "normalise operation",
			Source: method,
			Err:    fmt.Errorf("%s operation has empty path", method),
		}
	}

	parameters, warnings := normaliseParameters(operation.Parameters)
	state.warnings = append(state.warnings, warnings...)

	allParameters := mergeParameters(pathParameters, parameters)

	requestBody, requestWarnings := normaliseRequestBody(operation.RequestBody)
	state.warnings = append(state.warnings, requestWarnings...)

	responses, responseWarnings := normaliseResponses(operation.Responses)
	state.warnings = append(state.warnings, responseWarnings...)

	formBodyMediaType, hasFormBodyMediaType := swaggerFormBodyMediaTypeFromExtensions(operation.Extensions)
	if swaggerAssumedFormEncodingFromExtensions(operation.Extensions) {
		state.warnings = append(state.warnings, model.SpecWarning{
			Code:    model.SpecWarningAmbiguousBehavior,
			Message: fmt.Sprintf("swagger formData operation %s %s did not declare consumes; assumed application/x-www-form-urlencoded", method, pathName),
			Path:    pathName,
		})
	}

	return model.Operation{
		Key:                 model.NewOperationKey(method, pathName),
		ID:                  operation.OperationID,
		Method:              method,
		Path:                pathName,
		Summary:             operation.Summary,
		Description:         operation.Description,
		Tags:                append([]string{}, operation.Tags...),
		Deprecated:          operation.Deprecated,
		Parameters:          allParameters,
		RequestBody:         requestBody,
		Responses:           responses,
		Security:            normaliseSecurityRequirementsPtr(operation.Security),
		DefaultServerURLs:   normaliseServerURLs(effectiveServers(operation, pathItem, specServers)),
		FormBodyMediaType:   conditionalString(hasFormBodyMediaType, formBodyMediaType),
		SelectedContentType: defaultSelectedContentType(requestBody),
	}, nil
}

func effectiveServers(operation *openapi3.Operation, pathItem *openapi3.PathItem, specServers openapi3.Servers) openapi3.Servers {
	if operation != nil && operation.Servers != nil && len(*operation.Servers) > 0 {
		return *operation.Servers
	}
	if pathItem != nil && len(pathItem.Servers) > 0 {
		return pathItem.Servers
	}
	return specServers
}

func normaliseServers(servers openapi3.Servers) []model.Server {
	result := make([]model.Server, 0, len(servers))
	for _, server := range servers {
		if server == nil {
			continue
		}
		normalised := model.Server{
			URL:         server.URL,
			Description: server.Description,
		}
		if len(server.Variables) > 0 {
			normalised.Variables = make(map[string]model.ServerVariable, len(server.Variables))
			for name, variable := range server.Variables {
				normalised.Variables[name] = model.ServerVariable{
					Default:     variable.Default,
					Description: variable.Description,
					Enum:        append([]string{}, variable.Enum...),
				}
			}
		}
		result = append(result, normalised)
	}

	return result
}

func normaliseServerURLs(servers openapi3.Servers) []string {
	result := make([]string, 0, len(servers))
	for _, server := range servers {
		if server == nil {
			continue
		}
		result = append(result, server.URL)
	}
	return result
}

func normaliseParameters(parameters openapi3.Parameters) ([]model.Parameter, []model.SpecWarning) {
	result := make([]model.Parameter, 0, len(parameters))
	var warnings []model.SpecWarning

	for _, parameterRef := range parameters {
		if parameterRef == nil || parameterRef.Value == nil {
			continue
		}
		parameter := parameterRef.Value
		in, ok := normaliseParameterLocation(parameter.In, parameter.Extensions)
		if !ok {
			warnings = append(warnings, model.SpecWarning{
				Code:    model.SpecWarningUnsupportedFeature,
				Message: fmt.Sprintf("parameter %q in %q was skipped because that location is not supported", parameter.Name, parameter.In),
				Path:    parameter.Name,
			})
			continue
		}
		normalised, parameterWarnings := normaliseParameterModel(parameter.Name, in, parameter)
		result = append(result, normalised)
		warnings = append(warnings, parameterWarnings...)
	}

	return result, warnings
}

func mergeParameters(pathParameters, operationParameters []model.Parameter) []model.Parameter {
	result := make([]model.Parameter, 0, len(pathParameters)+len(operationParameters))
	indexByKey := make(map[string]int, len(pathParameters))

	for _, parameter := range pathParameters {
		key := parameterIdentity(parameter)
		indexByKey[key] = len(result)
		result = append(result, parameter)
	}

	for _, parameter := range operationParameters {
		key := parameterIdentity(parameter)
		if idx, ok := indexByKey[key]; ok {
			result[idx] = parameter
			continue
		}
		indexByKey[key] = len(result)
		result = append(result, parameter)
	}

	return result
}

func parameterIdentity(parameter model.Parameter) string {
	return string(parameter.In) + "\x00" + parameter.Name
}

func normaliseParameterLocation(in string, extensions map[string]any) (model.ParameterLocation, bool) {
	if location, ok := swaggerParameterLocationFromExtensions(extensions); ok {
		return location, true
	}

	switch in {
	case "path":
		return model.ParameterLocationPath, true
	case "query":
		return model.ParameterLocationQuery, true
	case "header":
		return model.ParameterLocationHeader, true
	case "cookie":
		return model.ParameterLocationCookie, true
	case "formData":
		return model.ParameterLocationForm, true
	default:
		return "", false
	}
}

func normaliseRequestBody(requestBody *openapi3.RequestBodyRef) (*model.RequestBodySpec, []model.SpecWarning) {
	if requestBody == nil || requestBody.Value == nil {
		return nil, nil
	}

	content, warnings := normaliseContent(requestBody.Value.Content)
	return &model.RequestBodySpec{
		Description: requestBody.Value.Description,
		Required:    requestBody.Value.Required,
		Content:     content,
	}, warnings
}

func normaliseResponses(responses *openapi3.Responses) ([]model.ResponseSpec, []model.SpecWarning) {
	if responses == nil {
		return nil, nil
	}

	var warnings []model.SpecWarning
	keys := make([]string, 0, len(responses.Map()))
	for code := range responses.Map() {
		keys = append(keys, code)
	}
	sort.Strings(keys)

	result := make([]model.ResponseSpec, 0, len(keys))
	for _, code := range keys {
		responseRef := responses.Value(code)
		if responseRef == nil || responseRef.Value == nil {
			continue
		}
		content, contentWarnings := normaliseContent(responseRef.Value.Content)
		warnings = append(warnings, contentWarnings...)
		headers, headerWarnings := normaliseResponseHeaders(responseRef.Value.Headers)
		warnings = append(warnings, headerWarnings...)
		description := ""
		if responseRef.Value.Description != nil {
			description = *responseRef.Value.Description
		}
		result = append(result, model.ResponseSpec{
			StatusCode:  code,
			Description: description,
			Content:     content,
			Headers:     headers,
		})
	}

	return result, warnings
}

func normaliseResponseHeaders(headers openapi3.Headers) ([]model.Parameter, []model.SpecWarning) {
	names := make([]string, 0, len(headers))
	for name := range headers {
		names = append(names, name)
	}
	sort.Strings(names)

	result := make([]model.Parameter, 0, len(names))
	var warnings []model.SpecWarning
	for _, name := range names {
		headerRef := headers[name]
		if headerRef == nil || headerRef.Value == nil {
			continue
		}
		normalised, headerWarnings := normaliseParameterModel(name, model.ParameterLocationHeader, &headerRef.Value.Parameter)
		result = append(result, normalised)
		warnings = append(warnings, headerWarnings...)
	}
	return result, warnings
}

func normaliseParameterModel(name string, in model.ParameterLocation, parameter *openapi3.Parameter) (model.Parameter, []model.SpecWarning) {
	if parameter == nil {
		return model.Parameter{}, nil
	}

	content, warnings := normaliseContent(parameter.Content)
	normalised := model.Parameter{
		Name:                name,
		In:                  in,
		FormInputKind:       swaggerFormInputKindFromExtensions(parameter.Extensions),
		Description:         parameter.Description,
		Required:            parameter.Required,
		Deprecated:          parameter.Deprecated,
		Style:               string(parameter.Style),
		Explode:             parameter.Explode,
		Schema:              normaliseSchema(parameter.Schema),
		Content:             content,
		SelectedContentType: defaultSelectedMediaType(content),
	}
	if len(content) == 0 {
		normalised.Example = parameter.Example
		normalised.Default = schemaDefault(parameter.Schema)
	}

	if collectionFormat, ok := swaggerCollectionFormatFromExtensions(parameter.Extensions); ok {
		normalised.CollectionFormat = collectionFormat
		warnings = append(warnings, model.SpecWarning{
			Code:    model.SpecWarningDowngradedFeature,
			Message: fmt.Sprintf("swagger collectionFormat %q for parameter %q was preserved separately because it is not losslessly representable via OpenAPI 3 style/explode", collectionFormat, name),
			Path:    name,
		})
	}

	return normalised, warnings
}

func normaliseContent(content openapi3.Content) ([]model.MediaTypeSpec, []model.SpecWarning) {
	if len(content) == 0 {
		return nil, nil
	}

	keys := make([]string, 0, len(content))
	for mediaType := range content {
		keys = append(keys, mediaType)
	}
	sort.Strings(keys)

	result := make([]model.MediaTypeSpec, 0, len(keys))
	var warnings []model.SpecWarning
	for _, mediaType := range keys {
		mediaTypeValue := content[mediaType]
		if mediaTypeValue == nil {
			continue
		}
		result = append(result, model.MediaTypeSpec{
			MediaType: mediaType,
			Schema:    normaliseSchema(mediaTypeValue.Schema),
			Example:   mediaTypeValue.Example,
			Examples:  normaliseExamples(mediaTypeValue.Examples),
		})
		if len(mediaTypeValue.Encoding) > 0 {
			warnings = append(warnings, model.SpecWarning{
				Code:    model.SpecWarningDowngradedFeature,
				Message: fmt.Sprintf("encoding details for media type %q were not preserved in the normalised model", mediaType),
				Path:    mediaType,
			})
		}
	}
	return result, warnings
}

func normaliseExamples(examples openapi3.Examples) map[string]model.Example {
	if len(examples) == 0 {
		return nil
	}

	result := make(map[string]model.Example, len(examples))
	for name, exampleRef := range examples {
		if exampleRef == nil || exampleRef.Value == nil {
			continue
		}
		result[name] = model.Example{
			Summary:     exampleRef.Value.Summary,
			Description: exampleRef.Value.Description,
			Value:       exampleRef.Value.Value,
		}
	}
	return result
}

func defaultSelectedContentType(body *model.RequestBodySpec) string {
	if body == nil || len(body.Content) == 0 {
		return ""
	}
	return defaultSelectedMediaType(body.Content)
}

func defaultSelectedMediaType(content []model.MediaTypeSpec) string {
	if len(content) == 0 {
		return ""
	}
	return content[0].MediaType
}

func swaggerCollectionFormatFromExtensions(extensions map[string]any) (string, bool) {
	if len(extensions) == 0 {
		return "", false
	}
	raw, ok := extensions[pipeline.SwaggerCollectionFormatExtension]
	if !ok || raw == nil {
		return "", false
	}

	value := strings.TrimSpace(fmt.Sprint(raw))
	if value == "" {
		return "", false
	}

	return value, true
}

func swaggerParameterLocationFromExtensions(extensions map[string]any) (model.ParameterLocation, bool) {
	if len(extensions) == 0 {
		return "", false
	}
	raw, ok := extensions[pipeline.SwaggerParameterLocationExtension]
	if !ok || raw == nil {
		return "", false
	}

	switch strings.ToLower(strings.TrimSpace(fmt.Sprint(raw))) {
	case "formdata":
		return model.ParameterLocationForm, true
	default:
		return "", false
	}
}

func swaggerFormBodyMediaTypeFromExtensions(extensions map[string]any) (string, bool) {
	if len(extensions) == 0 {
		return "", false
	}
	raw, ok := extensions[pipeline.SwaggerFormBodyMediaTypeExtension]
	if !ok || raw == nil {
		return "", false
	}

	value := strings.TrimSpace(fmt.Sprint(raw))
	if value == "" {
		return "", false
	}

	return value, true
}

func swaggerFormInputKindFromExtensions(extensions map[string]any) model.FormInputKind {
	if len(extensions) == 0 {
		return model.FormInputKindValue
	}
	raw, ok := extensions[pipeline.SwaggerFormFileParameterExtension]
	if !ok || raw == nil {
		return model.FormInputKindValue
	}

	file, ok := raw.(bool)
	if ok && file {
		return model.FormInputKindFile
	}

	return model.FormInputKindValue
}

func swaggerAssumedFormEncodingFromExtensions(extensions map[string]any) bool {
	if len(extensions) == 0 {
		return false
	}
	raw, ok := extensions[pipeline.SwaggerAssumedFormEncodingExtension]
	if !ok || raw == nil {
		return false
	}

	assumed, ok := raw.(bool)
	return ok && assumed
}

func conditionalString(ok bool, value string) string {
	if !ok {
		return ""
	}

	return value
}
