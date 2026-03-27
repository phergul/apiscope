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
