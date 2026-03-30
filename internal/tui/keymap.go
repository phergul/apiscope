package tui

import (
	"unicode/utf8"

	"github.com/phergul/apiscope/internal/model"

	tea "github.com/charmbracelet/bubbletea"
)

var paneFocusOrder = []model.FocusedPane{
	model.FocusedPaneOperations,
	model.FocusedPaneDetails,
	model.FocusedPaneRequest,
	model.FocusedPaneResponse,
}

func (m *Model) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.hasBlockingLoadError() {
		switch msg.String() {
		case "ctrl+c", "q", "enter", "esc":
			return m, tea.Quit
		default:
			return m, nil
		}
	}

	if m.viewState.ActiveEditorMode == model.EditorModeFilter {
		return m.updateFilterKey(msg)
	}
	if m.requestEditActive() {
		return m.updateRequestEditKey(msg)
	}

	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "ctrl+r":
		if cmd := m.executeCurrentRequest(); cmd != nil {
			return m, cmd
		}
		return m, nil
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
		m.setFocusedPane(model.FocusedPaneOperations)
		m.viewState.ActiveEditorMode = model.EditorModeFilter
		m.filterInput.SetValue(m.viewState.FilterText)
		m.filterInput.Focus()
	case "enter":
		if m.viewState.FocusedPane == model.FocusedPaneRequest {
			m.beginRequestEdit()
		}
	case "z":
		m.viewState.ZoomedPane = !m.viewState.ZoomedPane
	case "esc":
		if m.viewState.FilterText != "" {
			m.viewState.FilterText = ""
			m.syncVisibleOperations()
		}
	case "j", "down":
		if m.viewState.FocusedPane == model.FocusedPaneOperations {
			m.setSelectedOperationByVisibleIndex(m.viewState.OperationsCursor + 1)
		} else if m.viewState.FocusedPane == model.FocusedPaneDetails {
			m.scrollDetailsBy(1)
		} else if m.viewState.FocusedPane == model.FocusedPaneRequest {
			m.moveRequestRow(1)
		} else if m.viewState.FocusedPane == model.FocusedPaneResponse {
			m.scrollResponseBy(1)
		}
	case "k", "up":
		if m.viewState.FocusedPane == model.FocusedPaneOperations {
			m.setSelectedOperationByVisibleIndex(m.viewState.OperationsCursor - 1)
		} else if m.viewState.FocusedPane == model.FocusedPaneDetails {
			m.scrollDetailsBy(-1)
		} else if m.viewState.FocusedPane == model.FocusedPaneRequest {
			m.moveRequestRow(-1)
		} else if m.viewState.FocusedPane == model.FocusedPaneResponse {
			m.scrollResponseBy(-1)
		}
	case "home":
		switch m.viewState.FocusedPane {
		case model.FocusedPaneOperations:
			m.setSelectedOperationByVisibleIndex(0)
		case model.FocusedPaneDetails:
			m.scrollDetailsToBoundary(false)
		case model.FocusedPaneRequest:
			m.setRequestRowBoundary(false)
		case model.FocusedPaneResponse:
			m.scrollResponseToBoundary(false)
		}
	case "end":
		switch m.viewState.FocusedPane {
		case model.FocusedPaneOperations:
			m.setSelectedOperationByVisibleIndex(len(m.viewState.VisibleOperationKeys) - 1)
		case model.FocusedPaneDetails:
			m.scrollDetailsToBoundary(true)
		case model.FocusedPaneRequest:
			m.setRequestRowBoundary(true)
		case model.FocusedPaneResponse:
			m.scrollResponseToBoundary(true)
		}
	case "]", "l":
		switch m.viewState.FocusedPane {
		case model.FocusedPaneOperations:
			m.jumpToAdjacentOperationGroup(1)
		case model.FocusedPaneDetails:
			m.moveDetailsSection(1)
		case model.FocusedPaneRequest:
			m.moveRequestSection(1)
		case model.FocusedPaneResponse:
			m.moveResponseSection(1)
		}
	case "[", "h":
		switch m.viewState.FocusedPane {
		case model.FocusedPaneOperations:
			m.jumpToAdjacentOperationGroup(-1)
		case model.FocusedPaneDetails:
			m.moveDetailsSection(-1)
		case model.FocusedPaneRequest:
			m.moveRequestSection(-1)
		case model.FocusedPaneResponse:
			m.moveResponseSection(-1)
		}
	}

	return m, nil
}

func (m *Model) setFocusedPane(pane model.FocusedPane) {
	m.viewState.FocusedPane = pane
	switch pane {
	case model.FocusedPaneRequest, model.FocusedPaneResponse:
		m.viewState.ExpandedRightPane = pane
	}
}

func (m *Model) updateFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.ensureWidgetDefaults()
	if m.filterInput.Value() != m.viewState.FilterText {
		m.filterInput.SetValue(m.viewState.FilterText)
	}
	m.filterInput.Focus()

	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "enter":
		m.filterInput.Blur()
		m.viewState.ActiveEditorMode = model.EditorModeBrowse
	case "esc":
		m.viewState.FilterText = ""
		m.filterInput.SetValue("")
		m.filterInput.Blur()
		m.syncVisibleOperations()
		m.viewState.ActiveEditorMode = model.EditorModeBrowse
	case "backspace", "ctrl+h", "delete":
		if m.viewState.FilterText == "" {
			break
		}
		_, size := utf8.DecodeLastRuneInString(m.viewState.FilterText)
		if size <= 0 {
			m.viewState.FilterText = ""
		} else {
			m.viewState.FilterText = m.viewState.FilterText[:len(m.viewState.FilterText)-size]
		}
		m.filterInput.SetValue(m.viewState.FilterText)
		m.syncVisibleOperations()
	default:
		cmd := m.filterInput.Update(msg)
		m.viewState.FilterText = m.filterInput.Value()
		m.syncVisibleOperations()
		return m, cmd
	}

	return m, nil
}

func nextFocusedPane(current model.FocusedPane) model.FocusedPane {
	for index, pane := range paneFocusOrder {
		if pane == current {
			return paneFocusOrder[(index+1)%len(paneFocusOrder)]
		}
	}

	return paneFocusOrder[0]
}

func previousFocusedPane(current model.FocusedPane) model.FocusedPane {
	for index, pane := range paneFocusOrder {
		if pane == current {
			return paneFocusOrder[(index+len(paneFocusOrder)-1)%len(paneFocusOrder)]
		}
	}

	return paneFocusOrder[0]
}
