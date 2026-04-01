package app

import (
	"fmt"
	"strings"

	"github.com/phergul/apiscope/internal/model"
)

type AuthAlternativeStatus string

const (
	AuthAlternativeStatusReady         AuthAlternativeStatus = "ready"
	AuthAlternativeStatusIncomplete    AuthAlternativeStatus = "incomplete"
	AuthAlternativeStatusUnsupported   AuthAlternativeStatus = "unsupported"
	AuthAlternativeStatusMissingScheme AuthAlternativeStatus = "missing-scheme"
)

type AuthAlternativeProjection struct {
	Index                 int
	Status                AuthAlternativeStatus
	MissingFieldCount     int
	SatisfiedFieldCount   int
	FirstActionableTarget string
	Schemes               []AuthSchemeProjection
}

type AuthSchemeProjection struct {
	Ref               model.SecurityRequirementRef
	Scheme            model.SecurityScheme
	Found             bool
	ValidationTarget  string
	UnsupportedReason string
	Fields            []AuthFieldProjection
}

type AuthFieldProjection struct {
	Field            AuthField
	Label            string
	Meta             string
	Summary          string
	Satisfied        bool
	ValidationTarget string
}

// ProjectAuthAlternatives evaluates auth alternatives in declaration order for request-pane UX.
func ProjectAuthAlternatives(requirement *model.SecurityRequirement, securitySchemes map[string]model.SecurityScheme, authState map[string]model.AuthValue) []AuthAlternativeProjection {
	if requirement == nil || len(requirement.Alternatives) == 0 {
		return nil
	}

	projections := make([]AuthAlternativeProjection, 0, len(requirement.Alternatives))
	for index, alternative := range requirement.Alternatives {
		projections = append(projections, projectAuthAlternative(index, alternative, securitySchemes, authState))
	}

	return projections
}

func projectAuthAlternative(index int, alternative model.SecurityAlternative, securitySchemes map[string]model.SecurityScheme, authState map[string]model.AuthValue) AuthAlternativeProjection {
	projection := AuthAlternativeProjection{
		Index:   index,
		Schemes: make([]AuthSchemeProjection, 0, len(alternative.Schemes)),
	}

	missingScheme := false
	unsupported := false

	for _, ref := range alternative.Schemes {
		schemeProjection := AuthSchemeProjection{
			Ref:              ref,
			ValidationTarget: AuthAlternativeSchemeTarget(index, ref.Name),
		}

		scheme, ok := securitySchemes[ref.Name]
		if !ok {
			missingScheme = true
			projection.Schemes = append(projection.Schemes, schemeProjection)
			continue
		}

		schemeProjection.Found = true
		schemeProjection.Scheme = scheme

		fields, unsupportedReason := supportedAuthFieldsWithReason(scheme)
		if unsupportedReason != "" {
			unsupported = true
			schemeProjection.UnsupportedReason = unsupportedReason
			projection.Schemes = append(projection.Schemes, schemeProjection)
			continue
		}

		value := authState[ref.Name]
		for _, field := range fields {
			fieldProjection := AuthFieldProjection{
				Field:            field,
				Label:            AuthFieldLabel(scheme, field),
				Meta:             AuthFieldMeta(scheme, field),
				Summary:          AuthFieldSummary(value, field),
				Satisfied:        AuthFieldSatisfied(value, field),
				ValidationTarget: AuthAlternativeFieldTarget(index, ref.Name, field),
			}
			if fieldProjection.Satisfied {
				projection.SatisfiedFieldCount++
			} else {
				projection.MissingFieldCount++
				if projection.FirstActionableTarget == "" {
					projection.FirstActionableTarget = fieldProjection.ValidationTarget
				}
			}
			schemeProjection.Fields = append(schemeProjection.Fields, fieldProjection)
		}

		projection.Schemes = append(projection.Schemes, schemeProjection)
	}

	switch {
	case missingScheme:
		projection.Status = AuthAlternativeStatusMissingScheme
	case unsupported:
		projection.Status = AuthAlternativeStatusUnsupported
	case projection.MissingFieldCount == 0:
		projection.Status = AuthAlternativeStatusReady
	default:
		projection.Status = AuthAlternativeStatusIncomplete
	}

	return projection
}

func supportedAuthFieldsWithReason(scheme model.SecurityScheme) ([]AuthField, string) {
	switch scheme.Type {
	case model.SecuritySchemeTypeAPIKey:
		switch scheme.In {
		case model.ParameterLocationHeader, model.ParameterLocationQuery, model.ParameterLocationCookie:
			return []AuthField{AuthFieldAPIKey}, ""
		default:
			return nil, "API key location is not supported."
		}
	case model.SecuritySchemeTypeHTTP:
		switch scheme.Scheme {
		case model.HTTPAuthSchemeBearer:
			return []AuthField{AuthFieldBearerToken}, ""
		case model.HTTPAuthSchemeBasic:
			return []AuthField{AuthFieldUsername, AuthFieldPassword}, ""
		default:
			name := strings.TrimSpace(string(scheme.Scheme))
			if name == "" {
				return nil, "HTTP auth scheme is not supported."
			}
			return nil, fmt.Sprintf("HTTP auth scheme %q is not supported.", name)
		}
	default:
		name := strings.TrimSpace(string(scheme.Type))
		if name == "" {
			return nil, "Auth scheme type is not supported."
		}
		return nil, fmt.Sprintf("Auth scheme type %q is not supported.", name)
	}
}

func bestAuthAlternativeProjection(projections []AuthAlternativeProjection) AuthAlternativeProjection {
	bestIndex := 0
	bestScore := -1
	for index, projection := range projections {
		if projection.SatisfiedFieldCount > bestScore {
			bestIndex = index
			bestScore = projection.SatisfiedFieldCount
		}
	}

	return projections[bestIndex]
}
