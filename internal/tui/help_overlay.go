package tui

import (
	"strings"

	"github.com/phergul/apiscope/internal/tui/widgets"
	"github.com/phergul/apiscope/internal/util"

	"github.com/charmbracelet/lipgloss"
)

// helpOverlayView describes a shell-level help overlay projection.
type helpOverlayView struct {
	Title string
	Body  string
	Hint  string
}

// projectHelpOverlay projects the currently active shell help overlay.
func (m *Model) projectHelpOverlay() helpOverlayView {
	if overlay := m.currentRequestHelpOverlay(); overlay.Hint != "" || overlay.Title != "" || overlay.Body != "" {
		return overlay
	}

	return helpOverlayView{}
}

// renderHelpOverlay renders the current help overlay above the status bar when needed.
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
