package tui

import (
	"github.com/phergul/apiscope/internal/model"
	operationsui "github.com/phergul/apiscope/internal/tui/operations"

	tea "github.com/charmbracelet/bubbletea"
)

var paneFocusOrder = []model.FocusedPane{
	model.FocusedPaneOperations,
	model.FocusedPaneDetails,
	model.FocusedPaneRequest,
	model.FocusedPaneResponse,
}

// updateKey routes top-level key handling across global shortcuts and focused panes.
func (m *Model) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.helpOverlayOpen() {
		return m.updateHelpOverlayKey(msg)
	}

	if msg.String() == "?" {
		m.openHelpOverlay()
		return m, nil
	}

	if m.hasBlockingLoadError() {
		switch msg.String() {
		case "ctrl+c", "q", "enter", "esc":
			return m, tea.Quit
		default:
			return m, nil
		}
	}

	if m.viewState.ActiveEditorMode == model.EditorModeFilter {
		return m.updateOperationsFilterKey(msg)
	}
	if m.requestEditActive() {
		return m.updateRequestEditKey(msg)
	}
	if m.historyPopupOpen() {
		return m.updateHistoryPopupKey(msg)
	}

	if handledModel, handledCmd, handled := m.updateGlobalKey(msg); handled {
		return handledModel, handledCmd
	}

	return m.updateBrowseKey(msg)
}

// updateGlobalKey handles application-wide shortcuts before pane-specific routing.
func (m *Model) updateGlobalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit, true
	case "ctrl+r":
		if cmd := m.executeCurrentRequest(); cmd != nil {
			return m, cmd, true
		}
		return m, nil, true
	case "p":
		m.openHistoryPopup()
		return m, nil, true
	case "1":
		m.setFocusedPane(model.FocusedPaneOperations)
	case "2":
		m.setFocusedPane(model.FocusedPaneDetails)
	case "3":
		m.setFocusedPane(model.FocusedPaneRequest)
	case "4":
		m.setFocusedPane(model.FocusedPaneResponse)
	case "tab":
		m.setFocusedPane(nextFocusedPane(m.viewState.FocusedPane))
	case "shift+tab":
		m.setFocusedPane(previousFocusedPane(m.viewState.FocusedPane))
	case "/":
		m.beginOperationsFilterEdit()
	case "enter":
		if m.viewState.FocusedPane == model.FocusedPaneRequest {
			m.beginRequestEdit()
		}
	case "z":
		m.viewState.ZoomedPane = !m.viewState.ZoomedPane
	case "esc":
		if m.viewState.FilterText != "" {
			m.clearOperationsFilter()
		}
	default:
		return m, nil, false
	}

	return m, nil, true
}

func (m *Model) updateHistoryPopupKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc", "p", "q":
		m.closeHistoryPopup()
	case "j", "down":
		m.moveHistoryPopupRow(1)
	case "k", "up":
		m.moveHistoryPopupRow(-1)
	case "home":
		m.setHistoryPopupBoundary(false)
	case "end":
		m.setHistoryPopupBoundary(true)
	case "enter":
		m.loadSelectedHistoryResponse()
	case "r":
		m.restoreSelectedHistoryRequest()
	}

	return m, nil
}

// updateBrowseKey routes pane-local browse keys for the focused pane.
func (m *Model) updateBrowseKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		m.moveFocusedPaneDown()
	case "k", "up":
		m.moveFocusedPaneUp()
	case "home":
		m.moveFocusedPaneToBoundary(false)
	case "end":
		m.moveFocusedPaneToBoundary(true)
	case "]", "l":
		m.moveFocusedPaneSection(1)
	case "[", "h":
		m.moveFocusedPaneSection(-1)
	}

	return m, nil
}

// setFocusedPane updates the focused pane and right-pane emphasis when needed.
func (m *Model) setFocusedPane(pane model.FocusedPane) {
	m.viewState.FocusedPane = pane
	switch pane {
	case model.FocusedPaneRequest, model.FocusedPaneResponse:
		m.viewState.ExpandedRightPane = pane
	}
}

// beginOperationsFilterEdit enters operations filter editing mode.
func (m *Model) beginOperationsFilterEdit() {
	m.setFocusedPane(model.FocusedPaneOperations)
	m.ensureWidgetDefaults()
	if m.widgets.filterInput.Value() != m.viewState.FilterText {
		m.widgets.filterInput.SetValue(m.viewState.FilterText)
	}
	m.viewState.ActiveEditorMode = model.EditorModeFilter
	m.widgets.filterInput.Focus()
}

// clearOperationsFilter clears the current filter and refreshes the visible operations list.
func (m *Model) clearOperationsFilter() {
	m.viewState.FilterText = ""
	m.widgets.filterInput.SetValue("")
	m.syncVisibleOperations()
}

// updateOperationsFilterKey handles key input while the operations filter editor is active.
func (m *Model) updateOperationsFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.beginOperationsFilterEdit()

	update := operationsui.UpdateFilterEditor(msg, m.viewState.FilterText)
	if update.Quit {
		return m, tea.Quit
	}

	if update.UseWidgetUpdate {
		cmd := m.widgets.filterInput.Update(msg)
		m.viewState.FilterText = m.widgets.filterInput.Value()
		if update.RefreshVisible {
			m.syncVisibleOperations()
		}
		return m, cmd
	}

	m.viewState.FilterText = update.FilterText
	m.widgets.filterInput.SetValue(update.FilterText)
	if update.RefreshVisible {
		m.syncVisibleOperations()
	}
	if !update.Editing {
		m.widgets.filterInput.Blur()
		m.viewState.ActiveEditorMode = model.EditorModeBrowse
	}

	return m, nil
}

// moveFocusedPaneDown routes downward movement within the focused pane.
func (m *Model) moveFocusedPaneDown() {
	switch m.viewState.FocusedPane {
	case model.FocusedPaneOperations:
		m.setSelectedOperationByVisibleIndex(m.viewState.OperationsCursor + 1)
	case model.FocusedPaneDetails:
		m.scrollDetailsBy(1)
	case model.FocusedPaneRequest:
		m.moveRequestRow(1)
	case model.FocusedPaneResponse:
		m.scrollResponseBy(1)
	}
}

// moveFocusedPaneUp routes upward movement within the focused pane.
func (m *Model) moveFocusedPaneUp() {
	switch m.viewState.FocusedPane {
	case model.FocusedPaneOperations:
		m.setSelectedOperationByVisibleIndex(m.viewState.OperationsCursor - 1)
	case model.FocusedPaneDetails:
		m.scrollDetailsBy(-1)
	case model.FocusedPaneRequest:
		m.moveRequestRow(-1)
	case model.FocusedPaneResponse:
		m.scrollResponseBy(-1)
	}
}

// moveFocusedPaneToBoundary routes home/end behavior within the focused pane.
func (m *Model) moveFocusedPaneToBoundary(last bool) {
	switch m.viewState.FocusedPane {
	case model.FocusedPaneOperations:
		if last {
			m.setSelectedOperationByVisibleIndex(len(m.viewState.VisibleOperationKeys) - 1)
		} else {
			m.setSelectedOperationByVisibleIndex(0)
		}
	case model.FocusedPaneDetails:
		m.scrollDetailsToBoundary(last)
	case model.FocusedPaneRequest:
		m.setRequestRowBoundary(last)
	case model.FocusedPaneResponse:
		m.scrollResponseToBoundary(last)
	}
}

// moveFocusedPaneSection routes section or group movement within the focused pane.
func (m *Model) moveFocusedPaneSection(direction int) {
	switch m.viewState.FocusedPane {
	case model.FocusedPaneOperations:
		m.jumpToAdjacentOperationGroup(direction)
	case model.FocusedPaneDetails:
		m.moveDetailsSection(direction)
	case model.FocusedPaneRequest:
		m.moveRequestSection(direction)
	case model.FocusedPaneResponse:
		m.moveResponseSection(direction)
	}
}

// nextFocusedPane returns the next pane in the focus cycle.
func nextFocusedPane(current model.FocusedPane) model.FocusedPane {
	for index, pane := range paneFocusOrder {
		if pane == current {
			return paneFocusOrder[(index+1)%len(paneFocusOrder)]
		}
	}

	return paneFocusOrder[0]
}

// previousFocusedPane returns the previous pane in the focus cycle.
func previousFocusedPane(current model.FocusedPane) model.FocusedPane {
	for index, pane := range paneFocusOrder {
		if pane == current {
			return paneFocusOrder[(index+len(paneFocusOrder)-1)%len(paneFocusOrder)]
		}
	}

	return paneFocusOrder[0]
}
