package app

import (
	"strings"

	"github.com/phergul/apiscope/internal/model"
)

// EnsureRequestDraft returns the request draft for the selected operation, creating one when needed.
func EnsureRequestDraft(session *model.SessionState, operation *model.Operation) *model.RequestDraft {
	if session == nil || operation == nil {
		return nil
	}
	if session.RequestDrafts == nil {
		session.RequestDrafts = make(map[model.DraftKey]*model.RequestDraft)
	}

	key := model.NewDraftKey(session.SpecFingerprint, operation.Key)
	if draft, ok := session.RequestDrafts[key]; ok {
		return draft
	}

	draft := &model.RequestDraft{
		Key:              key,
		SpecFingerprint:  session.SpecFingerprint,
		OperationKey:     operation.Key,
		ServerURL:        session.SelectedServerURL,
		PathParams:       make(map[string]string),
		QueryParams:      make(map[string]string),
		HeaderParams:     make(map[string]string),
		CookieParams:     make(map[string]string),
		FormParams:       make(map[string]string),
		FormFileParams:   make(map[string]string),
		SelectedExamples: make(map[string]string),
	}
	seedRequestDraft(draft, operation)

	session.RequestDrafts[key] = draft
	return draft
}

// SetDraftParameter stores or clears a parameter value in the current request draft.
func SetDraftParameter(session *model.SessionState, operation *model.Operation, parameter model.Parameter, value string) *model.RequestDraft {
	draft := EnsureRequestDraft(session, operation)
	if draft == nil {
		return nil
	}

	target := parameterValueMap(draft, parameter)
	if target == nil {
		return draft
	}
	if value == "" {
		delete(target, parameter.Name)
		return draft
	}

	target[parameter.Name] = value
	return draft
}

// SetDraftBodyMediaType stores the selected request-body media type in the current draft.
func SetDraftBodyMediaType(session *model.SessionState, operation *model.Operation, mediaType string) *model.RequestDraft {
	draft := EnsureRequestDraft(session, operation)
	if draft == nil {
		return nil
	}

	replaceBody := shouldReplaceSeededBody(operation, draft)
	draft.BodyMediaType = strings.TrimSpace(mediaType)
	if replaceBody {
		draft.BodyRaw = ""
		seedDraftBody(draft, operation)
	}
	return draft
}

// SetDraftBodyRaw stores the raw request-body text in the current draft.
func SetDraftBodyRaw(session *model.SessionState, operation *model.Operation, value string) *model.RequestDraft {
	draft := EnsureRequestDraft(session, operation)
	if draft == nil {
		return nil
	}

	draft.BodyRaw = value
	return draft
}

// parameterValueMap returns the parameter map for the requested parameter.
func parameterValueMap(draft *model.RequestDraft, parameter model.Parameter) map[string]string {
	if draft == nil {
		return nil
	}

	switch parameter.In {
	case model.ParameterLocationPath:
		if draft.PathParams == nil {
			draft.PathParams = make(map[string]string)
		}
		return draft.PathParams
	case model.ParameterLocationQuery:
		if draft.QueryParams == nil {
			draft.QueryParams = make(map[string]string)
		}
		return draft.QueryParams
	case model.ParameterLocationHeader:
		if draft.HeaderParams == nil {
			draft.HeaderParams = make(map[string]string)
		}
		return draft.HeaderParams
	case model.ParameterLocationCookie:
		if draft.CookieParams == nil {
			draft.CookieParams = make(map[string]string)
		}
		return draft.CookieParams
	case model.ParameterLocationForm:
		if parameter.FormInputKind == model.FormInputKindFile {
			if draft.FormFileParams == nil {
				draft.FormFileParams = make(map[string]string)
			}
			return draft.FormFileParams
		}
		if draft.FormParams == nil {
			draft.FormParams = make(map[string]string)
		}
		return draft.FormParams
	default:
		return nil
	}
}
