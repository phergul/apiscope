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
	m.historyUI.previewScrollOffset = 0
}

// closeHistoryPopup hides the previous-requests popup and resets its transient cursor state.
func (m *Model) closeHistoryPopup() {
	m.historyUI.open = false
	m.historyUI.activeRow = 0
	m.historyUI.previewScrollOffset = 0
}

// moveHistoryPopupRow moves the popup selection through operation-scoped history entries.
func (m *Model) moveHistoryPopupRow(direction int) {
	next := historyui.MoveActiveRow(m.selectedOperationHistory(), m.historyUI.activeRow, direction)
	if next != m.historyUI.activeRow {
		m.historyUI.previewScrollOffset = 0
	}
	m.historyUI.activeRow = next
}

// setHistoryPopupBoundary jumps the popup selection to the first or last history row.
func (m *Model) setHistoryPopupBoundary(last bool) {
	next := historyui.BoundaryActiveRow(m.selectedOperationHistory(), last)
	if next != m.historyUI.activeRow {
		m.historyUI.previewScrollOffset = 0
	}
	m.historyUI.activeRow = next
}

// scrollHistoryPopupPreviewBy moves the preview viewport by the provided delta.
func (m *Model) scrollHistoryPopupPreviewBy(delta int) {
	projected := m.projectHistoryPopup()
	target := m.historyUI.previewScrollOffset + delta
	if target < 0 {
		target = 0
	}
	if target > projected.MaxPreviewScroll {
		target = projected.MaxPreviewScroll
	}

	m.historyUI.previewScrollOffset = target
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
	m.syncAppliedEnvironmentMarker()
	m.viewState.Notice = "Restored request #" + formatRequestID(entry.RequestID)
}

// projectHistoryPopup projects the current previous-requests popup for the resolved shell size.
func (m *Model) projectHistoryPopup() historyui.PopupData {
	width, height := m.resolvedDimensions()
	maxWidth := max(width-2, 20)
	minWidth := min(60, maxWidth)
	popupWidth := util.Clamp(int(float64(width)*0.8), minWidth, maxWidth)
	maxHeight := max(height-2, 8)
	minHeight := min(12, maxHeight)
	popupHeight := util.Clamp(int(float64(height)*0.8), minHeight, maxHeight)

	return historyui.ProjectPopup(historyui.PopupInput{
		Selected:            m.resolvedSelectedOperation(),
		Entries:             m.selectedOperationHistory(),
		ActiveRow:           m.historyUI.activeRow,
		PreviewScrollOffset: m.historyUI.previewScrollOffset,
		ContentWidth:        max(popupWidth-4, 1),
		ContentHeight:       max(popupHeight-4, 1),
	})
}

// renderHistoryPopup overlays the centered previous-requests popup above the shell layout.
func (m *Model) renderHistoryPopup(view string) string {
	if !m.historyPopupOpen() {
		return view
	}

	width, _ := m.resolvedDimensions()
	maxWidth := max(width-2, 20)
	minWidth := min(60, maxWidth)
	popupWidth := util.Clamp(int(float64(width)*0.8), minWidth, maxWidth)
	data := m.projectHistoryPopup()

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
