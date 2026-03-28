package request

import (
	"strconv"
	"strings"

	"github.com/phergul/apiscope/internal/model"
	"github.com/phergul/apiscope/internal/tui/describe"
)

type RowKind string

const (
	RowKindParameter     RowKind = "parameter"
	RowKindBodyMediaType RowKind = "body_media_type"
	RowKindBodyText      RowKind = "body_text"
	RowKindAuth          RowKind = "auth"
)

type RowDescriptor struct {
	ID        string
	Kind      RowKind
	Parameter *model.Parameter
	Label     string
	Meta      string
	Value     string
	Editable  bool
}

func ActiveRows(selected *model.Operation, draft *model.RequestDraft, activeSection string, security *model.SecurityRequirement) []RowDescriptor {
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
		return authRows(security)
	default:
		return nil
	}
}

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

func BodyPreview(draft *model.RequestDraft) string {
	if draft == nil || draft.BodyRaw == "" {
		return "<empty>"
	}

	lines := strings.Split(draft.BodyRaw, "\n")
	if len(lines) == 1 {
		return draft.BodyRaw
	}

	return lines[0] + " ... (" + strconv.Itoa(len(lines)) + " lines)"
}

func authRows(requirement *model.SecurityRequirement) []RowDescriptor {
	if requirement == nil || len(requirement.Alternatives) == 0 {
		return nil
	}

	rows := make([]RowDescriptor, 0, len(requirement.Alternatives))
	for index, alternative := range requirement.Alternatives {
		parts := make([]string, 0, len(alternative.Schemes))
		for _, scheme := range alternative.Schemes {
			part := scheme.Name
			if len(scheme.Scopes) > 0 {
				part += " (" + strings.Join(scheme.Scopes, ", ") + ")"
			}
			parts = append(parts, part)
		}
		rows = append(rows, RowDescriptor{
			ID:       "auth:" + strconv.Itoa(index),
			Kind:     RowKindAuth,
			Label:    "Alternative " + strconv.Itoa(index+1),
			Value:    strings.Join(parts, " AND "),
			Editable: false,
		})
	}

	return rows
}
