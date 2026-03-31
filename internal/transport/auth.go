package transport

import (
	"encoding/base64"
	"errors"
	"net/http"
	"strings"

	"github.com/phergul/apiscope/internal/model"
)

// applyAuth applies the first satisfied auth alternative to the prepared request.
func applyAuth(request *http.Request, requirement *model.SecurityRequirement, securitySchemes map[string]model.SecurityScheme, authState map[string]model.AuthValue) error {
	if request == nil || requirement == nil || len(requirement.Alternatives) == 0 {
		return nil
	}

	alternative, ok := firstSatisfiedAuthAlternative(requirement, securitySchemes, authState)
	if !ok {
		return errors.New("auth requirement not satisfied")
	}

	for _, ref := range alternative.Schemes {
		scheme, ok := securitySchemes[ref.Name]
		if !ok {
			return errors.New("security scheme missing from spec")
		}
		value := authState[ref.Name]
		if err := applyAuthScheme(request, scheme, value); err != nil {
			return err
		}
	}

	return nil
}

// firstSatisfiedAuthAlternative returns the first auth alternative with all required values present.
func firstSatisfiedAuthAlternative(requirement *model.SecurityRequirement, securitySchemes map[string]model.SecurityScheme, authState map[string]model.AuthValue) (model.SecurityAlternative, bool) {
	if requirement == nil {
		return model.SecurityAlternative{}, false
	}

	for _, alternative := range requirement.Alternatives {
		if authAlternativeSatisfied(alternative, securitySchemes, authState) {
			return alternative, true
		}
	}

	return model.SecurityAlternative{}, false
}

// authAlternativeSatisfied reports whether the provided auth alternative is ready to apply.
func authAlternativeSatisfied(alternative model.SecurityAlternative, securitySchemes map[string]model.SecurityScheme, authState map[string]model.AuthValue) bool {
	for _, ref := range alternative.Schemes {
		scheme, ok := securitySchemes[ref.Name]
		if !ok {
			return false
		}
		value := authState[ref.Name]
		switch {
		case scheme.Type == model.SecuritySchemeTypeAPIKey:
			if strings.TrimSpace(value.APIKey) == "" {
				return false
			}
		case scheme.Type == model.SecuritySchemeTypeHTTP && scheme.Scheme == model.HTTPAuthSchemeBearer:
			if strings.TrimSpace(value.BearerToken) == "" {
				return false
			}
		case scheme.Type == model.SecuritySchemeTypeHTTP && scheme.Scheme == model.HTTPAuthSchemeBasic:
			if strings.TrimSpace(value.Username) == "" || strings.TrimSpace(value.Password) == "" {
				return false
			}
		default:
			return false
		}
	}

	return true
}

// applyAuthScheme writes one concrete auth scheme onto the prepared HTTP request.
func applyAuthScheme(request *http.Request, scheme model.SecurityScheme, value model.AuthValue) error {
	switch {
	case scheme.Type == model.SecuritySchemeTypeAPIKey:
		return applyAPIKey(request, scheme, value.APIKey)
	case scheme.Type == model.SecuritySchemeTypeHTTP && scheme.Scheme == model.HTTPAuthSchemeBearer:
		if strings.TrimSpace(value.BearerToken) == "" {
			return errors.New("bearer token missing")
		}
		request.Header.Set("Authorization", "Bearer "+value.BearerToken)
		return nil
	case scheme.Type == model.SecuritySchemeTypeHTTP && scheme.Scheme == model.HTTPAuthSchemeBasic:
		if strings.TrimSpace(value.Username) == "" || strings.TrimSpace(value.Password) == "" {
			return errors.New("basic auth credentials missing")
		}
		encoded := base64.StdEncoding.EncodeToString([]byte(value.Username + ":" + value.Password))
		request.Header.Set("Authorization", "Basic "+encoded)
		return nil
	default:
		return errors.New("unsupported auth scheme type")
	}
}

// applyAPIKey writes one API-key scheme to the configured header, query param, or cookie.
func applyAPIKey(request *http.Request, scheme model.SecurityScheme, value string) error {
	if strings.TrimSpace(value) == "" {
		return errors.New("api key missing")
	}

	switch scheme.In {
	case model.ParameterLocationHeader:
		request.Header.Set(scheme.ParameterName, value)
	case model.ParameterLocationQuery:
		query := request.URL.Query()
		query.Set(scheme.ParameterName, value)
		request.URL.RawQuery = query.Encode()
	case model.ParameterLocationCookie:
		request.AddCookie(&http.Cookie{Name: scheme.ParameterName, Value: value})
	default:
		return errors.New("unsupported api key location")
	}

	return nil
}
