package converter

import (
	"fmt"

	"github.com/phergul/apiscope/internal/model"
	"github.com/phergul/apiscope/internal/spec/internal/pipeline"

	"github.com/getkin/kin-openapi/openapi3"
)

func Convert(parsed *pipeline.ParsedDocument) (*pipeline.ConvertedDocument, error) {
	switch parsed.SourceFamily {
	case model.SourceFamilyOpenAPI3:
		return &pipeline.ConvertedDocument{
			BaseDocument: pipeline.BaseDocument{
				Document:      parsed.Document,
				SourceFamily:  parsed.SourceFamily,
				SourceVersion: parsed.SourceVersion,
				OpenAPI3Doc:   parsed.OpenAPI3Doc,
			},
		}, nil
	case model.SourceFamilySwagger2:
		doc, err := convertSwaggerDocument(parsed)
		if err != nil {
			return nil, err
		}
		return &pipeline.ConvertedDocument{
			BaseDocument: pipeline.BaseDocument{
				Document:      parsed.Document,
				SourceFamily:  parsed.SourceFamily,
				SourceVersion: parsed.SourceVersion,
				OpenAPI3Doc:   doc,
			},
		}, nil
	default:
		return nil, &pipeline.Error{
			Kind:   pipeline.ErrorKindUnsupportedFamily,
			Op:     "convert document",
			Source: parsed.Document.CanonicalLocation,
			Err:    fmt.Errorf("unexpected source family %q", parsed.SourceFamily),
		}
	}
}

func convertSwaggerDocument(parsed *pipeline.ParsedDocument) (*openapi3.T, error) {
	doc := &openapi3.T{
		OpenAPI: "3.0.3",
		Info:    convertSwaggerInfo(parsed),
		Paths:   &openapi3.Paths{},
	}

	servers, err := convertSwaggerServers(parsed.SwaggerDoc)
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

	security, err := convertSecurityRequirementList(parsed.Document.CanonicalLocation, "top-level security", getSliceMap(parsed.SwaggerDoc, "security"))
	if err != nil {
		return nil, err
	}
	doc.Security = security

	globalConsumes := getStringSlice(parsed.SwaggerDoc, "consumes")
	globalProduces := getStringSlice(parsed.SwaggerDoc, "produces")
	rawParameterDefinitions, _ := getMap(parsed.SwaggerDoc, "parameters")

	paths, err := convertSwaggerPaths(parsed.Document.CanonicalLocation, parsed.SwaggerDoc, globalConsumes, globalProduces, rawParameterDefinitions)
	if err != nil {
		return nil, err
	}
	doc.Paths = paths

	return doc, nil
}
