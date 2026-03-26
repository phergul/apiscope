package spec

import (
	"fmt"
	"sort"
	"strings"

	"api-tui/internal/model"

	"github.com/getkin/kin-openapi/openapi3"
)

type normalizationState struct {
	warnings []model.SpecWarning
}

func (l *loader) normalizeDocument(resolved *resolvedDocument) (*model.APISpec, error) {
	state := &normalizationState{}

	servers := normalizeServers(resolved.openAPI3Doc.Servers)
	securitySchemes := normalizeSecuritySchemes(resolved.openAPI3Doc.Components, state)
	security := normalizeSecurityRequirements(resolved.openAPI3Doc.Security)

	operations, err := normalizeOperations(resolved, state)
	if err != nil {
		return nil, err
	}

	spec := &model.APISpec{
		Fingerprint:     fingerprintForDocument(resolved.document),
		Title:           resolved.openAPI3Doc.Info.Title,
		Summary:         "",
		Description:     resolved.openAPI3Doc.Info.Description,
		SourceFamily:    resolved.sourceFamily,
		SourceVersion:   resolved.sourceVersion,
		Capabilities:    deriveCapabilities(resolved),
		Warnings:        state.warnings,
		Servers:         servers,
		Operations:      operations,
		SecuritySchemes: securitySchemes,
		Security:        security,
	}

	return spec, nil
}

func normalizeOperations(resolved *resolvedDocument, state *normalizationState) ([]model.Operation, error) {
	if resolved.openAPI3Doc.Paths == nil {
		return nil, nil
	}

	pathNames := make([]string, 0, len(resolved.openAPI3Doc.Paths.Map()))
	for pathName := range resolved.openAPI3Doc.Paths.Map() {
		pathNames = append(pathNames, pathName)
	}
	sort.Strings(pathNames)

	operations := make([]model.Operation, 0)
	for _, pathName := range pathNames {
		pathItem := resolved.openAPI3Doc.Paths.Value(pathName)
		if pathItem == nil {
			continue
		}

		pathParameters, warnings := normalizeParameters(pathItem.Parameters)
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

			normalized, err := normalizeOperation(pathName, item.method, item.operation, pathParameters, resolved.openAPI3Doc.Servers, state)
			if err != nil {
				return nil, err
			}
			operations = append(operations, normalized)
		}
	}

	return operations, nil
}

func normalizeOperation(pathName, method string, operation *openapi3.Operation, pathParameters []model.Parameter, servers openapi3.Servers, state *normalizationState) (model.Operation, error) {
	if strings.TrimSpace(pathName) == "" {
		return model.Operation{}, &Error{
			Kind:   ErrorKindNormalizationFailure,
			Op:     "normalize operation",
			Source: method,
			Err:    fmt.Errorf("%s operation has empty path", method),
		}
	}

	parameters, warnings := normalizeParameters(operation.Parameters)
	state.warnings = append(state.warnings, warnings...)

	allParameters := append([]model.Parameter{}, pathParameters...)
	allParameters = append(allParameters, parameters...)

	requestBody, requestWarnings := normalizeRequestBody(operation.RequestBody)
	state.warnings = append(state.warnings, requestWarnings...)

	responses, responseWarnings := normalizeResponses(operation.Responses)
	state.warnings = append(state.warnings, responseWarnings...)

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
		Security:            normalizeSecurityRequirementsPtr(operation.Security),
		DefaultServerURLs:   normalizeServerURLs(servers),
		SelectedContentType: defaultSelectedContentType(requestBody),
	}, nil
}

func normalizeServers(servers openapi3.Servers) []model.Server {
	result := make([]model.Server, 0, len(servers))
	for _, server := range servers {
		if server == nil {
			continue
		}
		normalized := model.Server{
			URL:         server.URL,
			Description: server.Description,
		}
		if len(server.Variables) > 0 {
			normalized.Variables = make(map[string]model.ServerVariable, len(server.Variables))
			for name, variable := range server.Variables {
				normalized.Variables[name] = model.ServerVariable{
					Default:     variable.Default,
					Description: variable.Description,
					Enum:        append([]string{}, variable.Enum...),
				}
			}
		}
		result = append(result, normalized)
	}

	return result
}

func normalizeServerURLs(servers openapi3.Servers) []string {
	result := make([]string, 0, len(servers))
	for _, server := range servers {
		if server == nil {
			continue
		}
		result = append(result, server.URL)
	}
	return result
}

func normalizeParameters(parameters openapi3.Parameters) ([]model.Parameter, []model.SpecWarning) {
	result := make([]model.Parameter, 0, len(parameters))
	var warnings []model.SpecWarning

	for _, parameterRef := range parameters {
		if parameterRef == nil || parameterRef.Value == nil {
			continue
		}
		parameter := parameterRef.Value
		in, ok := normalizeParameterLocation(parameter.In)
		if !ok {
			warnings = append(warnings, model.SpecWarning{
				Code:    model.SpecWarningUnsupportedFeature,
				Message: fmt.Sprintf("parameter %q in %q was skipped because that location is not supported", parameter.Name, parameter.In),
				Path:    parameter.Name,
			})
			continue
		}
		result = append(result, model.Parameter{
			Name:        parameter.Name,
			In:          in,
			Description: parameter.Description,
			Required:    parameter.Required,
			Deprecated:  parameter.Deprecated,
			Style:       string(parameter.Style),
			Explode:     parameter.Explode,
			Schema:      normalizeSchema(parameter.Schema),
			Example:     parameter.Example,
			Default:     schemaDefault(parameter.Schema),
		})
	}

	return result, warnings
}

func normalizeParameterLocation(in string) (model.ParameterLocation, bool) {
	switch in {
	case "path":
		return model.ParameterLocationPath, true
	case "query":
		return model.ParameterLocationQuery, true
	case "header":
		return model.ParameterLocationHeader, true
	case "cookie":
		return model.ParameterLocationCookie, true
	default:
		return "", false
	}
}

func normalizeRequestBody(requestBody *openapi3.RequestBodyRef) (*model.RequestBodySpec, []model.SpecWarning) {
	if requestBody == nil || requestBody.Value == nil {
		return nil, nil
	}

	content, warnings := normalizeContent(requestBody.Value.Content)
	return &model.RequestBodySpec{
		Description: requestBody.Value.Description,
		Required:    requestBody.Value.Required,
		Content:     content,
	}, warnings
}

func normalizeResponses(responses *openapi3.Responses) ([]model.ResponseSpec, []model.SpecWarning) {
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
		content, contentWarnings := normalizeContent(responseRef.Value.Content)
		warnings = append(warnings, contentWarnings...)
		headers, headerWarnings := normalizeResponseHeaders(responseRef.Value.Headers)
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

func normalizeResponseHeaders(headers openapi3.Headers) ([]model.Parameter, []model.SpecWarning) {
	result := make([]model.Parameter, 0, len(headers))
	var warnings []model.SpecWarning
	for name, headerRef := range headers {
		if headerRef == nil || headerRef.Value == nil {
			continue
		}
		result = append(result, model.Parameter{
			Name:        name,
			In:          model.ParameterLocationHeader,
			Description: headerRef.Value.Description,
			Required:    headerRef.Value.Required,
			Deprecated:  headerRef.Value.Deprecated,
			Style:       string(headerRef.Value.Style),
			Explode:     headerRef.Value.Explode,
			Schema:      normalizeSchema(headerRef.Value.Schema),
			Example:     headerRef.Value.Example,
			Default:     schemaDefault(headerRef.Value.Schema),
		})
	}
	return result, warnings
}

func normalizeContent(content openapi3.Content) ([]model.MediaTypeSpec, []model.SpecWarning) {
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
			Schema:    normalizeSchema(mediaTypeValue.Schema),
			Example:   mediaTypeValue.Example,
			Examples:  normalizeExamples(mediaTypeValue.Examples),
		})
		if len(mediaTypeValue.Encoding) > 0 {
			warnings = append(warnings, model.SpecWarning{
				Code:    model.SpecWarningDowngradedFeature,
				Message: fmt.Sprintf("encoding details for media type %q were not preserved in the normalized model", mediaType),
				Path:    mediaType,
			})
		}
	}
	return result, warnings
}

func normalizeExamples(examples openapi3.Examples) map[string]model.Example {
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

func normalizeSchema(schemaRef *openapi3.SchemaRef) *model.Schema {
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
		schema.Items = normalizeSchema(value.Items)
	}
	if len(value.Properties) > 0 {
		schema.Properties = make(map[string]*model.Schema, len(value.Properties))
		for name, property := range value.Properties {
			schema.Properties[name] = normalizeSchema(property)
		}
	}
	if len(value.OneOf) > 0 {
		schema.OneOf = normalizeSchemaRefs(value.OneOf)
	}
	if len(value.AnyOf) > 0 {
		schema.AnyOf = normalizeSchemaRefs(value.AnyOf)
	}
	if len(value.AllOf) > 0 {
		schema.AllOf = normalizeSchemaRefs(value.AllOf)
	}

	return schema
}

func normalizeSchemaRefs(refs openapi3.SchemaRefs) []*model.Schema {
	result := make([]*model.Schema, 0, len(refs))
	for _, ref := range refs {
		result = append(result, normalizeSchema(ref))
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

func normalizeSecuritySchemes(components *openapi3.Components, state *normalizationState) map[string]model.SecurityScheme {
	if components == nil || len(components.SecuritySchemes) == 0 {
		return nil
	}

	result := make(map[string]model.SecurityScheme, len(components.SecuritySchemes))
	for name, schemeRef := range components.SecuritySchemes {
		if schemeRef == nil || schemeRef.Value == nil {
			continue
		}

		scheme := schemeRef.Value
		normalized := model.SecurityScheme{
			Name:         name,
			Description:  scheme.Description,
			BearerFormat: scheme.BearerFormat,
		}

		switch scheme.Type {
		case "apiKey":
			normalized.Type = model.SecuritySchemeTypeAPIKey
			if in, ok := normalizeParameterLocation(scheme.In); ok {
				normalized.In = in
			}
			normalized.ParameterName = scheme.Name
		case "http":
			normalized.Type = model.SecuritySchemeTypeHTTP
			switch strings.ToLower(scheme.Scheme) {
			case "basic":
				normalized.Scheme = model.HTTPAuthSchemeBasic
			case "bearer":
				normalized.Scheme = model.HTTPAuthSchemeBearer
			default:
				normalized.Scheme = model.HTTPAuthSchemeUnknown
				state.warnings = append(state.warnings, model.SpecWarning{
					Code:    model.SpecWarningUnsupportedFeature,
					Message: fmt.Sprintf("http security scheme %q uses unsupported scheme %q", name, scheme.Scheme),
					Path:    name,
				})
			}
		default:
			state.warnings = append(state.warnings, model.SpecWarning{
				Code:    model.SpecWarningUnsupportedFeature,
				Message: fmt.Sprintf("security scheme %q of type %q is not fully represented in the normalized model", name, scheme.Type),
				Path:    name,
			})
		}

		result[name] = normalized
	}

	return result
}

func normalizeSecurityRequirements(requirements openapi3.SecurityRequirements) *model.SecurityRequirement {
	if len(requirements) == 0 {
		return nil
	}

	alternatives := make([]model.SecurityAlternative, 0, len(requirements))
	for _, requirement := range requirements {
		names := make([]string, 0, len(requirement))
		for name := range requirement {
			names = append(names, name)
		}
		sort.Strings(names)

		alternative := model.SecurityAlternative{
			Schemes: make([]model.SecurityRequirementRef, 0, len(names)),
		}
		for _, name := range names {
			alternative.Schemes = append(alternative.Schemes, model.SecurityRequirementRef{
				Name:   name,
				Scopes: append([]string{}, requirement[name]...),
			})
		}
		alternatives = append(alternatives, alternative)
	}

	return &model.SecurityRequirement{Alternatives: alternatives}
}

func normalizeSecurityRequirementsPtr(requirements *openapi3.SecurityRequirements) *model.SecurityRequirement {
	if requirements == nil {
		return nil
	}
	return normalizeSecurityRequirements(*requirements)
}

func deriveCapabilities(resolved *resolvedDocument) model.CapabilitySet {
	hasCookieParameters := false
	hasServerVariables := false
	hasSecuritySchemes := false
	hasRequestBodies := false

	if resolved.openAPI3Doc.Components != nil && len(resolved.openAPI3Doc.Components.SecuritySchemes) > 0 {
		hasSecuritySchemes = true
	}
	for _, server := range resolved.openAPI3Doc.Servers {
		if server != nil && len(server.Variables) > 0 {
			hasServerVariables = true
		}
	}
	if resolved.openAPI3Doc.Paths != nil {
		for _, pathItem := range resolved.openAPI3Doc.Paths.Map() {
			if pathItem == nil {
				continue
			}
			for _, parameterRef := range pathItem.Parameters {
				if parameterRef != nil && parameterRef.Value != nil && parameterRef.Value.In == "cookie" {
					hasCookieParameters = true
				}
			}
			for _, operation := range []*openapi3.Operation{pathItem.Get, pathItem.Put, pathItem.Post, pathItem.Delete, pathItem.Options, pathItem.Head, pathItem.Patch} {
				if operation == nil {
					continue
				}
				if operation.RequestBody != nil {
					hasRequestBodies = true
				}
				for _, parameterRef := range operation.Parameters {
					if parameterRef != nil && parameterRef.Value != nil && parameterRef.Value.In == "cookie" {
						hasCookieParameters = true
					}
				}
			}
		}
	}

	return model.CapabilitySet{
		SupportsSwagger2Conversion: resolved.sourceFamily == model.SourceFamilySwagger2,
		SupportsOpenAPI3:           true,
		SupportsCookieParameters:   hasCookieParameters,
		SupportsRequestBodies:      hasRequestBodies,
		SupportsServerVariables:    hasServerVariables,
		SupportsSecuritySchemes:    hasSecuritySchemes,
	}
}

func defaultSelectedContentType(body *model.RequestBodySpec) string {
	if body == nil || len(body.Content) == 0 {
		return ""
	}
	return body.Content[0].MediaType
}
