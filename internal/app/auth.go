package app

import (
	"fmt"
	"strings"

	"github.com/phergul/apiscope/internal/model"
)

// ValidationSectionAuth is the request-pane section label used for auth validation issues.
const ValidationSectionAuth = "Auth"

// AuthField identifies one editable field for a supported auth scheme.
type AuthField string

const (
	AuthFieldAPIKey      AuthField = "api_key"
	AuthFieldBearerToken AuthField = "bearer_token"
	AuthFieldUsername    AuthField = "username"
	AuthFieldPassword    AuthField = "password"
)

// EffectiveSecurityRequirement returns the operation security requirement, falling back
// to the top-level spec security requirement when the operation does not override it.
func EffectiveSecurityRequirement(session model.SessionState, operation *model.Operation) *model.SecurityRequirement {
	if operation != nil && operation.Security != nil {
		return operation.Security
	}
	if session.Spec == nil {
		return nil
	}

	return session.Spec.Security
}

// EnsureAuthState returns the session auth-state map, initializing it when needed.
func EnsureAuthState(session *model.SessionState) map[string]model.AuthValue {
	if session == nil {
		return nil
	}
	if session.AuthState == nil {
		session.AuthState = make(map[string]model.AuthValue)
	}

	return session.AuthState
}

// AuthValue returns the stored auth value for the requested scheme name.
func AuthValue(session model.SessionState, schemeName string) model.AuthValue {
	if session.AuthState == nil {
		return model.AuthValue{}
	}

	return session.AuthState[schemeName]
}

// SetAuthField stores or clears one auth field value for the requested scheme.
func SetAuthField(session *model.SessionState, scheme model.SecurityScheme, field AuthField, value string) {
	state := EnsureAuthState(session)
	if state == nil {
		return
	}

	authValue := state[scheme.Name]
	switch field {
	case AuthFieldAPIKey:
		authValue.Type = model.AuthSchemeValueTypeAPIKey
		authValue.APIKey = value
	case AuthFieldBearerToken:
		authValue.Type = model.AuthSchemeValueTypeBearer
		authValue.BearerToken = value
	case AuthFieldUsername:
		authValue.Type = model.AuthSchemeValueTypeBasic
		authValue.Username = value
	case AuthFieldPassword:
		authValue.Type = model.AuthSchemeValueTypeBasic
		authValue.Password = value
	default:
		return
	}

	if authValueEmpty(authValue) {
		delete(state, scheme.Name)
		return
	}

	state[scheme.Name] = authValue
}

// authValueEmpty reports whether the auth value has no usable credentials stored.
func authValueEmpty(value model.AuthValue) bool {
	return strings.TrimSpace(value.APIKey) == "" &&
		strings.TrimSpace(value.Username) == "" &&
		strings.TrimSpace(value.Password) == "" &&
		strings.TrimSpace(value.BearerToken) == ""
}

// AuthFieldValue returns the raw stored value for the requested auth field.
func AuthFieldValue(value model.AuthValue, field AuthField) string {
	switch field {
	case AuthFieldAPIKey:
		return value.APIKey
	case AuthFieldBearerToken:
		return value.BearerToken
	case AuthFieldUsername:
		return value.Username
	case AuthFieldPassword:
		return value.Password
	default:
		return ""
	}
}

// AuthFieldSatisfied reports whether the requested auth field has a non-blank value.
func AuthFieldSatisfied(value model.AuthValue, field AuthField) bool {
	return strings.TrimSpace(AuthFieldValue(value, field)) != ""
}

// AuthFieldSummary returns the masked request-pane summary for the auth field value.
func AuthFieldSummary(value model.AuthValue, field AuthField) string {
	switch field {
	case AuthFieldAPIKey:
		if AuthFieldSatisfied(value, field) {
			return "key set"
		}
		return "<unset>"
	case AuthFieldBearerToken:
		if AuthFieldSatisfied(value, field) {
			return "token set"
		}
		return "<unset>"
	case AuthFieldUsername:
		if AuthFieldSatisfied(value, field) {
			return "username set"
		}
		return "<unset>"
	case AuthFieldPassword:
		if AuthFieldSatisfied(value, field) {
			return "password set"
		}
		return "<unset>"
	default:
		return "<unset>"
	}
}

// SupportedAuthFields returns the editable auth fields for the provided scheme.
func SupportedAuthFields(scheme model.SecurityScheme) []AuthField {
	fields, _ := supportedAuthFieldsWithReason(scheme)
	return fields
}

// AuthFieldLabel returns the request-pane row label for the auth field.
func AuthFieldLabel(scheme model.SecurityScheme, field AuthField) string {
	switch field {
	case AuthFieldAPIKey:
		return scheme.Name
	case AuthFieldBearerToken:
		return scheme.Name
	case AuthFieldUsername:
		return scheme.Name + " username"
	case AuthFieldPassword:
		return scheme.Name + " password"
	default:
		return scheme.Name
	}
}

// AuthFieldMeta returns the request-pane row metadata for the auth field.
func AuthFieldMeta(scheme model.SecurityScheme, field AuthField) string {
	switch field {
	case AuthFieldAPIKey:
		return "API key"
	case AuthFieldBearerToken:
		return "Bearer token"
	case AuthFieldUsername, AuthFieldPassword:
		return "Basic auth"
	default:
		return string(scheme.Type)
	}
}

// AuthFieldTarget builds the stable validation target ID for an auth field row.
func AuthFieldTarget(schemeName string, field AuthField) string {
	if field == "" {
		return "auth:" + schemeName
	}

	return "auth:" + schemeName + ":" + string(field)
}

// AuthAlternativeSchemeTarget builds the validation target for one scheme inside one auth alternative.
func AuthAlternativeSchemeTarget(alternativeIndex int, schemeName string) string {
	return fmt.Sprintf("auth:alt:%d:%s", alternativeIndex, schemeName)
}

// AuthAlternativeFieldTarget builds the validation target for one auth field inside one auth alternative.
func AuthAlternativeFieldTarget(alternativeIndex int, schemeName string, field AuthField) string {
	if field == "" {
		return AuthAlternativeSchemeTarget(alternativeIndex, schemeName)
	}

	return AuthAlternativeSchemeTarget(alternativeIndex, schemeName) + ":" + string(field)
}

// ValidateAuth checks whether one declared auth alternative is fully satisfied.
func ValidateAuth(requirement *model.SecurityRequirement, securitySchemes map[string]model.SecurityScheme, authState map[string]model.AuthValue) RequestValidationResult {
	projections := ProjectAuthAlternatives(requirement, securitySchemes, authState)
	if len(projections) == 0 {
		return RequestValidationResult{}
	}

	for _, alternative := range projections {
		if alternative.Status == AuthAlternativeStatusReady {
			return RequestValidationResult{}
		}
	}

	alternative := bestAuthAlternativeProjection(projections)
	result := RequestValidationResult{}
	for _, scheme := range alternative.Schemes {
		if !scheme.Found {
			result.Issues = append(result.Issues, RequestValidationIssue{
				Section: ValidationSectionAuth,
				Target:  scheme.ValidationTarget,
				Message: "Security scheme is missing from the spec.",
			})
			continue
		}

		if scheme.UnsupportedReason != "" {
			result.Issues = append(result.Issues, RequestValidationIssue{
				Section: ValidationSectionAuth,
				Target:  scheme.ValidationTarget,
				Message: scheme.UnsupportedReason,
			})
			continue
		}

		for _, field := range scheme.Fields {
			if field.Satisfied {
				continue
			}

			result.Issues = append(result.Issues, RequestValidationIssue{
				Section: ValidationSectionAuth,
				Target:  field.ValidationTarget,
				Message: authFieldMissingMessage(field.Field),
			})
		}
	}

	return result
}

// authAlternativeForValidation selects the most complete alternative so validation can
// point the user at the nearest viable auth setup instead of reporting every branch.
func authAlternativeForValidation(alternatives []model.SecurityAlternative, securitySchemes map[string]model.SecurityScheme, authState map[string]model.AuthValue) model.SecurityAlternative {
	bestIndex := 0
	bestScore := -1
	for index, alternative := range alternatives {
		score := authAlternativeScore(alternative, securitySchemes, authState)
		if score > bestScore {
			bestIndex = index
			bestScore = score
		}
	}

	return alternatives[bestIndex]
}

// authAlternativeScore counts how many required fields in one auth alternative are already present.
func authAlternativeScore(alternative model.SecurityAlternative, securitySchemes map[string]model.SecurityScheme, authState map[string]model.AuthValue) int {
	score := 0
	for _, ref := range alternative.Schemes {
		scheme, ok := securitySchemes[ref.Name]
		if !ok {
			continue
		}

		fields := SupportedAuthFields(scheme)
		if len(fields) == 0 {
			continue
		}

		value := authState[ref.Name]
		for _, field := range fields {
			if AuthFieldSatisfied(value, field) {
				score++
			}
		}
	}

	return score
}

// authAlternativeSatisfied reports whether all supported fields in the alternative are present.
func authAlternativeSatisfied(alternative model.SecurityAlternative, securitySchemes map[string]model.SecurityScheme, authState map[string]model.AuthValue) bool {
	for _, ref := range alternative.Schemes {
		scheme, ok := securitySchemes[ref.Name]
		if !ok {
			return false
		}

		fields := SupportedAuthFields(scheme)
		if len(fields) == 0 {
			return false
		}

		value := authState[ref.Name]
		for _, field := range fields {
			if !AuthFieldSatisfied(value, field) {
				return false
			}
		}
	}

	return true
}

// authFieldMissingMessage returns the validation copy for one missing auth field.
func authFieldMissingMessage(field AuthField) string {
	switch field {
	case AuthFieldAPIKey:
		return "API key is required."
	case AuthFieldBearerToken:
		return "Bearer token is required."
	case AuthFieldUsername:
		return "Username is required."
	case AuthFieldPassword:
		return "Password is required."
	default:
		return "Auth value is required."
	}
}
