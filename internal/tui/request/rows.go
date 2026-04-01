package request

import (
	"fmt"
	"strings"

	"github.com/phergul/apiscope/internal/app"
	"github.com/phergul/apiscope/internal/model"
	"github.com/phergul/apiscope/internal/tui/describe"
)

type RowKind string

const (
	RowKindServer        RowKind = "server"
	RowKindAuthOption    RowKind = "auth_option"
	RowKindParameter     RowKind = "parameter"
	RowKindBodyMediaType RowKind = "body_media_type"
	RowKindBodyText      RowKind = "body_text"
	RowKindAuthField     RowKind = "auth_field"
	RowKindAuthInfo      RowKind = "auth_info"
)

type RowDescriptor struct {
	ID               string
	Kind             RowKind
	Parameter        *model.Parameter
	ServerURL        string
	AuthSchemeName   string
	AuthField        app.AuthField
	ValidationTarget string
	Label            string
	Meta             string
	Value            string
	Editable         bool
}

// ActiveRows returns the request rows for the currently active request-pane section.
func ActiveRows(
	selected *model.Operation,
	draft *model.RequestDraft,
	activeSection string,
	security *model.SecurityRequirement,
	servers []model.Server,
	selectedServerURL string,
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
	case SectionServer:
		return serverRows(servers, selectedServerURL)
	case SectionBody:
		return bodyRows(selected.RequestBody, draft)
	case SectionAuth:
		return authRows(security, securitySchemes, authState)
	default:
		return nil
	}
}

// serverRows builds the request-pane row used to switch between top-level spec servers.
func serverRows(servers []model.Server, selectedServerURL string) []RowDescriptor {
	if len(servers) <= 1 {
		return nil
	}

	selected := selectedServerURL
	selectedDescription := ""
	for _, server := range servers {
		if server.URL != selectedServerURL {
			continue
		}
		selectedDescription = server.Description
		break
	}
	if strings.TrimSpace(selected) == "" {
		selected = servers[0].URL
		selectedDescription = servers[0].Description
	}

	meta := "spec server"
	if strings.TrimSpace(selectedDescription) != "" {
		meta = selectedDescription
	}

	return []RowDescriptor{{
		ID:               "server:url",
		Kind:             RowKindServer,
		ServerURL:        selected,
		ValidationTarget: "server:url",
		Label:            "Base URL",
		Meta:             meta,
		Value:            selected,
		Editable:         true,
	}}
}

// parameterRows builds request rows for one parameter location.
func parameterRows(parameters []model.Parameter, draft *model.RequestDraft) []RowDescriptor {
	rows := make([]RowDescriptor, 0, len(parameters))
	for index := range parameters {
		parameter := &parameters[index]
		value, editable := ParameterValue(*parameter, draft)
		rows = append(rows, RowDescriptor{
			ID:               string(parameter.In) + ":" + parameter.Name,
			Kind:             RowKindParameter,
			Parameter:        parameter,
			ValidationTarget: string(parameter.In) + ":" + parameter.Name,
			Label:            parameter.Name,
			Meta:             describe.BooleanRequirementLabel(parameter.Required) + ", " + describe.ParameterTypeHint(*parameter),
			Value:            value,
			Editable:         editable,
		})
	}

	return rows
}

// ParameterValue returns the rendered parameter value and whether the row is editable.
func ParameterValue(parameter model.Parameter, draft *model.RequestDraft) (string, bool) {
	if len(parameter.Content) > 0 {
		return "content-based input", false
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
			ID:               "body:media_type",
			Kind:             RowKindBodyMediaType,
			ValidationTarget: app.ValidationTargetBodyMediaType,
			Label:            "Media type",
			Value:            mediaType,
			Editable:         len(body.Content) > 0,
		},
		{
			ID:               "body:raw",
			Kind:             RowKindBodyText,
			ValidationTarget: app.ValidationTargetBodyRaw,
			Label:            "Body",
			Value:            BodyPreview(draft),
			Editable:         true,
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
	projections := app.ProjectAuthAlternatives(requirement, securitySchemes, authState)
	if len(projections) == 0 {
		return nil
	}

	rows := make([]RowDescriptor, 0, len(projections)*3)
	for _, alternative := range projections {
		rows = append(rows, RowDescriptor{
			ID:       fmt.Sprintf("auth:option:%d", alternative.Index),
			Kind:     RowKindAuthOption,
			Label:    fmt.Sprintf("Option %d", alternative.Index+1),
			Meta:     alternativeStatusMeta(alternative),
			Value:    alternativeSummary(alternative),
			Editable: false,
		})

		for _, scheme := range alternative.Schemes {
			if !scheme.Found {
				rows = append(rows, RowDescriptor{
					ID:               scheme.ValidationTarget,
					Kind:             RowKindAuthInfo,
					AuthSchemeName:   scheme.Ref.Name,
					ValidationTarget: scheme.ValidationTarget,
					Label:            scheme.Ref.Name,
					Meta:             "missing from spec",
					Value:            "Defined in the security requirement but missing from the normalized spec.",
					Editable:         false,
				})
				continue
			}

			if scheme.UnsupportedReason != "" {
				rows = append(rows, RowDescriptor{
					ID:               scheme.ValidationTarget,
					Kind:             RowKindAuthInfo,
					AuthSchemeName:   scheme.Ref.Name,
					ValidationTarget: scheme.ValidationTarget,
					Label:            scheme.Ref.Name,
					Meta:             "unsupported auth",
					Value:            scheme.UnsupportedReason,
					Editable:         false,
				})
				continue
			}

			for _, field := range scheme.Fields {
				rows = append(rows, RowDescriptor{
					ID:               field.ValidationTarget,
					Kind:             RowKindAuthField,
					AuthSchemeName:   scheme.Ref.Name,
					AuthField:        field.Field,
					ValidationTarget: field.ValidationTarget,
					Label:            field.Label,
					Meta:             field.Meta,
					Value:            field.Summary,
					Editable:         true,
				})
			}
		}
	}

	return rows
}

func alternativeStatusMeta(alternative app.AuthAlternativeProjection) string {
	switch alternative.Status {
	case app.AuthAlternativeStatusReady:
		return "ready"
	case app.AuthAlternativeStatusUnsupported:
		return "unsupported"
	case app.AuthAlternativeStatusMissingScheme:
		return "missing scheme"
	default:
		if alternative.MissingFieldCount == 1 {
			return "missing 1 field"
		}
		return fmt.Sprintf("missing %d fields", alternative.MissingFieldCount)
	}
}

func alternativeSummary(alternative app.AuthAlternativeProjection) string {
	if len(alternative.Schemes) == 0 {
		return "No auth required"
	}

	parts := make([]string, 0, len(alternative.Schemes))
	for _, scheme := range alternative.Schemes {
		part := scheme.Ref.Name
		if len(scheme.Ref.Scopes) > 0 {
			part += " (" + strings.Join(scheme.Ref.Scopes, ", ") + ")"
		}
		parts = append(parts, part)
	}

	return strings.Join(parts, " AND ")
}
