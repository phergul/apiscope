package app

import (
	"strings"

	"github.com/phergul/apiscope/internal/model"
)

const ValidationTargetBodyEncodingPrefix = "body:encoding:"

func DraftBodyPartContentType(draft *model.RequestDraft, fieldName string) string {
	if draft == nil {
		return ""
	}

	return strings.TrimSpace(draft.BodyPartEncoding[fieldName])
}

func SetDraftBodyPartContentType(session *model.SessionState, operation *model.Operation, fieldName, contentType string) *model.RequestDraft {
	draft := EnsureRequestDraft(session, operation)
	if draft == nil {
		return nil
	}

	fieldName = strings.TrimSpace(fieldName)
	if fieldName == "" {
		return draft
	}
	if draft.BodyPartEncoding == nil {
		draft.BodyPartEncoding = make(map[string]string)
	}

	contentType = strings.TrimSpace(contentType)
	if contentType == "" {
		delete(draft.BodyPartEncoding, fieldName)
		return draft
	}
	if _, ok := multipartBodyField(operation, draft, fieldName); !ok {
		return draft
	}

	draft.BodyPartEncoding[fieldName] = contentType
	return draft
}

func SeedDraftBodyPartEncoding(draft *model.RequestDraft, operation *model.Operation) {
	if draft == nil || operation == nil {
		return
	}
	if draft.BodyPartEncoding == nil {
		draft.BodyPartEncoding = make(map[string]string)
	}

	fields := multipartBodyEncodingFields(operation, draft)
	for name, encoding := range fields {
		if strings.TrimSpace(name) == "" {
			continue
		}
		if _, exists := draft.BodyPartEncoding[name]; exists {
			continue
		}
		if contentType := strings.TrimSpace(encoding.ContentType); contentType != "" {
			draft.BodyPartEncoding[name] = contentType
		}
	}
}

func multipartBodyEncodingFields(operation *model.Operation, draft *model.RequestDraft) map[string]model.MediaTypeEncoding {
	if operation == nil || operation.RequestBody == nil || len(operation.RequestBody.Content) == 0 {
		return nil
	}
	mediaType := bodyMediaType(operation, draft)
	if mediaType != "multipart/form-data" {
		return nil
	}

	for _, content := range operation.RequestBody.Content {
		if content.MediaType != mediaType || len(content.Encoding) == 0 {
			continue
		}
		fields := make(map[string]model.MediaTypeEncoding, len(content.Encoding))
		for name, encoding := range content.Encoding {
			fields[name] = encoding
		}
		return fields
	}

	return nil
}

func multipartBodyField(operation *model.Operation, draft *model.RequestDraft, fieldName string) (model.Parameter, bool) {
	for _, field := range ProjectBodyFieldParameters(operation, draft) {
		if field.In == model.ParameterLocationForm && field.Name == fieldName {
			return field, true
		}
	}

	return model.Parameter{}, false
}
