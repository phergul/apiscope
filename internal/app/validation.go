package app

import (
	"os"
	"strings"

	"github.com/phergul/apiscope/internal/model"
)

const (
	ValidationTargetBodyMediaType = "body:media_type"
	ValidationTargetBodyRaw       = "body:raw"
)

type RequestValidationIssue struct {
	Section string
	Target  string
	Message string
}

type RequestValidationResult struct {
	Issues []RequestValidationIssue
}

// HasIssues reports whether the validation result contains any issues.
func (r RequestValidationResult) HasIssues() bool {
	return len(r.Issues) > 0
}

// FirstIssue returns the first validation issue when one exists.
func (r RequestValidationResult) FirstIssue() (RequestValidationIssue, bool) {
	if len(r.Issues) == 0 {
		return RequestValidationIssue{}, false
	}

	return r.Issues[0], true
}

// IssueForTarget returns the validation issue for the requested target when present.
func (r RequestValidationResult) IssueForTarget(target string) (RequestValidationIssue, bool) {
	for _, issue := range r.Issues {
		if issue.Target == target {
			return issue, true
		}
	}

	return RequestValidationIssue{}, false
}

// MessagesForSection returns validation messages that belong to the requested request section.
func (r RequestValidationResult) MessagesForSection(section string) []string {
	messages := make([]string, 0, len(r.Issues))
	for _, issue := range r.Issues {
		if issue.Section == section {
			messages = append(messages, issue.Message)
		}
	}

	return messages
}

// ValidateRequest checks the current draft against the required operation inputs.
func ValidateRequest(operation *model.Operation, draft *model.RequestDraft) RequestValidationResult {
	if operation == nil {
		return RequestValidationResult{}
	}

	result := RequestValidationResult{}
	for _, parameter := range operation.Parameters {
		if !parameter.Required {
			continue
		}

		value := strings.TrimSpace(draftParameterValue(draft, parameter))
		if value != "" {
			continue
		}

		result.Issues = append(result.Issues, RequestValidationIssue{
			Section: requestSectionForLocation(parameter.In),
			Target:  string(parameter.In) + ":" + parameter.Name,
			Message: "Required value missing.",
		})
	}

	if operation.RequestBody != nil && operation.RequestBody.Required {
		if draft == nil || strings.TrimSpace(draft.BodyMediaType) == "" {
			result.Issues = append(result.Issues, RequestValidationIssue{
				Section: "Body",
				Target:  ValidationTargetBodyMediaType,
				Message: "Select a media type for the request body.",
			})
		}
		if bodyFields := ProjectBodyFieldParameters(operation, draft); len(bodyFields) > 0 {
			for _, parameter := range bodyFields {
				if !parameter.Required {
					continue
				}
				if strings.TrimSpace(draftParameterValue(draft, parameter)) != "" {
					continue
				}
				result.Issues = append(result.Issues, RequestValidationIssue{
					Section: "Body",
					Target:  string(parameter.In) + ":" + parameter.Name,
					Message: "Required value missing.",
				})
			}
		} else if draft == nil || strings.TrimSpace(draft.BodyRaw) == "" {
			result.Issues = append(result.Issues, RequestValidationIssue{
				Section: "Body",
				Target:  ValidationTargetBodyRaw,
				Message: "Request body is required.",
			})
		}
	}

	return result
}

// ValidateExecutableRequest checks request inputs and auth requirements before execution.
func ValidateExecutableRequest(session model.SessionState, operation *model.Operation, draft *model.RequestDraft) RequestValidationResult {
	if operation != nil {
		ResolveAuthFromDraftEnvVars(&session, operation)
	}
	result := ValidateRequest(operation, draft)
	securitySchemes := map[string]model.SecurityScheme(nil)
	if session.Spec != nil {
		securitySchemes = session.Spec.SecuritySchemes
	}
	authValidation := ValidateAuth(EffectiveSecurityRequirement(session, operation), securitySchemes, session.AuthState)
	result.Issues = append(result.Issues, authValidation.Issues...)
	return result
}

func DraftAuthEnvVar(session *model.SessionState, operation *model.Operation, schemeName string, field model.AuthField) string {
	if session == nil || operation == nil {
		return ""
	}

	draft := EnsureRequestDraft(session, operation)
	if draft == nil {
		return ""
	}
	if draft.BodyPartEncoding == nil {
		return ""
	}

	key := "auth:env:" + schemeName + ":" + string(field)
	return strings.TrimSpace(draft.BodyPartEncoding[key])
}

// ResolveAuthFromDraftEnvVars applies env-var auth bindings from the current draft into session auth state.
func ResolveAuthFromDraftEnvVars(session *model.SessionState, operation *model.Operation) {
	if session == nil || session.Spec == nil || operation == nil {
		return
	}

	requirement := EffectiveSecurityRequirement(*session, operation)
	if requirement == nil || len(requirement.Alternatives) == 0 {
		return
	}

	for _, alternative := range requirement.Alternatives {
		for _, ref := range alternative.Schemes {
			scheme, ok := session.Spec.SecuritySchemes[ref.Name]
			if !ok {
				continue
			}
			for _, field := range SupportedAuthFields(scheme) {
				authValue := AuthValue(*session, scheme.Name)
				if AuthFieldSatisfied(authValue, field) {
					continue
				}
				envVarName := DraftAuthEnvVar(session, operation, ref.Name, field)
				if strings.TrimSpace(envVarName) == "" {
					envVarName = DraftAuthEnvVar(session, operation, scheme.Name, field)
				}
				if strings.TrimSpace(envVarName) == "" {
					continue
				}
				value, ok := os.LookupEnv(envVarName)
				if !ok || strings.TrimSpace(value) == "" {
					continue
				}
				SetAuthField(session, scheme, field, value)
			}
		}
	}
}

// draftParameterValue returns the current draft value for the requested parameter.
func draftParameterValue(draft *model.RequestDraft, parameter model.Parameter) string {
	if draft == nil {
		return ""
	}

	switch parameter.In {
	case model.ParameterLocationPath:
		return draft.PathParams[parameter.Name]
	case model.ParameterLocationQuery:
		return draft.QueryParams[parameter.Name]
	case model.ParameterLocationHeader:
		return draft.HeaderParams[parameter.Name]
	case model.ParameterLocationCookie:
		return draft.CookieParams[parameter.Name]
	case model.ParameterLocationForm:
		if parameter.FormInputKind == model.FormInputKindFile {
			return draft.FormFileParams[parameter.Name]
		}
		return draft.FormParams[parameter.Name]
	default:
		return ""
	}
}

// requestSectionForLocation maps a parameter location to its request-pane section label.
func requestSectionForLocation(location model.ParameterLocation) string {
	switch location {
	case model.ParameterLocationPath:
		return "Path"
	case model.ParameterLocationQuery:
		return "Query"
	case model.ParameterLocationHeader:
		return "Header"
	case model.ParameterLocationCookie:
		return "Cookie"
	case model.ParameterLocationForm:
		return "Form"
	default:
		return string(location)
	}
}
