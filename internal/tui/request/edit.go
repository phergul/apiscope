package request

import (
	"github.com/phergul/apiscope/internal/app"
	"github.com/phergul/apiscope/internal/model"
	"github.com/phergul/apiscope/internal/tui/describe"
)

type EditStart struct {
	Kind               model.RequestEditKind
	Target             string
	Buffer             string
	FocusField         bool
	FocusBody          bool
	ResetScroll        bool
	CycleServerURL     bool
	CycleBodyMediaType bool
}

// StartEdit resolves the edit action for the active request row.
func StartEdit(
	selected *model.Operation,
	draft *model.RequestDraft,
	rows []RowDescriptor,
	activeRow int,
	securitySchemes map[string]model.SecurityScheme,
	authState map[string]model.AuthValue,
) EditStart {
	if selected == nil || len(rows) == 0 {
		return EditStart{}
	}

	row := rows[ClampActiveRow(activeRow, len(rows))]
	switch row.Kind {
	case RowKindServer:
		return EditStart{CycleServerURL: true}
	case RowKindParameter:
		if !row.Editable || row.Parameter == nil {
			return EditStart{}
		}
		return EditStart{
			Kind:       model.RequestEditKindField,
			Target:     row.ID,
			Buffer:     DraftParameterValue(draft, *row.Parameter),
			FocusField: true,
		}
	case RowKindAuthField:
		scheme, ok := lookupSecurityScheme(securitySchemes, row.AuthSchemeName)
		if !ok || !row.Editable {
			return EditStart{}
		}
		return EditStart{
			Kind:       model.RequestEditKindField,
			Target:     row.ID,
			Buffer:     app.AuthFieldValue(authState[scheme.Name], row.AuthField),
			FocusField: true,
		}
	case RowKindBodyMediaType:
		return EditStart{CycleBodyMediaType: true}
	case RowKindBodyText:
		buffer := ""
		if draft != nil {
			buffer = draft.BodyRaw
		}
		return EditStart{
			Kind:        model.RequestEditKindBody,
			Target:      row.ID,
			Buffer:      buffer,
			FocusBody:   true,
			ResetScroll: true,
		}
	default:
		return EditStart{}
	}
}

// SaveEdit persists the current request editor buffer back into session state.
func SaveEdit(
	session *model.SessionState,
	selected *model.Operation,
	rows []RowDescriptor,
	activeRow int,
	kind model.RequestEditKind,
	buffer string,
	securitySchemes map[string]model.SecurityScheme,
) bool {
	if session == nil || selected == nil || len(rows) == 0 {
		return false
	}

	row := rows[ClampActiveRow(activeRow, len(rows))]
	switch kind {
	case model.RequestEditKindField:
		if row.Parameter != nil {
			app.SetDraftParameter(session, selected, *row.Parameter, buffer)
		}
		if row.Kind == RowKindAuthField {
			scheme, ok := lookupSecurityScheme(securitySchemes, row.AuthSchemeName)
			if ok {
				app.SetAuthField(session, scheme, row.AuthField, buffer)
			}
		}
	case model.RequestEditKindBody:
		app.SetDraftBodyRaw(session, selected, buffer)
	}

	return true
}

// lookupSecurityScheme resolves one named security scheme from the spec map.
func lookupSecurityScheme(securitySchemes map[string]model.SecurityScheme, schemeName string) (model.SecurityScheme, bool) {
	if schemeName == "" {
		return model.SecurityScheme{}, false
	}

	scheme, ok := securitySchemes[schemeName]
	return scheme, ok
}

// CycleServerURL advances the selected top-level spec server for the running session.
func CycleServerURL(session *model.SessionState, servers []model.Server) bool {
	return app.CycleSelectedServer(session, servers)
}

// CycleBodyMediaType advances the selected request-body media type for the current draft.
func CycleBodyMediaType(session *model.SessionState, selected *model.Operation) bool {
	if session == nil || selected == nil || selected.RequestBody == nil || len(selected.RequestBody.Content) == 0 {
		return false
	}

	draft := app.EnsureRequestDraft(session, selected)
	if draft == nil {
		return false
	}

	mediaTypes := describe.MediaTypesForContent(selected.RequestBody.Content)
	if len(mediaTypes) == 0 {
		return false
	}

	currentIndex := 0
	for index, mediaType := range mediaTypes {
		if mediaType == draft.BodyMediaType {
			currentIndex = index
			break
		}
	}

	nextIndex := (currentIndex + 1) % len(mediaTypes)
	app.SetDraftBodyMediaType(session, selected, mediaTypes[nextIndex])
	return true
}
