package tui

import (
	"strings"

	"github.com/phergul/apiscope/internal/tui/widgets"
	"github.com/phergul/apiscope/internal/util"

	"github.com/charmbracelet/lipgloss"
)

type helpOverlayView struct {
	Title string
	Body  string
	Hint  string
}

func (m *Model) projectHelpOverlay() helpOverlayView {
	if overlay := m.currentRequestHelpOverlay(); overlay.Hint != "" || overlay.Title != "" || overlay.Body != "" {
		return overlay
	}

	return helpOverlayView{}
}

func (m *Model) renderHelpOverlay(view string, width int, overlay helpOverlayView) string {
	if strings.TrimSpace(overlay.Body) == "" {
		return view
	}

	popup := widgets.RenderPopup(widgets.PopupData{
		Title:   overlay.Title,
		Body:    overlay.Body,
		Width:   helpOverlayWidth(width, overlay.Title, overlay.Body),
		Focused: false,
	})

	x := max(width-lipgloss.Width(popup), 0)
	y := max(lipgloss.Height(view)-lipgloss.Height(m.renderStatusBar(width))-lipgloss.Height(popup), 0)
	return widgets.Overlay(view, popup, x, y)
}

func helpOverlayWidth(totalWidth int, title, body string) int {
	maxLineWidth := lipgloss.Width(strings.TrimSpace(title))
	for _, line := range strings.Split(strings.TrimSpace(body), "\n") {
		if lineWidth := lipgloss.Width(line); lineWidth > maxLineWidth {
			maxLineWidth = lineWidth
		}
	}

	return util.Clamp(maxLineWidth+4, 20, max(totalWidth-2, 20))
}
