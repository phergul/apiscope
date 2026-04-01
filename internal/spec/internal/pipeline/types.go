package pipeline

import (
	"github.com/phergul/apiscope/internal/model"

	"github.com/getkin/kin-openapi/openapi3"
)

type SourceKind string

const (
	SourceKindFile SourceKind = "file"
	SourceKindURL  SourceKind = "url"
)

type Source struct {
	Kind  SourceKind
	Value string
}

type DocumentFormat string

const (
	DocumentFormatJSON DocumentFormat = "json"
	DocumentFormatYAML DocumentFormat = "yaml"
)

const (
	SwaggerCollectionFormatExtension    = "x-apiscope-swagger-collection-format"
	SwaggerParameterLocationExtension   = "x-apiscope-swagger-parameter-location"
	SwaggerFormBodyMediaTypeExtension   = "x-apiscope-swagger-form-body-media-type"
	SwaggerAssumedFormEncodingExtension = "x-apiscope-swagger-assumed-form-encoding"
)

type LoadedDocument struct {
	Source            Source
	CanonicalLocation string
	Raw               []byte
	Format            DocumentFormat
	MediaType         string
	FinalURL          string
}

type BaseDocument struct {
	Document      *LoadedDocument
	SourceFamily  model.SourceFamily
	SourceVersion string
	OpenAPI3Doc   *openapi3.T
}

type ParsedDocument struct {
	BaseDocument
	SwaggerDoc map[string]any
}

type ConvertedDocument struct {
	BaseDocument
}

type ResolvedDocument struct {
	BaseDocument
}
