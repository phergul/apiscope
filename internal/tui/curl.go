package tui

import (
	"strings"

	"github.com/phergul/apiscope/internal/app"
	"github.com/phergul/apiscope/internal/model"
	"github.com/phergul/apiscope/internal/tui/request"
	"github.com/phergul/apiscope/internal/tui/widgets"
	"github.com/phergul/apiscope/internal/util"

	tea "github.com/charmbracelet/bubbletea"
)

// curlPopupOpen reports whether the shell-level curl export popup is active.
func (m *Model) curlPopupOpen() bool {
	return m.curlUI.open
}

// closeCurlPopup hides the curl export popup.
func (m *Model) closeCurlPopup() {
	m.curlUI = curlUIState{}
}

// exportCurrentCurl validates and exports the active request as a curl command.
func (m *Model) exportCurrentCurl() {
	selected := m.resolvedSelectedOperation()
	if selected == nil || m.viewState.LoadInFlight || m.viewState.ExecuteInFlight {
		return
	}

	result := m.service.ExportCurl(app.CloneExecutionSession(m.session))
	if result.Validation.HasIssues() {
		m.applyRequestValidation(result.Validation, "Curl export validation failed")
		return
	}
	if strings.TrimSpace(result.Error) != "" {
		m.viewState.Notice = "Curl export failed"
		return
	}

	m.clearRequestValidation()
	m.curlUI.open = true
	m.curlUI.command = result.Command
	m.viewState.Notice = "Curl export ready"
	if m.viewState.FocusedPane == model.FocusedPaneResponse {
		m.viewState.ExpandedRightPane = model.FocusedPaneResponse
	}
}

func (m *Model) updateCurlPopupKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc", "q", "c":
		m.closeCurlPopup()
	case "t":
		m.cycleTheme(true)
	case "T":
		m.cycleTheme(false)
	}

	return m, nil
}

func (m *Model) renderCurlPopup(view string) string {
	if !m.curlPopupOpen() {
		return view
	}

	width, _ := m.resolvedDimensions()
	maxWidth := max(width-2, 20)
	minWidth := min(72, maxWidth)
	popupWidth := util.Clamp(int(float64(width)*0.8), minWidth, maxWidth)

	popup := widgets.RenderPopup(widgets.PopupData{
		Title:       "Curl export",
		Meta:        m.session.SelectedOperationKey.String(),
		Body:        m.curlUI.command,
		Help:        "Esc close",
		HelpVisible: true,
		Width:       popupWidth,
		Focused:     !m.helpOverlayOpen(),
	})

	return widgets.OverlayCentered(widgets.CenteredOverlayData{
		Base:  view,
		Popup: popup,
	})
}

func (m *Model) applyRequestValidation(validation app.RequestValidationResult, notice string) {
	m.requestUI.validation = validation
	if issue, ok := validation.FirstIssue(); ok {
		m.panes.activeRequestSection = issue.Section
		if index := request.RowIndexByID(m.activeRequestRows(), issue.Target); index >= 0 {
			m.viewState.RequestActiveRow = index
		}
		m.ensureActiveRequestRowVisible()
	}
	m.viewState.FocusedPane = model.FocusedPaneRequest
	m.viewState.ExpandedRightPane = model.FocusedPaneRequest
	m.viewState.Notice = notice
}
