package tui

import (
	"slices"
	"strings"

	"github.com/phergul/apiscope/internal/model"
	"github.com/phergul/apiscope/internal/tui/panes"
)

type paneView struct {
	Title   string
	Body    string
	Focused bool
}

func (m *Model) operationsPaneContent() string {
	return panes.RenderOperations(m.projectOperationsPane())
}

func (m *Model) detailsPaneContent() string {
	return panes.RenderDetails(m.projectDetailsPane())
}

func (m *Model) requestPaneContent() string {
	return panes.RenderRequest(m.projectRequestPane())
}

func (m *Model) responsePaneContent() string {
	return panes.RenderResponse(m.projectResponsePane())
}

func (m *Model) paneView(pane model.FocusedPane) paneView {
	switch pane {
	case model.FocusedPaneDetails:
		return paneView{
			Title:   "2 Details",
			Body:    m.detailsPaneContent(),
			Focused: m.viewState.FocusedPane == pane,
		}
	case model.FocusedPaneRequest:
		return paneView{
			Title:   "3 Request",
			Body:    m.requestPaneContent(),
			Focused: m.viewState.FocusedPane == pane,
		}
	case model.FocusedPaneResponse:
		return paneView{
			Title:   "4 Response",
			Body:    m.responsePaneContent(),
			Focused: m.viewState.FocusedPane == pane,
		}
	default:
		return paneView{
			Title:   "1 Operations",
			Body:    m.operationsPaneContent(),
			Focused: m.viewState.FocusedPane == pane,
		}
	}
}

func (m *Model) projectOperationsPane() panes.OperationsData {
	data := panes.OperationsData{
		LoadInFlight:  m.viewState.LoadInFlight,
		LoadFailed:    m.loadErr != nil,
		HasSpec:       m.session.Spec != nil,
		FilterText:    m.viewState.FilterText,
		FilterEditing: m.viewState.ActiveEditorMode == model.EditorModeFilter,
	}
	if m.session.Spec == nil {
		return data
	}

	data.TotalOperations = len(m.session.Spec.Operations)
	selected := m.resolvedSelectedOperation()

	for _, group := range m.groupedVisibleOperations() {
		projectedGroup := panes.OperationsGroup{Name: group.Name}
		for _, key := range group.Keys {
			operation := m.operationByKey(key)
			if operation == nil {
				continue
			}

			projectedGroup.Rows = append(projectedGroup.Rows, panes.OperationRow{
				Method:   operation.Method,
				Path:     operation.Path,
				Selected: selected != nil && operation.Key == selected.Key,
			})
		}
		data.Groups = append(data.Groups, projectedGroup)
	}

	return data
}

func (m *Model) projectDetailsPane() panes.DetailsData {
	data := panes.DetailsData{
		LoadInFlight:  m.viewState.LoadInFlight,
		LoadErrorBody: "",
		Selected:      m.resolvedSelectedOperation(),
		FilterText:    m.viewState.FilterText,
		Sections:      m.availableDetailsSectionLabels(),
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

func (m *Model) availableDetailsSectionLabels() []string {
	available := m.availableDetailsSections()
	labels := make([]string, 0, len(available))
	for _, section := range available {
		labels = append(labels, string(section))
	}

	return labels
}

func (m *Model) projectRequestPane() panes.RequestData {
	return panes.RequestData{
		LoadInFlight: m.viewState.LoadInFlight,
	}
}

func (m *Model) projectResponsePane() panes.ResponseData {
	return panes.ResponseData{
		LoadInFlight: m.viewState.LoadInFlight,
	}
}

func (m *Model) projectStatusBar() panes.StatusBarData {
	data := panes.StatusBarData{
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
	if strings.TrimSpace(m.viewState.FilterText) != "" {
		data.FilterText = m.viewState.FilterText
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

func (m *Model) operationByKey(key model.OperationKey) *model.Operation {
	if m.session.Spec == nil {
		return nil
	}

	for index := range m.session.Spec.Operations {
		operation := &m.session.Spec.Operations[index]
		if operation.Key == key {
			return operation
		}
	}

	return nil
}

func (m *Model) resolvedSelectedOperation() *model.Operation {
	if operation := m.operationByKey(m.session.SelectedOperationKey); operation != nil {
		if len(m.viewState.VisibleOperationKeys) == 0 || slices.Contains(m.viewState.VisibleOperationKeys, operation.Key) {
			return operation
		}
	}
	if len(m.viewState.VisibleOperationKeys) == 0 {
		return nil
	}

	return m.operationByKey(m.viewState.VisibleOperationKeys[0])
}

func (m *Model) effectiveSecurityRequirement(operation *model.Operation) *model.SecurityRequirement {
	if operation != nil && operation.Security != nil {
		return operation.Security
	}

	if m.session.Spec == nil {
		return nil
	}

	return m.session.Spec.Security
}

func (m *Model) resolvedDimensions() (int, int) {
	width := m.width
	height := m.height
	if width <= 0 {
		width = defaultWidth
	}
	if height <= 0 {
		height = defaultHeight
	}

	return width, height
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
