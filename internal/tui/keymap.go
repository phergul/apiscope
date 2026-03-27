package tui

import (
	"api-tui/internal/model"

	tea "github.com/charmbracelet/bubbletea"
)

var paneFocusOrder = []model.FocusedPane{
	model.FocusedPaneOperations,
	model.FocusedPaneDetails,
	model.FocusedPaneRequest,
	model.FocusedPaneResponse,
}

func (m *Model) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.viewState.ActiveEditorMode == model.EditorModeFilter {
		return m.updateFilterKey(msg)
	}

	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "1":
		m.viewState.FocusedPane = model.FocusedPaneOperations
	case "2":
		m.viewState.FocusedPane = model.FocusedPaneDetails
	case "3":
		m.viewState.FocusedPane = model.FocusedPaneRequest
	case "4":
		m.viewState.FocusedPane = model.FocusedPaneResponse
	case "tab":
		m.viewState.FocusedPane = nextFocusedPane(m.viewState.FocusedPane)
	case "shift+tab":
		m.viewState.FocusedPane = previousFocusedPane(m.viewState.FocusedPane)
	case "/":
		m.viewState.FocusedPane = model.FocusedPaneOperations
		m.viewState.ActiveEditorMode = model.EditorModeFilter
	case "esc":
		if m.viewState.FilterText != "" {
			m.viewState.FilterText = ""
			m.syncVisibleOperations()
		}
	case "j", "down":
		if m.viewState.FocusedPane == model.FocusedPaneOperations {
			m.setSelectedOperationByVisibleIndex(m.viewState.OperationsCursor + 1)
		}
	case "k", "up":
		if m.viewState.FocusedPane == model.FocusedPaneOperations {
			m.setSelectedOperationByVisibleIndex(m.viewState.OperationsCursor - 1)
		}
	case "home":
		switch m.viewState.FocusedPane {
		case model.FocusedPaneOperations:
			m.setSelectedOperationByVisibleIndex(0)
		case model.FocusedPaneDetails:
			m.setDetailsSectionBoundary(false)
		}
	case "end":
		switch m.viewState.FocusedPane {
		case model.FocusedPaneOperations:
			m.setSelectedOperationByVisibleIndex(len(m.viewState.VisibleOperationKeys) - 1)
		case model.FocusedPaneDetails:
			m.setDetailsSectionBoundary(true)
		}
	case "]":
		switch m.viewState.FocusedPane {
		case model.FocusedPaneOperations:
			m.jumpToAdjacentOperationGroup(1)
		case model.FocusedPaneDetails:
			m.moveDetailsSection(1)
		}
	case "[":
		switch m.viewState.FocusedPane {
		case model.FocusedPaneOperations:
			m.jumpToAdjacentOperationGroup(-1)
		case model.FocusedPaneDetails:
			m.moveDetailsSection(-1)
		}
	}

	return m, nil
}

func (m *Model) updateFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "enter":
		m.viewState.ActiveEditorMode = model.EditorModeBrowse
	case "esc":
		m.viewState.FilterText = ""
		m.syncVisibleOperations()
		m.viewState.ActiveEditorMode = model.EditorModeBrowse
	case "backspace", "ctrl+h", "delete":
		m.viewState.FilterText = trimLastRune(m.viewState.FilterText)
		m.syncVisibleOperations()
	default:
		if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 {
			m.viewState.FilterText = appendFilterInput(m.viewState.FilterText, msg.Runes)
			m.syncVisibleOperations()
		}
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
