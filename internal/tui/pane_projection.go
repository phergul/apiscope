package tui

import (
	"strings"

	"github.com/phergul/apiscope/internal/app"
	"github.com/phergul/apiscope/internal/model"
	detailsui "github.com/phergul/apiscope/internal/tui/details"
	operationsui "github.com/phergul/apiscope/internal/tui/operations"
	requestui "github.com/phergul/apiscope/internal/tui/request"
	responseui "github.com/phergul/apiscope/internal/tui/response"
	statusbarui "github.com/phergul/apiscope/internal/tui/statusbar"
)

func (m *Model) projectOperationsPane() operationsui.Data {
	data := operationsui.Data{
		LoadInFlight: m.viewState.LoadInFlight,
		LoadFailed:   m.loadErr != nil,
		HasSpec:      m.session.Spec != nil,
	}
	if m.session.Spec == nil {
		return data
	}

	data.TotalOperations = len(m.session.Spec.Operations)
	selected := m.resolvedSelectedOperation()

	for _, group := range m.groupedVisibleOperations() {
		projectedGroup := operationsui.Group{Name: group.Name}
		for _, key := range group.Keys {
			operation := m.operationByKey(key)
			if operation == nil {
				continue
			}

			projectedGroup.Rows = append(projectedGroup.Rows, operationsui.Row{
				Method:   operation.Method,
				Path:     operation.Path,
				Selected: selected != nil && operation.Key == selected.Key,
			})
		}
		data.Groups = append(data.Groups, projectedGroup)
	}

	return data
}

func (m *Model) projectDetailsPane() detailsui.Data {
	data := detailsui.Data{
		LoadInFlight:  m.viewState.LoadInFlight,
		LoadErrorBody: "",
		Selected:      m.resolvedSelectedOperation(),
		FilterText:    m.viewState.FilterText,
		ActiveSection: string(m.activeDetailsSection),
	}
	if m.loadErr != nil {
		data.LoadErrorBody = m.renderLoadErrorContent()
		return data
	}
	if data.Selected != nil {
		data.Security = m.effectiveSecurityRequirement(data.Selected)
	}
	if m.session.Spec != nil {
		data.Warnings = append([]model.SpecWarning{}, m.session.Spec.Warnings...)
	}

	return data
}

func (m *Model) projectRequestPane() requestui.Data {
	data := requestui.Data{
		LoadInFlight: m.viewState.LoadInFlight,
	}

	selected := m.resolvedSelectedOperation()
	if selected == nil {
		data.EmptyState = "No operation selected.\nChoose an operation in pane 1 to inspect request details."
		return data
	}

	draft := app.EnsureRequestDraft(&m.session, selected)
	data.Sections = m.availableRequestSections()
	data.ActiveSection = m.activeRequestSection
	data.ActiveRow = m.viewState.RequestActiveRow
	data.Edit = requestui.EditView{
		Kind:      string(m.viewState.RequestEditKind),
		Buffer:    m.viewState.RequestEditBuffer,
		MediaType: requestui.DraftBodyMediaType(selected, draft),
		View:      m.currentRequestEditorView(),
	}
	for _, row := range m.activeRequestRows() {
		data.Rows = append(data.Rows, requestui.Row{
			Label:    row.Label,
			Meta:     row.Meta,
			Value:    row.Value,
			Editable: row.Editable,
		})
	}
	if len(data.Sections) == 0 {
		data.EmptyState = "This operation does not declare request parameters, request body, or auth requirements."
	}

	return data
}

func (m *Model) projectResponsePane() responseui.Data {
	data := responseui.Data{
		LoadInFlight: m.viewState.LoadInFlight,
	}

	selected := m.resolvedSelectedOperation()
	if selected == nil {
		data.EmptyState = "No operation selected.\nChoose an operation in pane 1 to inspect response details."
		return data
	}

	data.Sections = responseui.Sections(selected.Responses)
	data.ActiveSection = m.activeResponseSection
	if len(data.Sections) == 0 {
		data.EmptyState = "This operation does not declare any responses."
	}

	return data
}

func (m *Model) projectStatusBar() statusbarui.Data {
	data := statusbarui.Data{
		Source:  m.source,
		State:   m.loadStateLabel(),
		Focus:   focusedPaneLabel(m.viewState.FocusedPane),
		HasSpec: m.session.Spec != nil,
	}

	if selected := m.resolvedSelectedOperation(); selected != nil {
		data.OperationLabel = strings.ToUpper(selected.Method) + " " + selected.Path
	}
	if m.session.Spec != nil {
		data.OperationCount = len(m.session.Spec.Operations)
		data.VisibleCount = len(m.viewState.VisibleOperationKeys)
		data.WarningCount = len(m.session.Spec.Warnings)
	}

	return data
}

func (m *Model) loadStateLabel() string {
	switch {
	case m.viewState.LoadInFlight:
		return "loading"
	case m.loadErr != nil:
		return "load failed"
	case m.session.Spec != nil:
		return "loaded"
	default:
		return "idle"
	}
}

func focusedPaneLabel(pane model.FocusedPane) string {
	switch pane {
	case model.FocusedPaneDetails:
		return "details"
	case model.FocusedPaneRequest:
		return "request"
	case model.FocusedPaneResponse:
		return "response"
	default:
		return "operations"
	}
}
