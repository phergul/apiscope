package tui

import (
	"github.com/phergul/apiscope/internal/model"
	schemaexplorerui "github.com/phergul/apiscope/internal/tui/schemaexplorer"

	tea "github.com/charmbracelet/bubbletea"
)

// schemaExplorerUIState groups shell-owned schema explorer runtime state.
type schemaExplorerUIState struct {
	open  bool
	state schemaexplorerui.State
}

func (m *Model) schemaExplorerOpen() bool {
	return m.schemaExplorerUI.open
}

func (m *Model) schemaExplorerAvailable() bool {
	return schemaexplorerui.Available(m.resolvedSelectedOperation())
}

func (m *Model) schemaExplorerOperation() *model.Operation {
	if m.session.Spec == nil {
		return nil
	}

	key := m.schemaExplorerUI.state.OperationKey
	if key == "" {
		return nil
	}

	for index := range m.session.Spec.Operations {
		if m.session.Spec.Operations[index].Key == key {
			return &m.session.Spec.Operations[index]
		}
	}

	return nil
}

func (m *Model) openSchemaExplorer() {
	selected := m.resolvedSelectedOperation()
	if !schemaexplorerui.Available(selected) {
		return
	}

	m.schemaExplorerUI = schemaExplorerUIState{
		open:  true,
		state: schemaexplorerui.OpenState(selected),
	}
}

func (m *Model) closeSchemaExplorer() {
	m.schemaExplorerUI = schemaExplorerUIState{}
}

func (m *Model) schemaExplorerMetrics() (shellWidth, shellHeight, contentWidth, contentHeight int) {
	width, height := m.resolvedDimensions()
	bodyHeight := max(height-m.statusBarHeight(width), 12)
	return width, bodyHeight, max(width-4, 1), max(bodyHeight-2, 1)
}

func (m *Model) projectSchemaExplorerForSize(contentWidth, contentHeight int) schemaexplorerui.Projection {
	return schemaexplorerui.Project(schemaexplorerui.ProjectionInput{
		Operation:     m.schemaExplorerOperation(),
		State:         m.schemaExplorerUI.state,
		ContentWidth:  contentWidth,
		ContentHeight: contentHeight,
	})
}

func (m *Model) renderSchemaExplorer(width, height int) string {
	body := schemaexplorerui.Render(m.projectSchemaExplorerForSize(max(width-4, 1), max(height-2, 1)).Data)
	return m.renderPane("", "Schema Explorer", "Close Esc", body, "", width, height, !m.helpOverlayOpen())
}

func (m *Model) updateSchemaExplorerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "t":
		m.cycleTheme(true)
		return m, nil
	case "T":
		m.cycleTheme(false)
		return m, nil
	}

	_, _, contentWidth, contentHeight := m.schemaExplorerMetrics()
	projected := m.projectSchemaExplorerForSize(contentWidth, contentHeight)
	// The feature package owns all explorer navigation rules; root only passes the
	// current viewport bounds and applies any requested shell-level close action.
	result := schemaexplorerui.Update(m.schemaExplorerOperation(), m.schemaExplorerUI.state, schemaexplorerui.UpdateInput{
		Key:              msg.String(),
		VisibleRows:      projected.VisibleRows,
		MaxPreviewScroll: projected.MaxPreviewScroll,
	})
	m.schemaExplorerUI.state = result.State
	if result.Action.Close {
		m.closeSchemaExplorer()
	}

	return m, nil
}
