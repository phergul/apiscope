package tui

import (
	"context"

	"github.com/phergul/apiscope/internal/app"
	"github.com/phergul/apiscope/internal/model"
	requestui "github.com/phergul/apiscope/internal/tui/request"

	tea "github.com/charmbracelet/bubbletea"
)

// ensureSelectedRequestDraft returns the current request draft for the selected operation.
func (m *Model) ensureSelectedRequestDraft() *model.RequestDraft {
	return app.EnsureRequestDraft(&m.session, m.resolvedSelectedOperation())
}

// activeRequestRows returns the request rows for the active request section.
func (m *Model) activeRequestRows() []requestui.RowDescriptor {
	selected := m.resolvedSelectedOperation()
	if selected == nil {
		return nil
	}

	return requestui.ActiveRows(
		selected,
		app.EnsureRequestDraft(&m.session, selected),
		m.panes.activeRequestSection,
		m.effectiveSecurityRequirement(selected),
		m.topLevelServers(),
		m.session.SelectedServerURL,
		m.securitySchemes(),
		m.session.AuthState,
	)
}

// requestRowState returns the current request row selection and scroll state.
func (m *Model) requestRowState() requestui.RowState {
	return requestui.RowState{
		ActiveRow:    m.viewState.RequestActiveRow,
		ScrollOffset: m.viewState.RequestScrollOffset,
	}
}

// applyRequestRowState writes the request row state back to the root view state.
func (m *Model) applyRequestRowState(state requestui.RowState) {
	m.viewState.RequestActiveRow = state.ActiveRow
	m.viewState.RequestScrollOffset = state.ScrollOffset
}

// syncActiveRequestRow clamps the current request row state and keeps the active row visible.
func (m *Model) syncActiveRequestRow() {
	m.applyRequestRowState(requestui.SyncRowState(
		m.activeRequestRows(),
		m.requestRowState(),
		m.viewState.RequestEditKind,
		m.requestVisibleBodyLines(),
	))
}

// resetRequestCursorAndScroll resets the request row selection state.
func (m *Model) resetRequestCursorAndScroll() {
	m.applyRequestRowState(requestui.ResetRowState())
}

// moveRequestRow moves the active request row by the given direction.
func (m *Model) moveRequestRow(direction int) {
	m.applyRequestRowState(requestui.MoveRowState(
		m.activeRequestRows(),
		m.requestRowState(),
		direction,
		m.viewState.RequestEditKind,
		m.requestVisibleBodyLines(),
	))
}

// setRequestRowBoundary moves the active request row to the first or last row.
func (m *Model) setRequestRowBoundary(last bool) {
	m.applyRequestRowState(requestui.BoundaryRowState(
		m.activeRequestRows(),
		m.requestRowState(),
		last,
		m.viewState.RequestEditKind,
		m.requestVisibleBodyLines(),
	))
}

// ensureActiveRequestRowVisible syncs the request row state against the current viewport size.
func (m *Model) ensureActiveRequestRowVisible() {
	m.applyRequestRowState(requestui.SyncRowState(
		m.activeRequestRows(),
		m.requestRowState(),
		m.viewState.RequestEditKind,
		m.requestVisibleBodyLines(),
	))
}

// requestVisibleBodyLines returns the visible body height for the request pane content area.
func (m *Model) requestVisibleBodyLines() int {
	width, height := m.resolvedDimensions()
	bodyHeight := max(height-m.statusBarHeight(width), 12)

	var paneHeight int
	if m.viewState.RightPaneLayoutPreset == layoutPresetWide {
		requestHeight, _ := m.rightPaneHeights(computeWidePaneHeights(bodyHeight))
		paneHeight = requestHeight
	} else {
		requestHeight, _ := m.rightPaneHeights(computeNarrowPaneHeights(bodyHeight))
		paneHeight = requestHeight
	}

	// reserve six lines for the pane frame, section strip, and spacing above the body content.
	return max(paneHeight-6, 1)
}

// currentRequestEditorView returns the bare widget view for the active request editor.
func (m *Model) currentRequestEditorView() string {
	m.ensureWidgetDefaults()

	switch m.viewState.RequestEditKind {
	case model.RequestEditKindField:
		return m.widgets.requestFieldInput.BareView()
	case model.RequestEditKindBody:
		return m.widgets.requestBodyInput.BareView()
	default:
		return ""
	}
}

// requestEditorInput returns the low-level request editor inputs used by the request feature package.
func (m *Model) requestEditorInput() requestui.EditorInput {
	m.ensureWidgetDefaults()

	return requestui.EditorInput{
		Kind:      m.viewState.RequestEditKind,
		Buffer:    m.viewState.RequestEditBuffer,
		FieldView: m.widgets.requestFieldInput.BareView(),
		BodyView:  m.widgets.requestBodyInput.BareView(),
	}
}

// requestValidationState flattens request validation into request-pane inputs.
func (m *Model) requestValidationState(activeSection string) requestui.ValidationState {
	state := requestui.ValidationState{
		MessagesBySection: m.requestUI.validation.MessagesForSection(activeSection),
	}
	if len(m.requestUI.validation.Issues) == 0 {
		return state
	}

	state.RowErrors = make(map[string]string, len(m.requestUI.validation.Issues))
	for _, issue := range m.requestUI.validation.Issues {
		state.RowErrors[issue.Target] = issue.Message
	}

	return state
}

// requestSupportState flattens request support notes into request-pane inputs.
func (m *Model) requestSupportState(activeSection string) requestui.SupportState {
	selected := m.resolvedSelectedOperation()
	notes := app.ProjectRequestSupportNotes(selected)
	if len(notes) == 0 {
		return requestui.SupportState{}
	}

	state := requestui.SupportState{
		RowNotes: make(map[string][]requestui.SupportNote),
	}
	for _, note := range notes {
		projected := requestui.SupportNote{
			Severity: requestSupportSeverity(note.Severity),
			Summary:  note.Summary,
			Detail:   note.Detail,
		}
		if note.Section == activeSection {
			state.MessagesBySection = append(state.MessagesBySection, projected)
		}
		if note.Target != "" {
			state.RowNotes[note.Target] = append(state.RowNotes[note.Target], projected)
		}
	}

	return state
}

// requestSupportSeverity maps app-layer support severity into request-pane render severity.
func requestSupportSeverity(severity app.RequestSupportSeverity) requestui.SupportSeverity {
	switch severity {
	case app.RequestSupportSeverityDowngraded:
		return requestui.SupportSeverityDowngraded
	default:
		return requestui.SupportSeverityUnsupported
	}
}

// projectRequestPane returns the unwindowed request pane data for default rendering and tests.
func (m *Model) projectRequestPane() requestui.Data {
	return m.projectRequestPaneForSize(0, 0).Data
}

// projectRequestPaneForSize projects the request pane for the given pane size.
func (m *Model) projectRequestPaneForSize(width, height int) requestui.PaneProjection {
	contentWidth := 0
	contentHeight := 0
	if width > 0 {
		// subtract the pane frame padding and borders before handing width to the feature package.
		contentWidth = max(width-4, 1)
	}
	if height > 0 {
		// reserve space for the pane frame, section strip, and the blank spacer between strip and body.
		contentHeight = max(height-6, 1)
	}
	if contentWidth > 0 || contentHeight > 0 {
		m.configureRequestEditors(contentWidth, height)
	}

	selected := m.resolvedSelectedOperation()
	draft := app.EnsureRequestDraft(&m.session, selected)
	security := m.effectiveSecurityRequirement(selected)
	activeSection := requestui.ResolveActiveSection(m.panes.activeRequestSection, selected, security, m.topLevelServers())

	return requestui.ProjectPane(requestui.PaneInput{
		LoadInFlight:      m.viewState.LoadInFlight,
		Selected:          selected,
		Draft:             draft,
		Security:          security,
		Servers:           m.topLevelServers(),
		SelectedServerURL: m.session.SelectedServerURL,
		SecuritySchemes:   m.securitySchemes(),
		AuthState:         m.session.AuthState,
		ActiveSection:     activeSection,
		ActiveRow:         m.viewState.RequestActiveRow,
		ScrollOffset:      m.viewState.RequestScrollOffset,
		Validation:        m.requestValidationState(activeSection),
		Support:           m.requestSupportState(activeSection),
		Editor:            m.requestEditorInput(),
		ContentWidth:      contentWidth,
		ContentHeight:     contentHeight,
		HelpOpen:          m.requestUI.editHelpOpen,
	})
}

// configureRequestEditors sizes the request editor widgets for the current pane.
func (m *Model) configureRequestEditors(contentWidth, height int) {
	// keep field editors narrower so they stay visually tied to the selected row.
	fieldPopupWidth := min(max(contentWidth-10, 24), 64)
	// allow the body editor to use more width because it edits multi-line payload content.
	bodyPopupWidth := min(max(contentWidth-8, 28), 84)
	// subtract the popup frame chrome before sizing the embedded text input.
	m.widgets.requestFieldInput.SetWidth(max(fieldPopupWidth-4, 12))

	// keep the body editor short enough to preserve surrounding pane context while editing.
	m.widgets.requestBodyInput.SetSize(max(bodyPopupWidth-4, 20), max(min(height-10, 12), 4))
}

// currentRequestHelpOverlay returns the current request editor help overlay, if any.
func (m *Model) currentRequestHelpOverlay() helpOverlayView {
	overlay := m.projectRequestPaneForSize(0, 0).HelpOverlay
	return helpOverlayView{
		Title: overlay.Title,
		Body:  overlay.Body,
		Hint:  overlay.Hint,
	}
}

// securitySchemes returns the loaded security-scheme map when a spec is available.
func (m *Model) securitySchemes() map[string]model.SecurityScheme {
	if m.session.Spec == nil {
		return nil
	}

	return m.session.Spec.SecuritySchemes
}

// topLevelServers returns the normalized top-level spec servers for the loaded document.
func (m *Model) topLevelServers() []model.Server {
	if m.session.Spec == nil {
		return nil
	}

	return m.session.Spec.Servers
}

// requestPaneContentForSize renders the request pane body for the given pane size.
func (m *Model) requestPaneContentForSize(width, height int) string {
	return requestui.Render(m.projectRequestPaneForSize(width, height).Data)
}

// requestEditActive reports whether the request pane is currently editing a request input.
func (m *Model) requestEditActive() bool {
	return m.viewState.ActiveEditorMode == model.EditorModeEdit &&
		m.viewState.RequestEditKind != model.RequestEditKindNone
}

// beginRequestEdit starts the request editor for the active request row.
func (m *Model) beginRequestEdit() {
	m.clearRequestValidation()
	selected := m.resolvedSelectedOperation()
	start := requestui.StartEdit(
		selected,
		app.EnsureRequestDraft(&m.session, selected),
		m.activeRequestRows(),
		m.viewState.RequestActiveRow,
		m.securitySchemes(),
		m.session.AuthState,
	)
	if start.CycleBodyMediaType {
		requestui.CycleBodyMediaType(&m.session, selected)
		return
	}
	if start.CycleServerURL {
		requestui.CycleServerURL(&m.session, m.topLevelServers())
		return
	}
	if start.Kind == model.RequestEditKindNone {
		return
	}

	m.viewState.ActiveEditorMode = model.EditorModeEdit
	m.viewState.RequestEditKind = start.Kind
	m.viewState.RequestEditTarget = start.Target
	m.viewState.RequestEditBuffer = start.Buffer
	m.requestUI.editHelpOpen = false
	if start.ResetScroll {
		m.viewState.RequestScrollOffset = 0
	}
	if start.FocusField {
		m.widgets.requestFieldInput.SetValue(start.Buffer)
		m.widgets.requestFieldInput.Focus()
	}
	if start.FocusBody {
		m.widgets.requestBodyInput.SetValue(start.Buffer)
		m.widgets.requestBodyInput.Focus()
	}
}

// saveRequestEdit saves the active request editor contents back into the request draft.
func (m *Model) saveRequestEdit() {
	if requestui.SaveEdit(
		&m.session,
		m.resolvedSelectedOperation(),
		m.activeRequestRows(),
		m.viewState.RequestActiveRow,
		m.viewState.RequestEditKind,
		m.viewState.RequestEditBuffer,
		m.securitySchemes(),
	) {
		m.finishRequestEdit()
		return
	}

	m.cancelRequestEdit()
}

// cancelRequestEdit closes the active request editor without applying additional changes.
func (m *Model) cancelRequestEdit() {
	m.finishRequestEdit()
}

// finishRequestEdit resets the request editor state after saving or canceling.
func (m *Model) finishRequestEdit() {
	m.widgets.requestFieldInput.Blur()
	m.widgets.requestBodyInput.Blur()
	m.viewState.ActiveEditorMode = model.EditorModeBrowse
	m.viewState.RequestEditKind = model.RequestEditKindNone
	m.viewState.RequestEditTarget = ""
	m.viewState.RequestEditBuffer = ""
	m.requestUI.editHelpOpen = false
	m.clearRequestValidation()
	m.syncActiveRequestRow()
}

// scrollRequestEditBy scrolls the body editor overlay by the given delta.
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

// scrollRequestEditToBoundary scrolls the body editor overlay to the first or last line.
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

// maxRequestEditScrollOffset returns the maximum scroll offset for the active request editor body.
func (m *Model) maxRequestEditScrollOffset() int {
	return requestui.MaxActiveSectionScrollOffset(m.projectRequestPaneForSize(0, 0).Data, m.requestVisibleBodyLines())
}

// updateRequestEditKey handles key input while the request editor is active.
func (m *Model) updateRequestEditKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.requestUI.editHelpOpen {
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "?":
			m.requestUI.editHelpOpen = false
			return m, nil
		default:
			m.requestUI.editHelpOpen = false
			return m, nil
		}
	}

	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc":
		m.cancelRequestEdit()
	case "?":
		m.requestUI.editHelpOpen = !m.requestUI.editHelpOpen
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
		cmd := m.widgets.requestFieldInput.Update(msg)
		m.viewState.RequestEditBuffer = m.widgets.requestFieldInput.Value()
		return m, cmd
	case model.RequestEditKindBody:
		cmd := m.widgets.requestBodyInput.Update(msg)
		m.viewState.RequestEditBuffer = m.widgets.requestBodyInput.Value()
		return m, cmd
	}

	return m, nil
}

// clearRequestValidation clears the current request validation state.
func (m *Model) clearRequestValidation() {
	m.requestUI.validation = app.RequestValidationResult{}
}

// executeCurrentRequest validates and executes the active request draft.
func (m *Model) executeCurrentRequest() tea.Cmd {
	selected := m.resolvedSelectedOperation()
	if selected == nil || m.viewState.LoadInFlight || m.viewState.ExecuteInFlight {
		return nil
	}

	validation := app.ValidateExecutableRequest(m.session, selected, app.EnsureRequestDraft(&m.session, selected))
	if validation.HasIssues() {
		m.requestUI.validation = validation
		if issue, ok := validation.FirstIssue(); ok {
			m.panes.activeRequestSection = issue.Section
			if index := requestui.RowIndexByID(m.activeRequestRows(), issue.Target); index >= 0 {
				m.viewState.RequestActiveRow = index
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
	session := app.CloneExecutionSession(m.session)

	return func() tea.Msg {
		return executeFinishedMsg{
			requestID: requestID,
			result:    service.ExecuteCurrent(context.Background(), session),
		}
	}
}
