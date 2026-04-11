package tui

import (
	"strings"

	"github.com/phergul/apiscope/internal/tui/specdiff"
	"github.com/phergul/apiscope/internal/tui/widgets"
	"github.com/phergul/apiscope/internal/util"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) specDiffPopupOpen() bool {
	return m.specDiffUI.open
}

func (m *Model) closeSpecDiffPopup() {
	m.specDiffUI.open = false
}

func (m *Model) openSpecDiffPopup() {
	if !m.specDiffUI.hasBaseline {
		m.viewState.Notice = "Spec diff unavailable"
		return
	}

	m.specDiffUI.open = true
	m.viewState.Notice = "Spec diff ready"
}

func (m *Model) updateSpecDiffPopupKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc", "q", "d":
		m.closeSpecDiffPopup()
	case "R", "ctrl+l":
		m.closeSpecDiffPopup()
		if cmd := m.reloadSpec(); cmd != nil {
			return m, cmd
		}
	case "t":
		m.cycleTheme(true)
	case "T":
		m.cycleTheme(false)
	}

	return m, nil
}

func (m *Model) renderSpecDiffPopup(view string) string {
	if !m.specDiffPopupOpen() {
		return view
	}

	width, _ := m.resolvedDimensions()
	maxWidth := max(width-2, 20)
	minWidth := min(72, maxWidth)
	popupWidth := util.Clamp(int(float64(width)*0.8), minWidth, maxWidth)

	meta := strings.TrimSpace(string(m.specDiffUI.diff.FromFingerprint) + " -> " + string(m.specDiffUI.diff.ToFingerprint))
	popup := widgets.RenderPopup(widgets.PopupData{
		Title:       "Spec diff",
		Meta:        meta,
		Body:        specdiff.Render(m.specDiffUI.diff),
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
