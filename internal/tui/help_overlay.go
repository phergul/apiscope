package tui

import (
	"strings"

	"github.com/phergul/apiscope/internal/model"
	detailsui "github.com/phergul/apiscope/internal/tui/details"
	historyui "github.com/phergul/apiscope/internal/tui/history"
	operationsui "github.com/phergul/apiscope/internal/tui/operations"
	requestui "github.com/phergul/apiscope/internal/tui/request"
	responseui "github.com/phergul/apiscope/internal/tui/response"
	schemaexplorerui "github.com/phergul/apiscope/internal/tui/schemaexplorer"
	"github.com/phergul/apiscope/internal/tui/widgets"
	"github.com/phergul/apiscope/internal/util"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// helpOverlayOpen reports whether the shell help overlay is currently visible.
func (m *Model) helpOverlayOpen() bool {
	return m.helpUI.open && strings.TrimSpace(m.helpUI.view.Body) != ""
}

// openHelpOverlay captures and opens help for the current shell context.
func (m *Model) openHelpOverlay() {
	view := m.currentHelpView()
	if strings.TrimSpace(view.Body) == "" {
		return
	}

	m.helpUI.open = true
	m.helpUI.view = view
}

// closeHelpOverlay hides the current shell help overlay.
func (m *Model) closeHelpOverlay() {
	m.helpUI = helpUIState{}
}

// updateHelpOverlayKey handles key input while the shell help overlay is active.
func (m *Model) updateHelpOverlayKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc", "?":
		m.closeHelpOverlay()
	}

	return m, nil
}

// projectHelpOverlay projects the currently active shell help overlay.
func (m *Model) projectHelpOverlay() widgets.HelpView {
	if !m.helpOverlayOpen() {
		return widgets.HelpView{}
	}

	return m.helpUI.view
}

// currentHelpView returns the contextual help content for the highest-priority shell state.
func (m *Model) currentHelpView() widgets.HelpView {
	switch {
	case m.hasBlockingLoadError():
		return m.blockingLoadErrorHelpView()
	case m.schemaExplorerOpen():
		return schemaexplorerui.BuildHelpView()
	case m.historyPopupOpen():
		return historyui.BuildHelpView()
	case m.viewState.ActiveEditorMode == model.EditorModeFilter:
		return operationsui.BuildFilterHelpView()
	case m.requestEditActive():
		return requestui.BuildEditHelpView(m.currentRequestEditorState())
	}

	switch m.viewState.FocusedPane {
	case model.FocusedPaneDetails:
		return detailsui.BuildBrowseHelpView()
	case model.FocusedPaneRequest:
		return requestui.BuildBrowseHelpView()
	case model.FocusedPaneResponse:
		return responseui.BuildBrowseHelpView()
	default:
		return operationsui.BuildBrowseHelpView()
	}
}

// blockingLoadErrorHelpView returns the contextual help for the blocking load-error modal.
func (m *Model) blockingLoadErrorHelpView() widgets.HelpView {
	return widgets.HelpView{
		Hint:  "Help - ?",
		Title: "Load error help",
		Body: strings.Join([]string{
			"Enter, Esc, or q quit",
			"? or Esc close help",
		}, "\n"),
	}
}

// renderHelpOverlay renders the current help overlay above the status bar when needed.
func (m *Model) renderHelpOverlay(view string, width int, overlay widgets.HelpView) string {
	if strings.TrimSpace(overlay.Body) == "" {
		return view
	}

	popup := widgets.RenderPopup(widgets.PopupData{
		Title:   overlay.Title,
		Body:    overlay.Body,
		Width:   helpOverlayWidth(width, overlay.Title, overlay.Body),
		Focused: true,
	})

	return widgets.OverlayBottomRight(widgets.BottomRightOverlayData{
		Base:        view,
		Popup:       popup,
		BottomInset: m.statusBarHeight(width),
	})
}

// helpOverlayWidth returns the shell popup width for the current help overlay content.
func helpOverlayWidth(totalWidth int, title, body string) int {
	maxLineWidth := lipgloss.Width(strings.TrimSpace(title))
	for _, line := range strings.Split(strings.TrimSpace(body), "\n") {
		if lineWidth := lipgloss.Width(line); lineWidth > maxLineWidth {
			maxLineWidth = lineWidth
		}
	}

	// add the popup frame width, but keep a one-cell gutter inside the shell edges.
	return util.Clamp(maxLineWidth+4, 20, max(totalWidth-2, 20))
}
