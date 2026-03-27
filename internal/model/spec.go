package model

import (
	"strings"
)

type SourceFamily string

const (
	SourceFamilyUnknown  SourceFamily = "unknown"
	SourceFamilySwagger2 SourceFamily = "swagger2"
	SourceFamilyOpenAPI3 SourceFamily = "openapi3"
)

type ParameterLocation string

const (
	ParameterLocationPath   ParameterLocation = "path"
	ParameterLocationQuery  ParameterLocation = "query"
	ParameterLocationHeader ParameterLocation = "header"
	ParameterLocationCookie ParameterLocation = "cookie"
)

type SecuritySchemeType string

const (
	SecuritySchemeTypeAPIKey SecuritySchemeType = "apiKey"
	SecuritySchemeTypeHTTP   SecuritySchemeType = "http"
)

type HTTPAuthScheme string

const (
	HTTPAuthSchemeUnknown HTTPAuthScheme = ""
	HTTPAuthSchemeBasic   HTTPAuthScheme = "basic"
	HTTPAuthSchemeBearer  HTTPAuthScheme = "bearer"
)

type SpecWarningCode string

const (
	SpecWarningUnsupportedFeature SpecWarningCode = "unsupported_feature"
	SpecWarningDowngradedFeature  SpecWarningCode = "downgraded_feature"
	SpecWarningAmbiguousBehavior  SpecWarningCode = "ambiguous_behavior"
)

type SpecFingerprint string

type OperationKey string

func NewOperationKey(method, normalizedPath string) OperationKey {
	normalizedMethod := strings.ToUpper(strings.TrimSpace(method))
	path := normalizeOperationPath(normalizedPath)

	return OperationKey(normalizedMethod + " " + path)
}

func (k OperationKey) String() string {
	return string(k)
}

func normalizeOperationPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return path
}

type APISpec struct {
	Fingerprint     SpecFingerprint
	Title           string
	Summary         string
	Description     string
	SourceFamily    SourceFamily
	SourceVersion   string
	Capabilities    CapabilitySet
	Warnings        []SpecWarning
	Servers         []Server
	Operations      []Operation
	SecuritySchemes map[string]SecurityScheme
	Security        *SecurityRequirement
}

type Server struct {
	URL         string
	Description string
	Variables   map[string]ServerVariable
}

type ServerVariable struct {
	Default     string
	Description string
	Enum        []string
}

type Operation struct {
	Key                 OperationKey
	ID                  string
	Method              string
	Path                string
	Summary             string
	Description         string
	Tags                []string
	Deprecated          bool
	Parameters          []Parameter
	RequestBody         *RequestBodySpec
	Responses           []ResponseSpec
	Security            *SecurityRequirement
	DefaultServerURLs   []string
	SelectedContentType string
}

type Parameter struct {
	Name                string
	In                  ParameterLocation
	Description         string
	Required            bool
	Deprecated          bool
	Style               string
	Explode             *bool
	Schema              *Schema
	Content             []MediaTypeSpec
	SelectedContentType string
	Example             any
	Default             any
	CollectionFormat    string
}

type RequestBodySpec struct {
	Description string
	Required    bool
	Content     []MediaTypeSpec
}

type MediaTypeSpec struct {
	MediaType string
	Schema    *Schema
	Example   any
	Examples  map[string]Example
}

type Example struct {
	Summary     string
	Description string
	Value       any
}

type ResponseSpec struct {
	StatusCode  string
	Description string
	Content     []MediaTypeSpec
	Headers     []Parameter
}

type SecurityRequirement struct {
	Alternatives []SecurityAlternative
}

type SecurityAlternative struct {
	Schemes []SecurityRequirementRef
}

type SecurityRequirementRef struct {
	Name   string
	Scopes []string
}

type SecurityScheme struct {
	Name             string
	Type             SecuritySchemeType
	Description      string
	In               ParameterLocation
	ParameterName    string
	Scheme           HTTPAuthScheme
	BearerFormat     string
	RequiredScopes   []string
	ExtensionSummary map[string]string
}

type CapabilitySet struct {
	SupportsSwagger2Conversion bool
	SupportsOpenAPI3           bool
	SupportsCookieParameters   bool
	SupportsRequestBodies      bool
	SupportsServerVariables    bool
	SupportsSecuritySchemes    bool
}

type SpecWarning struct {
	Code    SpecWarningCode
	Message string
	Path    string
}

type Schema struct {
	Ref         string
	Type        string
	Format      string
	Title       string
	Description string
	Nullable    bool
	Properties  map[string]*Schema
	Items       *Schema
	Required    []string
	Enum        []any
	OneOf       []*Schema
	AnyOf       []*Schema
	AllOf       []*Schema
}
