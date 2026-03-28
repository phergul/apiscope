package tui

import (
	"strconv"
	"strings"

	"github.com/phergul/apiscope/internal/app"
	"github.com/phergul/apiscope/internal/model"
	"github.com/phergul/apiscope/internal/tui/panes"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type requestRowKind string

const (
	requestRowKindParameter     requestRowKind = "parameter"
	requestRowKindBodyMediaType requestRowKind = "body_media_type"
	requestRowKindBodyText      requestRowKind = "body_text"
	requestRowKindAuth          requestRowKind = "auth"
)

type requestRowDescriptor struct {
	ID        string
	Kind      requestRowKind
	Parameter *model.Parameter
	Label     string
	Meta      string
	Value     string
	Editable  bool
}

func (m *Model) requestEditActive() bool {
	return m.viewState.ActiveEditorMode == model.EditorModeEdit &&
		m.viewState.RequestEditKind != model.RequestEditKindNone
}

func (m *Model) ensureSelectedRequestDraft() *model.RequestDraft {
	return app.EnsureRequestDraft(&m.session, m.resolvedSelectedOperation())
}

func (m *Model) activeRequestRows() []requestRowDescriptor {
	selected := m.resolvedSelectedOperation()
	if selected == nil {
		return nil
	}

	draft := app.EnsureRequestDraft(&m.session, selected)
	switch m.activeRequestSection {
	case "Path":
		return parameterRows(parametersInLocation(selected.Parameters, model.ParameterLocationPath), draft)
	case "Query":
		return parameterRows(parametersInLocation(selected.Parameters, model.ParameterLocationQuery), draft)
	case "Header":
		return parameterRows(parametersInLocation(selected.Parameters, model.ParameterLocationHeader), draft)
	case "Cookie":
		return parameterRows(parametersInLocation(selected.Parameters, model.ParameterLocationCookie), draft)
	case requestSectionBody:
		return bodyRows(selected.RequestBody, draft)
	case requestSectionAuth:
		return authRows(m.effectiveSecurityRequirement(selected))
	default:
		return nil
	}
}

func parameterRows(parameters []model.Parameter, draft *model.RequestDraft) []requestRowDescriptor {
	rows := make([]requestRowDescriptor, 0, len(parameters))
	for index := range parameters {
		parameter := &parameters[index]
		value, editable := requestParameterValue(*parameter, draft)
		rows = append(rows, requestRowDescriptor{
			ID:        string(parameter.In) + ":" + parameter.Name,
			Kind:      requestRowKindParameter,
			Parameter: parameter,
			Label:     parameter.Name,
			Meta:      booleanRequirementLabel(parameter.Required) + ", " + parameterTypeHint(*parameter),
			Value:     value,
			Editable:  editable,
		})
	}

	return rows
}

func requestParameterValue(parameter model.Parameter, draft *model.RequestDraft) (string, bool) {
	if len(parameter.Content) > 0 {
		return "<unsupported: content-based parameter>", false
	}

	value := draftParameterValue(draft, parameter)
	if value == "" {
		return "<unset>", true
	}

	return value, true
}

func draftParameterValue(draft *model.RequestDraft, parameter model.Parameter) string {
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

func bodyRows(body *model.RequestBodySpec, draft *model.RequestDraft) []requestRowDescriptor {
	if body == nil {
		return nil
	}

	mediaType := requestDraftBodyMediaType(&model.Operation{RequestBody: body}, draft)

	return []requestRowDescriptor{
		{
			ID:       "body:media_type",
			Kind:     requestRowKindBodyMediaType,
			Label:    "Media type",
			Value:    mediaType,
			Editable: len(body.Content) > 0,
		},
		{
			ID:       "body:raw",
			Kind:     requestRowKindBodyText,
			Label:    "Body",
			Value:    requestBodyPreview(draft),
			Editable: true,
		},
	}
}

func requestDraftBodyMediaType(operation *model.Operation, draft *model.RequestDraft) string {
	if draft != nil && draft.BodyMediaType != "" {
		return draft.BodyMediaType
	}
	if operation != nil && operation.RequestBody != nil && len(operation.RequestBody.Content) > 0 {
		return operation.RequestBody.Content[0].MediaType
	}

	return "none"
}

func requestBodyPreview(draft *model.RequestDraft) string {
	if draft == nil || draft.BodyRaw == "" {
		return "<empty>"
	}

	lines := strings.Split(draft.BodyRaw, "\n")
	if len(lines) == 1 {
		return draft.BodyRaw
	}

	return lines[0] + " ... (" + strconvInt(len(lines)) + " lines)"
}

func authRows(requirement *model.SecurityRequirement) []requestRowDescriptor {
	if requirement == nil || len(requirement.Alternatives) == 0 {
		return nil
	}

	rows := make([]requestRowDescriptor, 0, len(requirement.Alternatives))
	for index, alternative := range requirement.Alternatives {
		parts := make([]string, 0, len(alternative.Schemes))
		for _, scheme := range alternative.Schemes {
			part := scheme.Name
			if len(scheme.Scopes) > 0 {
				part += " (" + strings.Join(scheme.Scopes, ", ") + ")"
			}
			parts = append(parts, part)
		}
		rows = append(rows, requestRowDescriptor{
			ID:       "auth:" + strconvInt(index),
			Kind:     requestRowKindAuth,
			Label:    "Alternative " + strconvInt(index+1),
			Value:    strings.Join(parts, " AND "),
			Editable: false,
		})
	}

	return rows
}

func strconvInt(value int) string {
	return strconv.Itoa(value)
}

func (m *Model) syncActiveRequestRow() {
	rows := m.activeRequestRows()
	if len(rows) == 0 {
		m.viewState.RequestActiveRow = 0
		m.viewState.RequestScrollOffset = 0
		return
	}

	if m.viewState.RequestActiveRow < 0 {
		m.viewState.RequestActiveRow = 0
	}
	if m.viewState.RequestActiveRow >= len(rows) {
		m.viewState.RequestActiveRow = len(rows) - 1
	}
	m.ensureActiveRequestRowVisible()
}

func (m *Model) resetRequestCursorAndScroll() {
	m.viewState.RequestActiveRow = 0
	m.viewState.RequestScrollOffset = 0
}

func (m *Model) moveRequestRow(direction int) {
	rows := m.activeRequestRows()
	if len(rows) == 0 {
		m.resetRequestCursorAndScroll()
		return
	}

	target := m.viewState.RequestActiveRow + direction
	if target < 0 {
		target = 0
	}
	if target >= len(rows) {
		target = len(rows) - 1
	}

	m.viewState.RequestActiveRow = target
	m.ensureActiveRequestRowVisible()
}

func (m *Model) setRequestRowBoundary(last bool) {
	rows := m.activeRequestRows()
	if len(rows) == 0 {
		m.resetRequestCursorAndScroll()
		return
	}

	if last {
		m.viewState.RequestActiveRow = len(rows) - 1
	} else {
		m.viewState.RequestActiveRow = 0
	}
	m.ensureActiveRequestRowVisible()
}

func (m *Model) ensureActiveRequestRowVisible() {
	if m.viewState.RequestEditKind == model.RequestEditKindBody {
		return
	}

	visible := m.requestVisibleBodyLines()
	if visible <= 0 {
		visible = 1
	}
	if m.viewState.RequestActiveRow < m.viewState.RequestScrollOffset {
		m.viewState.RequestScrollOffset = m.viewState.RequestActiveRow
		return
	}
	if m.viewState.RequestActiveRow >= m.viewState.RequestScrollOffset+visible {
		m.viewState.RequestScrollOffset = m.viewState.RequestActiveRow - visible + 1
	}
}

func (m *Model) requestVisibleBodyLines() int {
	width, height := m.resolvedDimensions()
	bodyHeight := maxInt(height-lipgloss.Height(m.renderStatusBar(width)), 12)

	var paneHeight int
	if m.viewState.RightPaneLayoutPreset == layoutPresetWide {
		requestHeight, _ := m.rightPaneHeights(computeWidePaneHeights(bodyHeight))
		paneHeight = requestHeight
	} else {
		requestHeight, _ := m.rightPaneHeights(computeNarrowPaneHeights(bodyHeight))
		paneHeight = requestHeight
	}

	return maxInt(paneHeight-6, 1)
}

func (m *Model) currentRequestEditorView() string {
	m.ensureWidgetDefaults()

	switch m.viewState.RequestEditKind {
	case model.RequestEditKindField:
		return m.requestFieldInput.View()
	case model.RequestEditKindBody:
		return m.requestBodyInput.View()
	default:
		return ""
	}
}

func (m *Model) requestPaneContentForSize(width, height int) string {
	contentWidth := maxInt(width-4, 1)
	m.requestFieldInput.SetWidth(maxInt(contentWidth-18, 12))
	m.requestBodyInput.SetSize(contentWidth, maxInt(height-9, 3))

	data := m.projectRequestPane()
	if data.LoadInFlight || len(data.Sections) == 0 {
		return panes.RenderRequest(data)
	}

	visibleLines := maxInt(height-6, 1)
	if data.Edit.Kind == string(model.RequestEditKindBody) {
		return panes.RenderRequest(data)
	}

	if len(data.Rows) == 0 {
		return panes.RenderRequest(data)
	}

	offset := m.viewState.RequestScrollOffset
	if offset < 0 {
		offset = 0
	}
	if offset >= len(data.Rows) {
		offset = len(data.Rows) - 1
	}
	end := offset + visibleLines
	if end > len(data.Rows) {
		end = len(data.Rows)
	}

	data.ActiveRow = m.viewState.RequestActiveRow - offset
	data.Rows = data.Rows[offset:end]
	return panes.RenderRequest(data)
}

func (m *Model) beginRequestEdit() {
	rows := m.activeRequestRows()
	if len(rows) == 0 {
		return
	}

	row := rows[clampInt(m.viewState.RequestActiveRow, 0, len(rows)-1)]
	selected := m.resolvedSelectedOperation()
	if selected == nil {
		return
	}

	switch row.Kind {
	case requestRowKindParameter:
		if !row.Editable || row.Parameter == nil {
			return
		}
		m.viewState.ActiveEditorMode = model.EditorModeEdit
		m.viewState.RequestEditKind = model.RequestEditKindField
		m.viewState.RequestEditTarget = row.ID
		m.viewState.RequestEditBuffer = draftParameterValue(app.EnsureRequestDraft(&m.session, selected), *row.Parameter)
		m.requestFieldInput.SetValue(m.viewState.RequestEditBuffer)
		m.requestFieldInput.Focus()
	case requestRowKindBodyMediaType:
		m.cycleRequestBodyMediaType()
	case requestRowKindBodyText:
		m.viewState.ActiveEditorMode = model.EditorModeEdit
		m.viewState.RequestEditKind = model.RequestEditKindBody
		m.viewState.RequestEditTarget = row.ID
		draft := app.EnsureRequestDraft(&m.session, selected)
		if draft != nil {
			m.viewState.RequestEditBuffer = draft.BodyRaw
		} else {
			m.viewState.RequestEditBuffer = ""
		}
		m.viewState.RequestScrollOffset = 0
		m.requestBodyInput.SetValue(m.viewState.RequestEditBuffer)
		m.requestBodyInput.Focus()
	}
}

func (m *Model) saveRequestEdit() {
	selected := m.resolvedSelectedOperation()
	if selected == nil {
		m.cancelRequestEdit()
		return
	}

	rows := m.activeRequestRows()
	if len(rows) == 0 {
		m.cancelRequestEdit()
		return
	}
	row := rows[clampInt(m.viewState.RequestActiveRow, 0, len(rows)-1)]

	switch m.viewState.RequestEditKind {
	case model.RequestEditKindField:
		if row.Parameter != nil {
			app.SetDraftParameter(&m.session, selected, *row.Parameter, m.viewState.RequestEditBuffer)
		}
	case model.RequestEditKindBody:
		app.SetDraftBodyRaw(&m.session, selected, m.viewState.RequestEditBuffer)
	}

	m.finishRequestEdit()
}

func (m *Model) cancelRequestEdit() {
	m.finishRequestEdit()
}

func (m *Model) finishRequestEdit() {
	m.requestFieldInput.Blur()
	m.requestBodyInput.Blur()
	m.viewState.ActiveEditorMode = model.EditorModeBrowse
	m.viewState.RequestEditKind = model.RequestEditKindNone
	m.viewState.RequestEditTarget = ""
	m.viewState.RequestEditBuffer = ""
	m.syncActiveRequestRow()
}

func (m *Model) cycleRequestBodyMediaType() {
	selected := m.resolvedSelectedOperation()
	if selected == nil || selected.RequestBody == nil || len(selected.RequestBody.Content) == 0 {
		return
	}

	draft := app.EnsureRequestDraft(&m.session, selected)
	if draft == nil {
		return
	}

	mediaTypes := mediaTypesForContent(selected.RequestBody.Content)
	if len(mediaTypes) == 0 {
		return
	}

	currentIndex := 0
	for index, mediaType := range mediaTypes {
		if mediaType == draft.BodyMediaType {
			currentIndex = index
			break
		}
	}

	nextIndex := (currentIndex + 1) % len(mediaTypes)
	app.SetDraftBodyMediaType(&m.session, selected, mediaTypes[nextIndex])
}

func (m *Model) scrollRequestEditBy(delta int) {
	if m.viewState.RequestEditKind != model.RequestEditKindBody {
		return
	}

	maxOffset := m.maxRequestEditScrollOffset()
	target := m.viewState.RequestScrollOffset + delta
	if target < 0 {
		target = 0
	}
	if target > maxOffset {
		target = maxOffset
	}

	m.viewState.RequestScrollOffset = target
}

func (m *Model) scrollRequestEditToBoundary(last bool) {
	if m.viewState.RequestEditKind != model.RequestEditKindBody {
		return
	}

	if last {
		m.viewState.RequestScrollOffset = m.maxRequestEditScrollOffset()
		return
	}

	m.viewState.RequestScrollOffset = 0
}

func (m *Model) maxRequestEditScrollOffset() int {
	data := m.projectRequestPane()
	lines := len(splitLines(panes.RenderActiveRequestSection(data)))
	visible := m.requestVisibleBodyLines()
	if lines <= visible {
		return 0
	}

	return lines - visible
}

func (m *Model) updateRequestEditKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc":
		m.cancelRequestEdit()
	case "ctrl+s":
		if m.viewState.RequestEditKind == model.RequestEditKindBody {
			m.saveRequestEdit()
		}
	case "enter":
		if m.viewState.RequestEditKind == model.RequestEditKindField {
			m.saveRequestEdit()
			return m, nil
		}
	}

	switch m.viewState.RequestEditKind {
	case model.RequestEditKindField:
		cmd := m.requestFieldInput.Update(msg)
		m.viewState.RequestEditBuffer = m.requestFieldInput.Value()
		return m, cmd
	case model.RequestEditKindBody:
		cmd := m.requestBodyInput.Update(msg)
		m.viewState.RequestEditBuffer = m.requestBodyInput.Value()
		return m, cmd
	}

	return m, nil
}
