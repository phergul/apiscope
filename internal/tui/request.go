package tui

import (
	"context"
	"strings"

	"github.com/phergul/apiscope/internal/app"
	"github.com/phergul/apiscope/internal/model"
	requestui "github.com/phergul/apiscope/internal/tui/request"
	responseui "github.com/phergul/apiscope/internal/tui/response"
	"github.com/phergul/apiscope/internal/tui/widgets"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m *Model) ensureSelectedRequestDraft() *model.RequestDraft {
	return app.EnsureRequestDraft(&m.session, m.resolvedSelectedOperation())
}

func (m *Model) activeRequestRows() []requestui.RowDescriptor {
	selected := m.resolvedSelectedOperation()
	if selected == nil {
		return nil
	}

	return requestui.ActiveRows(
		selected,
		app.EnsureRequestDraft(&m.session, selected),
		m.activeRequestSection,
		m.effectiveSecurityRequirement(selected),
	)
}

func (m *Model) syncActiveRequestRow() {
	rows := m.activeRequestRows()
	if len(rows) == 0 {
		m.viewState.RequestActiveRow = 0
		m.viewState.RequestScrollOffset = 0
		return
	}

	m.viewState.RequestActiveRow = requestui.ClampActiveRow(m.viewState.RequestActiveRow, len(rows))
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

	m.viewState.RequestActiveRow = requestui.MoveActiveRow(m.viewState.RequestActiveRow, len(rows), direction)
	m.ensureActiveRequestRowVisible()
}

func (m *Model) setRequestRowBoundary(last bool) {
	rows := m.activeRequestRows()
	if len(rows) == 0 {
		m.resetRequestCursorAndScroll()
		return
	}

	m.viewState.RequestActiveRow = requestui.BoundaryActiveRow(len(rows), last)
	m.ensureActiveRequestRowVisible()
}

func (m *Model) ensureActiveRequestRowVisible() {
	if m.viewState.RequestEditKind == model.RequestEditKindBody {
		return
	}

	m.viewState.RequestScrollOffset = requestui.EnsureVisibleOffset(
		m.viewState.RequestActiveRow,
		m.viewState.RequestScrollOffset,
		m.requestVisibleBodyLines(),
	)
}

func (m *Model) requestVisibleBodyLines() int {
	width, height := m.resolvedDimensions()
	bodyHeight := max(height-lipgloss.Height(m.renderStatusBar(width)), 12)

	var paneHeight int
	if m.viewState.RightPaneLayoutPreset == layoutPresetWide {
		requestHeight, _ := m.rightPaneHeights(computeWidePaneHeights(bodyHeight))
		paneHeight = requestHeight
	} else {
		requestHeight, _ := m.rightPaneHeights(computeNarrowPaneHeights(bodyHeight))
		paneHeight = requestHeight
	}

	return max(paneHeight-6, 1)
}

func (m *Model) currentRequestEditorView() string {
	m.ensureWidgetDefaults()

	switch m.viewState.RequestEditKind {
	case model.RequestEditKindField:
		return m.requestFieldInput.BareView()
	case model.RequestEditKindBody:
		return m.requestBodyInput.BareView()
	default:
		return ""
	}
}

func (m *Model) currentRequestEditorState(selected *model.Operation, draft *model.RequestDraft) requestui.EditorState {
	state := requestui.EditorState{
		Kind:   string(m.viewState.RequestEditKind),
		Buffer: m.viewState.RequestEditBuffer,
		View:   m.currentRequestEditorView(),
	}

	switch m.viewState.RequestEditKind {
	case model.RequestEditKindBody:
		state.BodyMediaType = requestui.DraftBodyMediaType(selected, draft)
	case model.RequestEditKindField:
		if row := m.currentRequestEditRow(); row != nil {
			state.ActiveRowLabel = row.Label
			state.ActiveRowMeta = row.Meta
		}
	}

	return state
}

func (m *Model) currentRequestHelpOverlay() helpOverlayView {
	if !m.requestEditActive() {
		return helpOverlayView{}
	}

	help := requestui.BuildHelpView(m.currentRequestEditorState(m.resolvedSelectedOperation(), m.ensureSelectedRequestDraft()))
	overlay := helpOverlayView{Hint: help.Hint}
	if !m.requestEditHelpOpen || strings.TrimSpace(help.Body) == "" {
		return overlay
	}

	overlay.Title = help.Title
	overlay.Body = help.Body
	return overlay
}

func (m *Model) requestPaneContentForSize(width, height int) string {
	contentWidth := max(width-4, 1)
	fieldPopupWidth := min(max(contentWidth-10, 24), 64)
	bodyPopupWidth := min(max(contentWidth-8, 28), 84)
	m.requestFieldInput.SetWidth(max(fieldPopupWidth-4, 12))
	m.requestBodyInput.SetSize(max(bodyPopupWidth-4, 20), max(min(height-10, 12), 4))

	data := m.projectRequestPane()
	data.ContentWidth = contentWidth
	data.ContentHeight = max(height-6, 1)
	if data.LoadInFlight || len(data.Sections) == 0 {
		return requestui.Render(data)
	}

	visibleLines := max(height-6, 1)
	return requestui.Render(requestui.VisibleData(data, m.viewState.RequestScrollOffset, visibleLines))
}

func (m *Model) requestEditActive() bool {
	return m.viewState.ActiveEditorMode == model.EditorModeEdit &&
		m.viewState.RequestEditKind != model.RequestEditKindNone
}

func (m *Model) beginRequestEdit() {
	m.clearRequestValidation()
	selected := m.resolvedSelectedOperation()
	start := requestui.StartEdit(
		selected,
		app.EnsureRequestDraft(&m.session, selected),
		m.activeRequestRows(),
		m.viewState.RequestActiveRow,
	)
	if start.CycleBodyMediaType {
		requestui.CycleBodyMediaType(&m.session, selected)
		return
	}
	if start.Kind == model.RequestEditKindNone {
		return
	}

	m.viewState.ActiveEditorMode = model.EditorModeEdit
	m.viewState.RequestEditKind = start.Kind
	m.viewState.RequestEditTarget = start.Target
	m.viewState.RequestEditBuffer = start.Buffer
	m.requestEditHelpOpen = false
	if start.ResetScroll {
		m.viewState.RequestScrollOffset = 0
	}
	if start.FocusField {
		m.requestFieldInput.SetValue(start.Buffer)
		m.requestFieldInput.Focus()
	}
	if start.FocusBody {
		m.requestBodyInput.SetValue(start.Buffer)
		m.requestBodyInput.Focus()
	}
}

func (m *Model) saveRequestEdit() {
	if requestui.SaveEdit(
		&m.session,
		m.resolvedSelectedOperation(),
		m.activeRequestRows(),
		m.viewState.RequestActiveRow,
		m.viewState.RequestEditKind,
		m.viewState.RequestEditBuffer,
	) {
		m.finishRequestEdit()
		return
	}

	m.cancelRequestEdit()
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
	m.requestEditHelpOpen = false
	m.clearRequestValidation()
	m.syncActiveRequestRow()
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
	lines := len(splitLines(requestui.RenderActiveSection(data)))
	visible := m.requestVisibleBodyLines()
	if lines <= visible {
		return 0
	}

	return lines - visible
}

func (m *Model) updateRequestEditKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.requestEditHelpOpen {
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "?":
			m.requestEditHelpOpen = false
			return m, nil
		default:
			m.requestEditHelpOpen = false
			return m, nil
		}
	}

	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc":
		m.cancelRequestEdit()
	case "?":
		m.requestEditHelpOpen = !m.requestEditHelpOpen
		return m, nil
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

func (m *Model) clearRequestValidation() {
	m.requestValidation = app.RequestValidationResult{}
}

func (m *Model) executeCurrentRequest() tea.Cmd {
	selected := m.resolvedSelectedOperation()
	if selected == nil || m.viewState.LoadInFlight || m.viewState.ExecuteInFlight {
		return nil
	}

	validation := app.ValidateRequest(selected, app.EnsureRequestDraft(&m.session, selected))
	if validation.HasIssues() {
		m.requestValidation = validation
		if issue, ok := validation.FirstIssue(); ok {
			m.activeRequestSection = issue.Section
			rows := m.activeRequestRows()
			for index, row := range rows {
				if row.ID == issue.Target {
					m.viewState.RequestActiveRow = index
					break
				}
			}
			m.ensureActiveRequestRowVisible()
		}
		m.viewState.FocusedPane = model.FocusedPaneRequest
		m.viewState.ExpandedRightPane = model.FocusedPaneRequest
		m.viewState.Notice = "request validation failed"
		return nil
	}

	m.clearRequestValidation()
	requestID := m.viewState.ActiveExecuteRequestID + 1
	m.session.ActiveExecRequestID = requestID
	m.viewState.ActiveExecuteRequestID = requestID
	m.viewState.ExecuteInFlight = true
	m.viewState.Notice = "executing request"

	service := m.service
	session := m.session

	return func() tea.Msg {
		return executeFinishedMsg{
			requestID: requestID,
			result:    service.ExecuteCurrent(context.Background(), session),
		}
	}
}

func (m *Model) responsePaneContentForSize(width, height int) string {
	data := m.projectResponsePane()
	if data.LoadInFlight || len(data.Sections) == 0 {
		return responseui.Render(data)
	}

	visibleLines := max(height-6, 1)
	contentWidth := max(width-4, 1)
	viewport := widgets.NewViewport(contentWidth, visibleLines)
	viewport.SetContent(responseui.ActiveSectionBody(data.Sections, data.ActiveSection))
	viewport.SetYOffset(m.viewState.ResponseScrollOffset)
	clipped := viewport.View()

	sections := append([]widgets.Section(nil), data.Sections...)
	active := data.ActiveSection
	if active == "" && len(sections) > 0 {
		active = sections[0].Label
	}
	for index := range sections {
		if sections[index].Label == active {
			sections[index].Body = clipped
			break
		}
	}

	return responseui.Render(responseui.Data{
		LoadInFlight:  data.LoadInFlight,
		Sections:      sections,
		ActiveSection: active,
		EmptyState:    data.EmptyState,
	})
}

func (m *Model) currentRequestEditRow() *requestui.RowDescriptor {
	rows := m.activeRequestRows()
	if len(rows) == 0 {
		return nil
	}

	index := m.viewState.RequestActiveRow
	if index < 0 {
		index = 0
	}
	if index >= len(rows) {
		index = len(rows) - 1
	}

	return &rows[index]
}
