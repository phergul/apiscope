package normalise

import (
	"fmt"
	"sort"
	"strings"

	"github.com/phergul/apiscope/internal/model"
	"github.com/phergul/apiscope/internal/spec/internal/pipeline"

	"github.com/getkin/kin-openapi/openapi3"
)

func normaliseSecuritySchemes(components *openapi3.Components, state *normalisationState) map[string]model.SecurityScheme {
	if components == nil || len(components.SecuritySchemes) == 0 {
		return nil
	}

	result := make(map[string]model.SecurityScheme, len(components.SecuritySchemes))
	for name, schemeRef := range components.SecuritySchemes {
		if schemeRef == nil || schemeRef.Value == nil {
			continue
		}

		scheme := schemeRef.Value
		normalised := model.SecurityScheme{
			Name:         name,
			Description:  scheme.Description,
			BearerFormat: scheme.BearerFormat,
		}

		switch scheme.Type {
		case "apiKey":
			normalised.Type = model.SecuritySchemeTypeAPIKey
			if in, ok := normaliseParameterLocation(scheme.In, nil); ok {
				normalised.In = in
			}
			normalised.ParameterName = scheme.Name
		case "http":
			normalised.Type = model.SecuritySchemeTypeHTTP
			switch strings.ToLower(scheme.Scheme) {
			case "basic":
				normalised.Scheme = model.HTTPAuthSchemeBasic
			case "bearer":
				normalised.Scheme = model.HTTPAuthSchemeBearer
			default:
				normalised.Scheme = model.HTTPAuthSchemeUnknown
				state.warnings = append(state.warnings, model.SpecWarning{
					Code:    model.SpecWarningUnsupportedFeature,
					Message: fmt.Sprintf("http security scheme %q uses unsupported scheme %q", name, scheme.Scheme),
					Path:    name,
				})
			}
		default:
			state.warnings = append(state.warnings, model.SpecWarning{
				Code:    model.SpecWarningUnsupportedFeature,
				Message: fmt.Sprintf("security scheme %q of type %q is not fully represented in the normalised model", name, scheme.Type),
				Path:    name,
			})
		}

		result[name] = normalised
	}

	return result
}

func normaliseSecurityRequirements(requirements openapi3.SecurityRequirements) *model.SecurityRequirement {
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

func normaliseSecurityRequirementsPtr(requirements *openapi3.SecurityRequirements) *model.SecurityRequirement {
	if requirements == nil {
		return nil
	}
	return normaliseSecurityRequirements(*requirements)
}

func deriveCapabilities(resolved *pipeline.ResolvedDocument) model.CapabilitySet {
	return model.CapabilitySet{
		SupportsSwagger2Conversion: resolved.SourceFamily == model.SourceFamilySwagger2,
		SupportsOpenAPI3:           resolved.SourceFamily == model.SourceFamilyOpenAPI3,
		SupportsCookieParameters:   resolved.SourceFamily == model.SourceFamilyOpenAPI3,
		SupportsRequestBodies:      resolved.SourceFamily == model.SourceFamilyOpenAPI3 || resolved.SourceFamily == model.SourceFamilySwagger2,
		SupportsServerVariables:    resolved.SourceFamily == model.SourceFamilyOpenAPI3,
		SupportsSecuritySchemes:    resolved.SourceFamily == model.SourceFamilyOpenAPI3 || resolved.SourceFamily == model.SourceFamilySwagger2,
	}
}
