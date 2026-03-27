package spec

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

	spec.Fingerprint = fingerprintForSpec(spec)

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

			normalized, err := normalizeOperation(pathName, item.method, item.operation, pathItem, resolved.openAPI3Doc.Servers, pathParameters, state)
			if err != nil {
				return nil, err
			}
			operations = append(operations, normalized)
		}
	}

	return operations, nil
}

func normalizeOperation(pathName, method string, operation *openapi3.Operation, pathItem *openapi3.PathItem, specServers openapi3.Servers, pathParameters []model.Parameter, state *normalizationState) (model.Operation, error) {
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

	allParameters := mergeParameters(pathParameters, parameters)

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
		DefaultServerURLs:   normalizeServerURLs(effectiveServers(operation, pathItem, specServers)),
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
		normalized, parameterWarnings := normalizeParameterModel(parameter.Name, in, parameter)
		result = append(result, normalized)
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
		normalized, headerWarnings := normalizeParameterModel(name, model.ParameterLocationHeader, &headerRef.Value.Parameter)
		result = append(result, normalized)
		warnings = append(warnings, headerWarnings...)
	}
	return result, warnings
}

func normalizeParameterModel(name string, in model.ParameterLocation, parameter *openapi3.Parameter) (model.Parameter, []model.SpecWarning) {
	if parameter == nil {
		return model.Parameter{}, nil
	}

	content, warnings := normalizeContent(parameter.Content)
	normalized := model.Parameter{
		Name:                name,
		In:                  in,
		Description:         parameter.Description,
		Required:            parameter.Required,
		Deprecated:          parameter.Deprecated,
		Style:               string(parameter.Style),
		Explode:             parameter.Explode,
		Schema:              normalizeSchema(parameter.Schema),
		Content:             content,
		SelectedContentType: defaultSelectedMediaType(content),
	}
	if len(content) == 0 {
		normalized.Example = parameter.Example
		normalized.Default = schemaDefault(parameter.Schema)
	}

	if collectionFormat, ok := swaggerCollectionFormatFromExtensions(parameter.Extensions); ok {
		normalized.CollectionFormat = collectionFormat
		warnings = append(warnings, model.SpecWarning{
			Code:    model.SpecWarningDowngradedFeature,
			Message: fmt.Sprintf("swagger collectionFormat %q for parameter %q was preserved separately because it is not losslessly representable via OpenAPI 3 style/explode", collectionFormat, name),
			Path:    name,
		})
	}

	return normalized, warnings
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
	return model.CapabilitySet{
		SupportsSwagger2Conversion: resolved.sourceFamily == model.SourceFamilySwagger2,
		SupportsOpenAPI3:           resolved.sourceFamily == model.SourceFamilyOpenAPI3,
		SupportsCookieParameters:   resolved.sourceFamily == model.SourceFamilyOpenAPI3,
		SupportsRequestBodies:      resolved.sourceFamily == model.SourceFamilyOpenAPI3 || resolved.sourceFamily == model.SourceFamilySwagger2,
		SupportsServerVariables:    resolved.sourceFamily == model.SourceFamilyOpenAPI3,
		SupportsSecuritySchemes:    resolved.sourceFamily == model.SourceFamilyOpenAPI3 || resolved.sourceFamily == model.SourceFamilySwagger2,
	}
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
	raw, ok := extensions[swaggerCollectionFormatExtension]
	if !ok || raw == nil {
		return "", false
	}

	value := strings.TrimSpace(fmt.Sprint(raw))
	if value == "" {
		return "", false
	}

	return value, true
}

func fingerprintForSpec(spec *model.APISpec) model.SpecFingerprint {
	payload := canonicalFingerprintSpec(spec)
	sum := sha256.Sum256(payload)
	return model.SpecFingerprint(hex.EncodeToString(sum[:]))
}

type fingerprintSpec struct {
	SourceFamily    model.SourceFamily              `json:"source_family"`
	SourceVersion   string                          `json:"source_version"`
	Title           string                          `json:"title"`
	Description     string                          `json:"description"`
	Servers         []fingerprintServer             `json:"servers"`
	Operations      []fingerprintOperation          `json:"operations"`
	SecuritySchemes []fingerprintSecurityScheme     `json:"security_schemes"`
	Security        *fingerprintSecurityRequirement `json:"security,omitempty"`
	Capabilities    model.CapabilitySet             `json:"capabilities"`
}

type fingerprintServer struct {
	URL         string                      `json:"url"`
	Description string                      `json:"description"`
	Variables   []fingerprintServerVariable `json:"variables,omitempty"`
}

type fingerprintServerVariable struct {
	Name        string   `json:"name"`
	Default     string   `json:"default"`
	Description string   `json:"description"`
	Enum        []string `json:"enum,omitempty"`
}

type fingerprintOperation struct {
	Key               model.OperationKey              `json:"key"`
	ID                string                          `json:"id"`
	Method            string                          `json:"method"`
	Path              string                          `json:"path"`
	Summary           string                          `json:"summary"`
	Description       string                          `json:"description"`
	Deprecated        bool                            `json:"deprecated"`
	Tags              []string                        `json:"tags,omitempty"`
	DefaultServerURLs []string                        `json:"default_server_urls,omitempty"`
	Parameters        []fingerprintParameter          `json:"parameters,omitempty"`
	RequestBody       *fingerprintRequestBody         `json:"request_body,omitempty"`
	Responses         []fingerprintResponse           `json:"responses,omitempty"`
	Security          *fingerprintSecurityRequirement `json:"security,omitempty"`
}

type fingerprintParameter struct {
	Name                string                  `json:"name"`
	In                  model.ParameterLocation `json:"in"`
	Description         string                  `json:"description"`
	Required            bool                    `json:"required"`
	Deprecated          bool                    `json:"deprecated"`
	Style               string                  `json:"style"`
	Explode             *bool                   `json:"explode,omitempty"`
	Schema              *fingerprintSchema      `json:"schema,omitempty"`
	Content             []fingerprintMediaType  `json:"content,omitempty"`
	SelectedContentType string                  `json:"selected_content_type,omitempty"`
	Example             any                     `json:"example,omitempty"`
	Default             any                     `json:"default,omitempty"`
	CollectionFormat    string                  `json:"collection_format,omitempty"`
}

type fingerprintRequestBody struct {
	Description string                 `json:"description"`
	Required    bool                   `json:"required"`
	Content     []fingerprintMediaType `json:"content,omitempty"`
}

type fingerprintResponse struct {
	StatusCode  string                 `json:"status_code"`
	Description string                 `json:"description"`
	Content     []fingerprintMediaType `json:"content,omitempty"`
	Headers     []fingerprintParameter `json:"headers,omitempty"`
}

type fingerprintMediaType struct {
	MediaType string                    `json:"media_type"`
	Schema    *fingerprintSchema        `json:"schema,omitempty"`
	Example   any                       `json:"example,omitempty"`
	Examples  []fingerprintNamedExample `json:"examples,omitempty"`
}

type fingerprintNamedExample struct {
	Name        string `json:"name"`
	Summary     string `json:"summary"`
	Description string `json:"description"`
	Value       any    `json:"value,omitempty"`
}

type fingerprintSecurityScheme struct {
	Name          string                   `json:"name"`
	Type          model.SecuritySchemeType `json:"type"`
	Description   string                   `json:"description"`
	In            model.ParameterLocation  `json:"in"`
	ParameterName string                   `json:"parameter_name"`
	Scheme        model.HTTPAuthScheme     `json:"scheme"`
	BearerFormat  string                   `json:"bearer_format"`
}

type fingerprintSecurityRequirement struct {
	Alternatives []fingerprintSecurityAlternative `json:"alternatives,omitempty"`
}

type fingerprintSecurityAlternative struct {
	Schemes []model.SecurityRequirementRef `json:"schemes,omitempty"`
}

type fingerprintSchema struct {
	Ref         string                   `json:"ref"`
	Type        string                   `json:"type"`
	Format      string                   `json:"format"`
	Title       string                   `json:"title"`
	Description string                   `json:"description"`
	Nullable    bool                     `json:"nullable"`
	Required    []string                 `json:"required,omitempty"`
	Enum        []any                    `json:"enum,omitempty"`
	Properties  []fingerprintNamedSchema `json:"properties,omitempty"`
	Items       *fingerprintSchema       `json:"items,omitempty"`
	OneOf       []*fingerprintSchema     `json:"one_of,omitempty"`
	AnyOf       []*fingerprintSchema     `json:"any_of,omitempty"`
	AllOf       []*fingerprintSchema     `json:"all_of,omitempty"`
}

type fingerprintNamedSchema struct {
	Name   string             `json:"name"`
	Schema *fingerprintSchema `json:"schema,omitempty"`
}

func canonicalFingerprintSpec(spec *model.APISpec) []byte {
	payload := fingerprintSpec{
		SourceFamily:    spec.SourceFamily,
		SourceVersion:   spec.SourceVersion,
		Title:           spec.Title,
		Description:     spec.Description,
		Servers:         canonicalServers(spec.Servers),
		Operations:      canonicalOperations(spec.Operations),
		SecuritySchemes: canonicalSecuritySchemes(spec.SecuritySchemes),
		Security:        canonicalSecurityRequirement(spec.Security),
		Capabilities:    spec.Capabilities,
	}

	bytes, err := json.Marshal(payload)
	if err != nil {
		panic(fmt.Sprintf("canonical fingerprint marshal failed: %v", err))
	}
	return bytes
}

func canonicalServers(servers []model.Server) []fingerprintServer {
	result := make([]fingerprintServer, 0, len(servers))
	for _, server := range servers {
		result = append(result, fingerprintServer{
			URL:         server.URL,
			Description: server.Description,
			Variables:   canonicalServerVariables(server.Variables),
		})
	}
	return result
}

func canonicalServerVariables(variables map[string]model.ServerVariable) []fingerprintServerVariable {
	if len(variables) == 0 {
		return nil
	}
	names := make([]string, 0, len(variables))
	for name := range variables {
		names = append(names, name)
	}
	sort.Strings(names)

	result := make([]fingerprintServerVariable, 0, len(names))
	for _, name := range names {
		variable := variables[name]
		result = append(result, fingerprintServerVariable{
			Name:        name,
			Default:     variable.Default,
			Description: variable.Description,
			Enum:        append([]string{}, variable.Enum...),
		})
	}
	return result
}

func canonicalOperations(operations []model.Operation) []fingerprintOperation {
	result := make([]fingerprintOperation, 0, len(operations))
	for _, operation := range operations {
		result = append(result, fingerprintOperation{
			Key:               operation.Key,
			ID:                operation.ID,
			Method:            operation.Method,
			Path:              operation.Path,
			Summary:           operation.Summary,
			Description:       operation.Description,
			Deprecated:        operation.Deprecated,
			Tags:              append([]string{}, operation.Tags...),
			DefaultServerURLs: append([]string{}, operation.DefaultServerURLs...),
			Parameters:        canonicalParameters(operation.Parameters),
			RequestBody:       canonicalRequestBody(operation.RequestBody),
			Responses:         canonicalResponses(operation.Responses),
			Security:          canonicalSecurityRequirement(operation.Security),
		})
	}
	return result
}

func canonicalParameters(parameters []model.Parameter) []fingerprintParameter {
	result := make([]fingerprintParameter, 0, len(parameters))
	for _, parameter := range parameters {
		result = append(result, fingerprintParameter{
			Name:                parameter.Name,
			In:                  parameter.In,
			Description:         parameter.Description,
			Required:            parameter.Required,
			Deprecated:          parameter.Deprecated,
			Style:               parameter.Style,
			Explode:             parameter.Explode,
			Schema:              canonicalSchema(parameter.Schema),
			Content:             canonicalMediaTypes(parameter.Content),
			SelectedContentType: parameter.SelectedContentType,
			Example:             parameter.Example,
			Default:             parameter.Default,
			CollectionFormat:    parameter.CollectionFormat,
		})
	}
	return result
}

func canonicalRequestBody(body *model.RequestBodySpec) *fingerprintRequestBody {
	if body == nil {
		return nil
	}
	return &fingerprintRequestBody{
		Description: body.Description,
		Required:    body.Required,
		Content:     canonicalMediaTypes(body.Content),
	}
}

func canonicalResponses(responses []model.ResponseSpec) []fingerprintResponse {
	result := make([]fingerprintResponse, 0, len(responses))
	for _, response := range responses {
		result = append(result, fingerprintResponse{
			StatusCode:  response.StatusCode,
			Description: response.Description,
			Content:     canonicalMediaTypes(response.Content),
			Headers:     canonicalParameters(response.Headers),
		})
	}
	return result
}

func canonicalMediaTypes(mediaTypes []model.MediaTypeSpec) []fingerprintMediaType {
	result := make([]fingerprintMediaType, 0, len(mediaTypes))
	for _, mediaType := range mediaTypes {
		result = append(result, fingerprintMediaType{
			MediaType: mediaType.MediaType,
			Schema:    canonicalSchema(mediaType.Schema),
			Example:   mediaType.Example,
			Examples:  canonicalExamples(mediaType.Examples),
		})
	}
	return result
}

func canonicalExamples(examples map[string]model.Example) []fingerprintNamedExample {
	if len(examples) == 0 {
		return nil
	}
	names := make([]string, 0, len(examples))
	for name := range examples {
		names = append(names, name)
	}
	sort.Strings(names)

	result := make([]fingerprintNamedExample, 0, len(names))
	for _, name := range names {
		example := examples[name]
		result = append(result, fingerprintNamedExample{
			Name:        name,
			Summary:     example.Summary,
			Description: example.Description,
			Value:       example.Value,
		})
	}
	return result
}

func canonicalSecuritySchemes(schemes map[string]model.SecurityScheme) []fingerprintSecurityScheme {
	if len(schemes) == 0 {
		return nil
	}
	names := make([]string, 0, len(schemes))
	for name := range schemes {
		names = append(names, name)
	}
	sort.Strings(names)

	result := make([]fingerprintSecurityScheme, 0, len(names))
	for _, name := range names {
		scheme := schemes[name]
		result = append(result, fingerprintSecurityScheme{
			Name:          name,
			Type:          scheme.Type,
			Description:   scheme.Description,
			In:            scheme.In,
			ParameterName: scheme.ParameterName,
			Scheme:        scheme.Scheme,
			BearerFormat:  scheme.BearerFormat,
		})
	}
	return result
}

func canonicalSecurityRequirement(requirement *model.SecurityRequirement) *fingerprintSecurityRequirement {
	if requirement == nil {
		return nil
	}

	result := &fingerprintSecurityRequirement{
		Alternatives: make([]fingerprintSecurityAlternative, 0, len(requirement.Alternatives)),
	}
	for _, alternative := range requirement.Alternatives {
		schemes := append([]model.SecurityRequirementRef{}, alternative.Schemes...)
		sort.Slice(schemes, func(i, j int) bool {
			if schemes[i].Name == schemes[j].Name {
				return strings.Join(schemes[i].Scopes, "\x00") < strings.Join(schemes[j].Scopes, "\x00")
			}
			return schemes[i].Name < schemes[j].Name
		})
		result.Alternatives = append(result.Alternatives, fingerprintSecurityAlternative{Schemes: schemes})
	}
	return result
}

func canonicalSchema(schema *model.Schema) *fingerprintSchema {
	if schema == nil {
		return nil
	}
	result := &fingerprintSchema{
		Ref:         schema.Ref,
		Type:        schema.Type,
		Format:      schema.Format,
		Title:       schema.Title,
		Description: schema.Description,
		Nullable:    schema.Nullable,
		Required:    append([]string{}, schema.Required...),
		Enum:        append([]any{}, schema.Enum...),
		Items:       canonicalSchema(schema.Items),
		OneOf:       canonicalSchemaList(schema.OneOf),
		AnyOf:       canonicalSchemaList(schema.AnyOf),
		AllOf:       canonicalSchemaList(schema.AllOf),
	}
	if len(schema.Properties) > 0 {
		names := make([]string, 0, len(schema.Properties))
		for name := range schema.Properties {
			names = append(names, name)
		}
		sort.Strings(names)
		result.Properties = make([]fingerprintNamedSchema, 0, len(names))
		for _, name := range names {
			result.Properties = append(result.Properties, fingerprintNamedSchema{
				Name:   name,
				Schema: canonicalSchema(schema.Properties[name]),
			})
		}
	}
	return result
}

func canonicalSchemaList(schemas []*model.Schema) []*fingerprintSchema {
	result := make([]*fingerprintSchema, 0, len(schemas))
	for _, schema := range schemas {
		result = append(result, canonicalSchema(schema))
	}
	return result
}
