package request

import (
	"strings"

	"github.com/phergul/apiscope/internal/app"
	"github.com/phergul/apiscope/internal/model"
	"github.com/phergul/apiscope/internal/tui/describe"
)

type RowKind string

const (
	RowKindParameter     RowKind = "parameter"
	RowKindBodyMediaType RowKind = "body_media_type"
	RowKindBodyText      RowKind = "body_text"
	RowKindAuthField     RowKind = "auth_field"
	RowKindAuthInfo      RowKind = "auth_info"
)

type RowDescriptor struct {
	ID             string
	Kind           RowKind
	Parameter      *model.Parameter
	AuthSchemeName string
	AuthField      app.AuthField
	Label          string
	Meta           string
	Value          string
	Editable       bool
}

// ActiveRows returns the request rows for the currently active request-pane section.
func ActiveRows(
	selected *model.Operation,
	draft *model.RequestDraft,
	activeSection string,
	security *model.SecurityRequirement,
	securitySchemes map[string]model.SecurityScheme,
	authState map[string]model.AuthValue,
) []RowDescriptor {
	if selected == nil {
		return nil
	}

	switch activeSection {
	case "Path":
		return parameterRows(describe.ParametersInLocation(selected.Parameters, model.ParameterLocationPath), draft)
	case "Query":
		return parameterRows(describe.ParametersInLocation(selected.Parameters, model.ParameterLocationQuery), draft)
	case "Header":
		return parameterRows(describe.ParametersInLocation(selected.Parameters, model.ParameterLocationHeader), draft)
	case "Cookie":
		return parameterRows(describe.ParametersInLocation(selected.Parameters, model.ParameterLocationCookie), draft)
	case SectionBody:
		return bodyRows(selected.RequestBody, draft)
	case SectionAuth:
		return authRows(security, securitySchemes, authState)
	default:
		return nil
	}
}

// parameterRows builds request rows for one parameter location.
func parameterRows(parameters []model.Parameter, draft *model.RequestDraft) []RowDescriptor {
	rows := make([]RowDescriptor, 0, len(parameters))
	for index := range parameters {
		parameter := &parameters[index]
		value, editable := ParameterValue(*parameter, draft)
		rows = append(rows, RowDescriptor{
			ID:        string(parameter.In) + ":" + parameter.Name,
			Kind:      RowKindParameter,
			Parameter: parameter,
			Label:     parameter.Name,
			Meta:      describe.BooleanRequirementLabel(parameter.Required) + ", " + describe.ParameterTypeHint(*parameter),
			Value:     value,
			Editable:  editable,
		})
	}

	return rows
}

// ParameterValue returns the rendered parameter value and whether the row is editable.
func ParameterValue(parameter model.Parameter, draft *model.RequestDraft) (string, bool) {
	if len(parameter.Content) > 0 {
		return "<unsupported: content-based parameter>", false
	}

	value := DraftParameterValue(draft, parameter)
	if value == "" {
		return "<unset>", true
	}

	return value, true
}

// DraftParameterValue returns the raw stored draft value for the provided parameter.
func DraftParameterValue(draft *model.RequestDraft, parameter model.Parameter) string {
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

// bodyRows returns the editable request-body rows for the active request section.
func bodyRows(body *model.RequestBodySpec, draft *model.RequestDraft) []RowDescriptor {
	if body == nil {
		return nil
	}

	mediaType := DraftBodyMediaType(&model.Operation{RequestBody: body}, draft)

	return []RowDescriptor{
		{
			ID:       "body:media_type",
			Kind:     RowKindBodyMediaType,
			Label:    "Media type",
			Value:    mediaType,
			Editable: len(body.Content) > 0,
		},
		{
			ID:       "body:raw",
			Kind:     RowKindBodyText,
			Label:    "Body",
			Value:    BodyPreview(draft),
			Editable: true,
		},
	}
}

func DraftBodyMediaType(operation *model.Operation, draft *model.RequestDraft) string {
	if draft != nil && draft.BodyMediaType != "" {
		return draft.BodyMediaType
	}
	if operation != nil && operation.RequestBody != nil && len(operation.RequestBody.Content) > 0 {
		return operation.RequestBody.Content[0].MediaType
	}

	return "none"
}

// BodyPreview returns the full request body text used in the request-pane preview row.
func BodyPreview(draft *model.RequestDraft) string {
	if draft == nil || strings.TrimSpace(draft.BodyRaw) == "" {
		return "<empty>"
	}

	return strings.TrimRight(draft.BodyRaw, "\n")
}

// authRows builds editable auth rows for the effective security requirement.
func authRows(requirement *model.SecurityRequirement, securitySchemes map[string]model.SecurityScheme, authState map[string]model.AuthValue) []RowDescriptor {
	if requirement == nil || len(requirement.Alternatives) == 0 {
		return nil
	}

	seen := make(map[string]bool)
	rows := make([]RowDescriptor, 0, len(requirement.Alternatives)*2)
	for _, alternative := range requirement.Alternatives {
		for _, ref := range alternative.Schemes {
			if seen[ref.Name] {
				continue
			}
			seen[ref.Name] = true

			scheme, ok := securitySchemes[ref.Name]
			if !ok {
				rows = append(rows, RowDescriptor{
					ID:             app.AuthFieldTarget(ref.Name, ""),
					Kind:           RowKindAuthInfo,
					AuthSchemeName: ref.Name,
					Label:          ref.Name,
					Meta:           "missing security scheme",
					Value:          "<missing from spec>",
					Editable:       false,
				})
				continue
			}

			fields := app.SupportedAuthFields(scheme)
			if len(fields) == 0 {
				rows = append(rows, RowDescriptor{
					ID:             app.AuthFieldTarget(ref.Name, ""),
					Kind:           RowKindAuthInfo,
					AuthSchemeName: ref.Name,
					Label:          ref.Name,
					Meta:           "unsupported auth",
					Value:          "<unsupported auth type>",
					Editable:       false,
				})
				continue
			}

			value := authState[ref.Name]
			for _, field := range fields {
				rows = append(rows, RowDescriptor{
					ID:             app.AuthFieldTarget(ref.Name, field),
					Kind:           RowKindAuthField,
					AuthSchemeName: ref.Name,
					AuthField:      field,
					Label:          app.AuthFieldLabel(scheme, field),
					Meta:           app.AuthFieldMeta(scheme, field),
					Value:          app.AuthFieldSummary(value, field),
					Editable:       true,
				})
			}
		}
	}

	return rows
}
