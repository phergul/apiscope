package app

import (
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

func (r RequestValidationResult) HasIssues() bool {
	return len(r.Issues) > 0
}

func (r RequestValidationResult) FirstIssue() (RequestValidationIssue, bool) {
	if len(r.Issues) == 0 {
		return RequestValidationIssue{}, false
	}

	return r.Issues[0], true
}

func (r RequestValidationResult) IssueForTarget(target string) (RequestValidationIssue, bool) {
	for _, issue := range r.Issues {
		if issue.Target == target {
			return issue, true
		}
	}

	return RequestValidationIssue{}, false
}

func (r RequestValidationResult) MessagesForSection(section string) []string {
	messages := make([]string, 0, len(r.Issues))
	for _, issue := range r.Issues {
		if issue.Section == section {
			messages = append(messages, issue.Message)
		}
	}

	return messages
}

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
		if draft == nil || strings.TrimSpace(draft.BodyRaw) == "" {
			result.Issues = append(result.Issues, RequestValidationIssue{
				Section: "Body",
				Target:  ValidationTargetBodyRaw,
				Message: "Request body is required.",
			})
		}
	}

	return result
}

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
	default:
		return ""
	}
}

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
	default:
		return string(location)
	}
}
