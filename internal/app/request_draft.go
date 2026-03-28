package app

import "github.com/phergul/apiscope/internal/model"

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
		SelectedExamples: make(map[string]string),
	}
	if operation.RequestBody != nil && len(operation.RequestBody.Content) > 0 {
		draft.BodyMediaType = operation.RequestBody.Content[0].MediaType
	}

	session.RequestDrafts[key] = draft
	return draft
}

func SetDraftParameter(session *model.SessionState, operation *model.Operation, parameter model.Parameter, value string) *model.RequestDraft {
	draft := EnsureRequestDraft(session, operation)
	if draft == nil {
		return nil
	}

	target := parameterValueMap(draft, parameter.In)
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

func SetDraftBodyMediaType(session *model.SessionState, operation *model.Operation, mediaType string) *model.RequestDraft {
	draft := EnsureRequestDraft(session, operation)
	if draft == nil {
		return nil
	}

	draft.BodyMediaType = mediaType
	return draft
}

func SetDraftBodyRaw(session *model.SessionState, operation *model.Operation, value string) *model.RequestDraft {
	draft := EnsureRequestDraft(session, operation)
	if draft == nil {
		return nil
	}

	draft.BodyRaw = value
	return draft
}

func parameterValueMap(draft *model.RequestDraft, location model.ParameterLocation) map[string]string {
	if draft == nil {
		return nil
	}

	switch location {
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
	default:
		return nil
	}
}
