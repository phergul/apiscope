package tui

import (
	"strconv"

	"github.com/phergul/apiscope/internal/app"
	"github.com/phergul/apiscope/internal/model"
	historyui "github.com/phergul/apiscope/internal/tui/history"
	requestui "github.com/phergul/apiscope/internal/tui/request"
	responseui "github.com/phergul/apiscope/internal/tui/response"
	"github.com/phergul/apiscope/internal/tui/widgets"
	"github.com/phergul/apiscope/internal/util"
)

// historyPopupOpen reports whether the shell-level previous-requests popup is active.
func (m *Model) historyPopupOpen() bool {
	return m.historyUI.open
}

// openHistoryPopup shows the previous-requests popup for the current operation.
func (m *Model) openHistoryPopup() {
	m.historyUI.open = true
	m.historyUI.activeRow = historyui.ClampActiveRow(m.selectedOperationHistory(), m.historyUI.activeRow)
}

// closeHistoryPopup hides the previous-requests popup and resets its transient cursor state.
func (m *Model) closeHistoryPopup() {
	m.historyUI.open = false
	m.historyUI.activeRow = 0
}

// moveHistoryPopupRow moves the popup selection through operation-scoped history entries.
func (m *Model) moveHistoryPopupRow(direction int) {
	m.historyUI.activeRow = historyui.MoveActiveRow(m.selectedOperationHistory(), m.historyUI.activeRow, direction)
}

// setHistoryPopupBoundary jumps the popup selection to the first or last history row.
func (m *Model) setHistoryPopupBoundary(last bool) {
	m.historyUI.activeRow = historyui.BoundaryActiveRow(m.selectedOperationHistory(), last)
}

// selectedOperationHistory returns newest-first history for the currently selected operation.
func (m *Model) selectedOperationHistory() []model.HistoryEntry {
	return app.HistoryForOperation(m.session, m.session.SelectedOperationKey)
}

// loadSelectedHistoryResponse recalls only the stored response and returns the user to pane 4.
func (m *Model) loadSelectedHistoryResponse() {
	entry, ok := historyui.ActiveEntry(m.selectedOperationHistory(), m.historyUI.activeRow)
	if !ok || !app.LoadHistoryResponse(&m.session, entry) {
		return
	}

	m.closeHistoryPopup()
	m.setFocusedPane(model.FocusedPaneResponse)
	m.panes.activeResponseSection = responseui.SectionLive
	m.viewState.ResponseScrollOffset = 0
	m.viewState.Notice = "Loaded previous response #" + formatRequestID(entry.RequestID)
}

// restoreSelectedHistoryRequest restores the executed request inputs so the user can rerun it.
func (m *Model) restoreSelectedHistoryRequest() {
	entry, ok := historyui.ActiveEntry(m.selectedOperationHistory(), m.historyUI.activeRow)
	if !ok || !app.RestoreHistoryRequest(&m.session, entry) {
		return
	}

	m.closeHistoryPopup()
	m.setFocusedPane(model.FocusedPaneRequest)
	m.panes.activeRequestSection = requestui.ResolveActiveSection(
		m.panes.activeRequestSection,
		m.resolvedSelectedOperation(),
		m.effectiveSecurityRequirement(m.resolvedSelectedOperation()),
		m.topLevelServers(),
	)
	m.syncActiveRequestRow()
	m.clearRequestValidation()
	m.viewState.Notice = "Restored request #" + formatRequestID(entry.RequestID)
}

// renderHistoryPopup overlays the centered previous-requests popup above the shell layout.
func (m *Model) renderHistoryPopup(view string) string {
	if !m.historyPopupOpen() {
		return view
	}

	width, height := m.resolvedDimensions()
	// keep the popup wide enough for request summaries while still fitting compact terminals.
	popupWidth := util.Clamp(int(float64(width)*0.78), 40, max(width-2, 40))
	contentHeight := util.Clamp(height-14, 8, 16)
	data := historyui.ProjectPopup(historyui.PopupInput{
		Selected:      m.resolvedSelectedOperation(),
		Entries:       m.selectedOperationHistory(),
		ActiveRow:     m.historyUI.activeRow,
		ContentWidth:  max(popupWidth-4, 1),
		ContentHeight: contentHeight,
	})

	popup := widgets.RenderPopup(widgets.PopupData{
		Title:   data.Title,
		Meta:    data.Meta,
		Body:    data.Body,
		Width:   popupWidth,
		Focused: !m.helpOverlayOpen(),
	})

	return widgets.OverlayCentered(widgets.CenteredOverlayData{
		Base:  view,
		Popup: popup,
	})
}

// formatRequestID keeps user-facing history notices consistent with popup row labels.
func formatRequestID(requestID uint64) string {
	return strconv.FormatUint(requestID, 10)
}
