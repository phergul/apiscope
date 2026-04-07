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
	seedDraftBodyFields(draft, operation)
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

// DraftBodyExampleNames returns the sorted named examples for the active body media type.
func DraftBodyExampleNames(operation *model.Operation, draft *model.RequestDraft) []string {
	mediaType, spec, ok := activeBodyMediaTypeSpec(operation, draft)
	if !ok || strings.TrimSpace(mediaType) == "" {
		return nil
	}

	return append([]string(nil), mediaTypeExampleNames(spec)...)
}

// DraftBodyExampleName returns the selected named example for the active body media type.
func DraftBodyExampleName(operation *model.Operation, draft *model.RequestDraft) string {
	mediaType, spec, ok := activeBodyMediaTypeSpec(operation, draft)
	if !ok || strings.TrimSpace(mediaType) == "" {
		return ""
	}

	_, exampleName, ok := seededMediaTypeValue(spec, selectedBodyExampleName(draft, mediaType))
	if !ok {
		return ""
	}

	return exampleName
}

// SetDraftBodyExample applies one named example to the active body media type.
func SetDraftBodyExample(session *model.SessionState, operation *model.Operation, exampleName string) *model.RequestDraft {
	draft := EnsureRequestDraft(session, operation)
	if draft == nil {
		return nil
	}

	mediaType, _, ok := activeBodyMediaTypeSpec(operation, draft)
	if !ok {
		return draft
	}
	body, selectedName, ok := seededRequestBody(operation, draft, mediaType)
	preferred := strings.TrimSpace(exampleName)
	if preferred != "" {
		body, selectedName, ok = seededRequestBodyForExample(operation, mediaType, preferred)
	}
	if !ok {
		return draft
	}

	draft.BodyRaw = body
	setSelectedBodyExample(draft, mediaType, selectedName)
	return draft
}

// CycleDraftBodyExample advances the active body example selection when multiple named examples exist.
func CycleDraftBodyExample(session *model.SessionState, operation *model.Operation) bool {
	draft := EnsureRequestDraft(session, operation)
	if draft == nil {
		return false
	}

	names := DraftBodyExampleNames(operation, draft)
	if len(names) < 2 {
		return false
	}

	current := DraftBodyExampleName(operation, draft)
	currentIndex := 0
	for index, name := range names {
		if name == current {
			currentIndex = index
			break
		}
	}

	next := names[(currentIndex+1)%len(names)]
	return SetDraftBodyExample(session, operation, next) != nil
}

func activeBodyMediaTypeSpec(operation *model.Operation, draft *model.RequestDraft) (string, model.MediaTypeSpec, bool) {
	if operation == nil || operation.RequestBody == nil || len(operation.RequestBody.Content) == 0 {
		return "", model.MediaTypeSpec{}, false
	}

	mediaType := defaultDraftBodyMediaType(operation)
	if draft != nil && strings.TrimSpace(draft.BodyMediaType) != "" {
		mediaType = strings.TrimSpace(draft.BodyMediaType)
	}
	spec, ok := requestBodyMediaType(operation, mediaType)
	if !ok {
		return "", model.MediaTypeSpec{}, false
	}

	return mediaType, spec, true
}

func seededRequestBodyForExample(operation *model.Operation, mediaType, exampleName string) (string, string, bool) {
	spec, ok := requestBodyMediaType(operation, mediaType)
	if !ok {
		return "", "", false
	}

	value, selectedName, ok := seededMediaTypeValue(spec, exampleName)
	if !ok {
		return "", "", false
	}
	body, ok := formatSeededBody(value)
	if !ok {
		return "", "", false
	}

	return body, selectedName, true
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
