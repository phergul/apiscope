package tui

import (
	"slices"
	"strings"

	"github.com/phergul/apiscope/internal/app"
	"github.com/phergul/apiscope/internal/model"
	"github.com/phergul/apiscope/internal/tui/panes"
	"github.com/phergul/apiscope/internal/tui/widgets"
)

type paneView struct {
	Title   string
	Body    string
	Footer  string
	Focused bool
}

func (m *Model) operationsPaneContent() string {
	return panes.RenderOperations(m.projectOperationsPane())
}

func (m *Model) detailsPaneContent() string {
	return panes.RenderDetails(m.projectDetailsPane())
}

func (m *Model) detailsPaneContentForHeight(height int) string {
	width, _ := m.resolvedDimensions()
	if m.viewState.RightPaneLayoutPreset == layoutPresetWide {
		leftWidth := clampInt(int(float64(width)*0.32), 30, 40)
		leftWidth = minInt(leftWidth, width-20)
		width = maxInt(width-leftWidth, 20)
	}

	return m.detailsPaneContentForSize(width, height)
}

func (m *Model) detailsPaneContentForSize(width, height int) string {
	data := m.projectDetailsPane()
	if data.LoadInFlight || strings.TrimSpace(data.LoadErrorBody) != "" || data.Selected == nil {
		return panes.RenderDetails(data)
	}

	visibleLines := maxInt(height-6, 1)
	contentWidth := maxInt(width-4, 1)
	viewport := widgets.NewViewport(contentWidth, visibleLines)
	viewport.SetContent(panes.RenderActiveDetailsSectionForProjection(data))
	viewport.SetYOffset(m.viewState.DetailsScrollOffset)
	clipped := viewport.View()
	sections := panes.BuildDetailsSectionsForProjection(data)
	for index := range sections {
		if sections[index].Label == data.ActiveSection {
			sections[index].Body = clipped
			return widgets.RenderSectionView(widgets.SectionViewData{
				Sections:   sections,
				Active:     data.ActiveSection,
				EmptyState: "",
			})
		}
	}

	if len(sections) > 0 {
		sections[0].Body = clipped
	}

	return widgets.RenderSectionView(widgets.SectionViewData{
		Sections:   sections,
		Active:     data.ActiveSection,
		EmptyState: "",
	})
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
			Footer:  m.operationsPaneFooter(),
			Focused: m.viewState.FocusedPane == pane,
		}
	}
}

func (m *Model) operationsPaneFooter() string {
	m.ensureWidgetDefaults()

	if m.viewState.ActiveEditorMode != model.EditorModeFilter && strings.TrimSpace(m.viewState.FilterText) == "" {
		return ""
	}

	if m.viewState.ActiveEditorMode == model.EditorModeFilter {
		return m.filterInput.View()
	}

	return widgets.InputFrameStyle(false).
		Render("Filter: " + m.viewState.FilterText)
}

func (m *Model) projectOperationsPane() panes.OperationsData {
	data := panes.OperationsData{
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

func (m *Model) projectRequestPane() panes.RequestData {
	data := panes.RequestData{
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
	data.Edit = panes.RequestEditView{
		Kind:      string(m.viewState.RequestEditKind),
		Buffer:    m.viewState.RequestEditBuffer,
		MediaType: requestDraftBodyMediaType(selected, draft),
		View:      m.currentRequestEditorView(),
	}
	for _, row := range m.activeRequestRows() {
		data.Rows = append(data.Rows, panes.RequestRow{
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

func (m *Model) projectResponsePane() panes.ResponseData {
	data := panes.ResponseData{
		LoadInFlight: m.viewState.LoadInFlight,
	}

	selected := m.resolvedSelectedOperation()
	if selected == nil {
		data.EmptyState = "No operation selected.\nChoose an operation in pane 1 to inspect response details."
		return data
	}

	data.Sections = projectResponseSections(selected.Responses)
	data.ActiveSection = m.activeResponseSection
	if len(data.Sections) == 0 {
		data.EmptyState = "This operation does not declare any responses."
	}

	return data
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

func projectResponseSections(responses []model.ResponseSpec) []widgets.Section {
	sections := make([]widgets.Section, 0, len(responses))
	for _, response := range responses {
		sections = append(sections, widgets.Section{
			Label: response.StatusCode,
			Body:  responseSectionBody(response),
		})
	}

	return sections
}

func mediaTypesForContent(content []model.MediaTypeSpec) []string {
	mediaTypes := make([]string, 0, len(content))
	for _, item := range content {
		mediaTypes = append(mediaTypes, item.MediaType)
	}

	return mediaTypes
}

func parametersInLocation(parameters []model.Parameter, location model.ParameterLocation) []model.Parameter {
	filtered := make([]model.Parameter, 0, len(parameters))
	for _, parameter := range parameters {
		if parameter.In == location {
			filtered = append(filtered, parameter)
		}
	}

	return filtered
}

func parameterTypeHint(parameter model.Parameter) string {
	if parameter.Schema != nil {
		return schemaTypeHint(parameter.Schema)
	}
	if len(parameter.Content) > 0 {
		return "content"
	}

	return "unknown"
}

func schemaTypeHint(schema *model.Schema) string {
	if schema == nil {
		return "unknown"
	}

	parts := make([]string, 0, 2)
	if schema.Type != "" {
		parts = append(parts, schema.Type)
	}
	if schema.Format != "" {
		parts = append(parts, schema.Format)
	}
	if len(parts) > 0 {
		return strings.Join(parts, "/")
	}
	if schema.Ref != "" {
		return schema.Ref
	}
	if len(schema.OneOf) > 0 {
		return "oneOf"
	}
	if len(schema.AnyOf) > 0 {
		return "anyOf"
	}
	if len(schema.AllOf) > 0 {
		return "allOf"
	}

	return "object"
}

func responseSectionBody(response model.ResponseSpec) string {
	lines := []string{
		"Description: " + normaliseInlineText(fallbackText(response.Description, "None")),
		"Headers:",
	}
	if len(response.Headers) == 0 {
		lines = append(lines, "- none")
	} else {
		for _, header := range response.Headers {
			lines = append(lines, "- "+header.Name+" ("+parameterTypeHint(header)+")")
		}
	}
	lines = append(lines, "Media types: "+strings.Join(defaultIfEmpty(mediaTypesForContent(response.Content), "none"), ", "))

	return strings.Join(lines, "\n")
}

func normaliseInlineText(value string) string {
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return "None"
	}

	return strings.Join(fields, " ")
}

func defaultIfEmpty(values []string, fallback string) []string {
	if len(values) > 0 {
		return values
	}

	return []string{fallback}
}

func booleanRequirementLabel(required bool) string {
	if required {
		return "required"
	}

	return "optional"
}
