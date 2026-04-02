package normalise

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/phergul/apiscope/internal/model"
)

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
	FormBodyMediaType string                          `json:"form_body_media_type,omitempty"`
	Parameters        []fingerprintParameter          `json:"parameters,omitempty"`
	RequestBody       *fingerprintRequestBody         `json:"request_body,omitempty"`
	Responses         []fingerprintResponse           `json:"responses,omitempty"`
	Security          *fingerprintSecurityRequirement `json:"security,omitempty"`
}

type fingerprintParameter struct {
	Name                string                  `json:"name"`
	In                  model.ParameterLocation `json:"in"`
	FormInputKind       model.FormInputKind     `json:"form_input_kind,omitempty"`
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
	Example     any                      `json:"example,omitempty"`
	Default     any                      `json:"default,omitempty"`
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
			FormBodyMediaType: operation.FormBodyMediaType,
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
			FormInputKind:       parameter.FormInputKind,
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
		Example:     schema.Example,
		Default:     schema.Default,
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
