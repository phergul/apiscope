package converter

import (
	"fmt"

	"github.com/phergul/apiscope/internal/spec/internal/pipeline"

	"github.com/getkin/kin-openapi/openapi3"
)

func convertSwaggerSecuritySchemes(parsed *pipeline.ParsedDocument) (openapi3.SecuritySchemes, error) {
	rawDefinitions, ok := getMap(parsed.SwaggerDoc, "securityDefinitions")
	if !ok || len(rawDefinitions) == 0 {
		return nil, nil
	}

	schemes := make(openapi3.SecuritySchemes, len(rawDefinitions))
	for name, rawDefinition := range rawDefinitions {
		definitionMap, ok := rawDefinition.(map[string]any)
		if !ok {
			return nil, &pipeline.Error{
				Kind:   pipeline.ErrorKindSwaggerConversionFailure,
				Op:     "convert security definitions",
				Source: parsed.Document.CanonicalLocation,
				Err:    fmt.Errorf("security definition %q must be an object", name),
			}
		}

		scheme, err := convertSwaggerSecurityScheme(parsed.Document.CanonicalLocation, name, definitionMap)
		if err != nil {
			return nil, err
		}
		schemes[name] = scheme
	}

	return schemes, nil
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

func securityRequirementPtr(requirements openapi3.SecurityRequirements) *openapi3.SecurityRequirements {
	if len(requirements) == 0 {
		return nil
	}

	return &requirements
}
