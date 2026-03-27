package spec

import (
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"api-tui/internal/model"

	"github.com/getkin/kin-openapi/openapi3"
)

const swaggerCollectionFormatExtension = "x-api-tui-swagger-collection-format"

type convertedDocument struct {
	document      *loadedDocument
	sourceFamily  model.SourceFamily
	sourceVersion string
	openAPI3Doc   *openapi3.T
}

func (l *loader) convertDocument(parsed *parsedDocument) (*convertedDocument, error) {
	switch parsed.sourceFamily {
	case model.SourceFamilyOpenAPI3:
		return &convertedDocument{
			document:      parsed.document,
			sourceFamily:  parsed.sourceFamily,
			sourceVersion: parsed.sourceVersion,
			openAPI3Doc:   parsed.openAPI3Doc,
		}, nil
	case model.SourceFamilySwagger2:
		doc, err := convertSwaggerDocument(parsed)
		if err != nil {
			return nil, err
		}
		return &convertedDocument{
			document:      parsed.document,
			sourceFamily:  parsed.sourceFamily,
			sourceVersion: parsed.sourceVersion,
			openAPI3Doc:   doc,
		}, nil
	default:
		return nil, &Error{
			Kind:   ErrorKindUnsupportedFamily,
			Op:     "convert document",
			Source: parsed.document.CanonicalLocation,
			Err:    fmt.Errorf("unexpected source family %q", parsed.sourceFamily),
		}
	}
}

func convertSwaggerDocument(parsed *parsedDocument) (*openapi3.T, error) {
	doc := &openapi3.T{
		OpenAPI: "3.0.3",
		Info:    convertSwaggerInfo(parsed),
		Paths:   &openapi3.Paths{},
	}

	servers, err := convertSwaggerServers(parsed.swaggerDoc)
	if err != nil {
		return nil, err
	}
	doc.Servers = servers

	securitySchemes, err := convertSwaggerSecuritySchemes(parsed)
	if err != nil {
		return nil, err
	}
	schemas, err := convertSwaggerDefinitions(parsed)
	if err != nil {
		return nil, err
	}
	parameters, err := convertSwaggerParameterDefinitions(parsed)
	if err != nil {
		return nil, err
	}
	responses, err := convertSwaggerResponseDefinitions(parsed)
	if err != nil {
		return nil, err
	}
	if securitySchemes != nil || len(schemas) > 0 || len(parameters) > 0 || len(responses) > 0 {
		components := openapi3.NewComponents()
		if securitySchemes != nil {
			components.SecuritySchemes = securitySchemes
		}
		if len(schemas) > 0 {
			components.Schemas = schemas
		}
		if len(parameters) > 0 {
			components.Parameters = parameters
		}
		if len(responses) > 0 {
			components.Responses = responses
		}
		doc.Components = &components
	}

	security, err := convertSecurityRequirementList(parsed.document.CanonicalLocation, "top-level security", getSliceMap(parsed.swaggerDoc, "security"))
	if err != nil {
		return nil, err
	}
	doc.Security = security

	globalConsumes := getStringSlice(parsed.swaggerDoc, "consumes")
	globalProduces := getStringSlice(parsed.swaggerDoc, "produces")

	paths, err := convertSwaggerPaths(parsed.document.CanonicalLocation, parsed.swaggerDoc, globalConsumes, globalProduces)
	if err != nil {
		return nil, err
	}
	doc.Paths = paths

	return doc, nil
}

func convertSwaggerInfo(parsed *parsedDocument) *openapi3.Info {
	infoMap, _ := getMap(parsed.swaggerDoc, "info")

	return &openapi3.Info{
		Title:          getString(infoMap, "title"),
		Description:    getString(infoMap, "description"),
		Version:        getString(infoMap, "version"),
		TermsOfService: getString(infoMap, "termsOfService"),
	}
}

func convertSwaggerServers(swaggerDoc map[string]any) (openapi3.Servers, error) {
	host := strings.TrimSpace(getString(swaggerDoc, "host"))
	basePath := strings.TrimSpace(getString(swaggerDoc, "basePath"))
	if basePath == "" {
		basePath = "/"
	}
	if !strings.HasPrefix(basePath, "/") {
		basePath = "/" + basePath
	}

	schemes := getStringSlice(swaggerDoc, "schemes")
	if len(schemes) == 0 {
		schemes = []string{"https"}
	}

	if host == "" {
		return openapi3.Servers{
			&openapi3.Server{URL: basePath},
		}, nil
	}

	servers := make(openapi3.Servers, 0, len(schemes))
	for _, scheme := range schemes {
		if scheme == "" {
			continue
		}
		serverURL := (&url.URL{
			Scheme: scheme,
			Host:   host,
			Path:   basePath,
		}).String()
		servers = append(servers, &openapi3.Server{URL: serverURL})
	}

	if len(servers) == 0 {
		return nil, &Error{
			Kind:   ErrorKindSwaggerConversionFailure,
			Op:     "convert servers",
			Source: getString(swaggerDoc, "host"),
			Err:    errors.New("swagger schemes did not produce any servers"),
		}
	}

	return servers, nil
}

func convertSwaggerPaths(source string, swaggerDoc map[string]any, globalConsumes, globalProduces []string) (*openapi3.Paths, error) {
	rawPaths, _ := getMap(swaggerDoc, "paths")
	converted := &openapi3.Paths{}

	for pathName, rawPathItem := range rawPaths {
		pathMap, ok := rawPathItem.(map[string]any)
		if !ok {
			return nil, &Error{
				Kind:   ErrorKindSwaggerConversionFailure,
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
			return nil, &Error{
				Kind:   ErrorKindSwaggerConversionFailure,
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

func convertSwaggerOperation(source, pathName, method string, operationMap map[string]any, globalConsumes, globalProduces []string) (*openapi3.Operation, error) {
	parameters, requestBody, err := convertSwaggerOperationInputs(source, pathName, method, operationMap, globalConsumes)
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
	}, nil
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
			return nil, &Error{
				Kind:   ErrorKindSwaggerConversionFailure,
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

func convertSwaggerParameterDefinitions(parsed *parsedDocument) (openapi3.ParametersMap, error) {
	rawDefinitions, ok := getMap(parsed.swaggerDoc, "parameters")
	if !ok || len(rawDefinitions) == 0 {
		return nil, nil
	}

	parameters := make(openapi3.ParametersMap, len(rawDefinitions))
	for name, rawDefinition := range rawDefinitions {
		definitionMap, ok := rawDefinition.(map[string]any)
		if !ok {
			return nil, &Error{
				Kind:   ErrorKindSwaggerConversionFailure,
				Op:     "convert parameter definitions",
				Source: parsed.document.CanonicalLocation,
				Err:    fmt.Errorf("parameter definition %q must be an object", name),
			}
		}

		parameter, err := convertSwaggerParameter(parsed.document.CanonicalLocation, "parameters."+name, definitionMap)
		if err != nil {
			return nil, err
		}
		parameters[name] = parameter
	}

	return parameters, nil
}

func convertSwaggerResponseDefinitions(parsed *parsedDocument) (openapi3.ResponseBodies, error) {
	rawDefinitions, ok := getMap(parsed.swaggerDoc, "responses")
	if !ok || len(rawDefinitions) == 0 {
		return nil, nil
	}

	responses := make(openapi3.ResponseBodies, len(rawDefinitions))
	globalProduces := getStringSlice(parsed.swaggerDoc, "produces")
	for name, rawDefinition := range rawDefinitions {
		definitionMap, ok := rawDefinition.(map[string]any)
		if !ok {
			return nil, &Error{
				Kind:   ErrorKindSwaggerConversionFailure,
				Op:     "convert response definitions",
				Source: parsed.document.CanonicalLocation,
				Err:    fmt.Errorf("response definition %q must be an object", name),
			}
		}

		response, err := convertSwaggerResponse(parsed.document.CanonicalLocation, "responses."+name, definitionMap, globalProduces)
		if err != nil {
			return nil, err
		}
		responses[name] = response
	}

	return responses, nil
}

func convertSwaggerSecuritySchemes(parsed *parsedDocument) (openapi3.SecuritySchemes, error) {
	rawDefinitions, ok := getMap(parsed.swaggerDoc, "securityDefinitions")
	if !ok || len(rawDefinitions) == 0 {
		return nil, nil
	}

	schemes := make(openapi3.SecuritySchemes, len(rawDefinitions))
	for name, rawDefinition := range rawDefinitions {
		definitionMap, ok := rawDefinition.(map[string]any)
		if !ok {
			return nil, &Error{
				Kind:   ErrorKindSwaggerConversionFailure,
				Op:     "convert security definitions",
				Source: parsed.document.CanonicalLocation,
				Err:    fmt.Errorf("security definition %q must be an object", name),
			}
		}

		scheme, err := convertSwaggerSecurityScheme(parsed.document.CanonicalLocation, name, definitionMap)
		if err != nil {
			return nil, err
		}
		schemes[name] = scheme
	}

	return schemes, nil
}

func convertSwaggerDefinitions(parsed *parsedDocument) (openapi3.Schemas, error) {
	rawDefinitions, ok := getMap(parsed.swaggerDoc, "definitions")
	if !ok || len(rawDefinitions) == 0 {
		return nil, nil
	}

	schemas := make(openapi3.Schemas, len(rawDefinitions))
	for name, rawDefinition := range rawDefinitions {
		schema, err := convertSchemaRef(parsed.document.CanonicalLocation, "definitions."+name, rawDefinition)
		if err != nil {
			return nil, err
		}
		schemas[name] = schema
	}

	return schemas, nil
}

func convertSwaggerHeaders(source, location string, rawHeaders map[string]any) (openapi3.Headers, error) {
	headers := make(openapi3.Headers, len(rawHeaders))
	for name, rawHeader := range rawHeaders {
		headerMap, ok := rawHeader.(map[string]any)
		if !ok {
			return nil, &Error{
				Kind:   ErrorKindSwaggerConversionFailure,
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

func convertSwaggerSecurityScheme(source, name string, definitionMap map[string]any) (*openapi3.SecuritySchemeRef, error) {
	schemeType := getString(definitionMap, "type")
	switch schemeType {
	case "apiKey":
		inValue := getString(definitionMap, "in")
		if inValue != "query" && inValue != "header" {
			return nil, unsupportedSwaggerConstruct(source, "securityDefinitions."+name, fmt.Sprintf("apiKey in=%q is not supported", inValue))
		}
		return &openapi3.SecuritySchemeRef{
			Value: &openapi3.SecurityScheme{
				Type:        "apiKey",
				Description: getString(definitionMap, "description"),
				Name:        getString(definitionMap, "name"),
				In:          inValue,
			},
		}, nil
	case "basic":
		return &openapi3.SecuritySchemeRef{
			Value: &openapi3.SecurityScheme{
				Type:        "http",
				Description: getString(definitionMap, "description"),
				Scheme:      "basic",
			},
		}, nil
	case "oauth2":
		flows, err := convertSwaggerOAuthFlows(source, name, definitionMap)
		if err != nil {
			return nil, err
		}
		return &openapi3.SecuritySchemeRef{
			Value: &openapi3.SecurityScheme{
				Type:        "oauth2",
				Description: getString(definitionMap, "description"),
				Flows:       flows,
			},
		}, nil
	default:
		return nil, unsupportedSwaggerConstruct(source, "securityDefinitions."+name, fmt.Sprintf("security definition type %q is not supported", schemeType))
	}
}

func convertSwaggerOAuthFlows(source, name string, definitionMap map[string]any) (*openapi3.OAuthFlows, error) {
	flowName := getString(definitionMap, "flow")
	flow := &openapi3.OAuthFlow{
		AuthorizationURL: getString(definitionMap, "authorizationUrl"),
		TokenURL:         getString(definitionMap, "tokenUrl"),
		Scopes:           stringMapFromAny(definitionMap["scopes"]),
	}

	flows := &openapi3.OAuthFlows{}
	switch flowName {
	case "implicit":
		flows.Implicit = flow
	case "password":
		flows.Password = flow
	case "application":
		flows.ClientCredentials = flow
	case "accessCode":
		flows.AuthorizationCode = flow
	default:
		return nil, unsupportedSwaggerConstruct(source, "securityDefinitions."+name, fmt.Sprintf("oauth2 flow %q is not supported", flowName))
	}

	return flows, nil
}

func convertSecurityRequirementList(source, location string, items []map[string]any) (openapi3.SecurityRequirements, error) {
	if len(items) == 0 {
		return nil, nil
	}

	requirements := make(openapi3.SecurityRequirements, 0, len(items))
	for _, item := range items {
		requirement := openapi3.SecurityRequirement{}
		for name, rawScopes := range item {
			requirement[name] = stringSliceFromAny(rawScopes)
		}
		requirements = append(requirements, requirement)
	}

	return requirements, nil
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
		return nil, &Error{
			Kind:   ErrorKindSwaggerConversionFailure,
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

func consumesForOperation(operationMap map[string]any, global []string) []string {
	if local := getStringSlice(operationMap, "consumes"); len(local) > 0 {
		return local
	}
	return global
}

func producesForOperation(operationMap map[string]any, global []string) []string {
	if local := getStringSlice(operationMap, "produces"); len(local) > 0 {
		return local
	}
	return global
}

func unsupportedSwaggerConstruct(source, location, message string) error {
	return &Error{
		Kind:   ErrorKindUnsupportedSwaggerConstruct,
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
	extensions[swaggerCollectionFormatExtension] = collectionFormat
	return extensions
}

func securityRequirementPtr(requirements openapi3.SecurityRequirements) *openapi3.SecurityRequirements {
	if len(requirements) == 0 {
		return nil
	}

	return &requirements
}

func schemaTypes(value string) *openapi3.Types {
	if value == "" {
		return nil
	}
	types := openapi3.Types{value}
	return &types
}
