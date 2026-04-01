package tui

import (
	"strings"

	"github.com/phergul/apiscope/internal/model"
	statusbarui "github.com/phergul/apiscope/internal/tui/statusbar"
	"github.com/phergul/apiscope/internal/tui/widgets"
	"github.com/phergul/apiscope/internal/util"

	"github.com/charmbracelet/lipgloss"
)

// paneLayoutSpec describes one pane rendered by the root shell layout.
type paneLayoutSpec struct {
	pane   model.FocusedPane
	width  int
	height int
}

// render renders the full root shell, including panes, status bar, and overlays.
func (m *Model) render() string {
	width, height := m.resolvedDimensions()
	statusBar := m.renderStatusBar(width)
	bodyHeight := max(height-m.statusBarHeight(width), 12)
	helpOverlay := m.projectHelpOverlay()

	var body string
	if m.hasBlockingLoadError() {
		body = m.renderBlockingLoadError(width, bodyHeight)
	} else {
		preset := m.viewState.RightPaneLayoutPreset
		if preset == "" {
			preset = chooseLayoutPreset(width)
		}

		if m.viewState.ZoomedPane {
			body = m.renderZoomLayout(width, bodyHeight)
		} else {
			body = m.renderPresetLayout(preset, width, bodyHeight)
		}
	}

	view := lipgloss.JoinVertical(lipgloss.Left, body, statusBar)
	view = m.renderHistoryPopup(view)
	return m.renderHelpOverlay(view, width, helpOverlay)
}

// renderPresetLayout renders the shell body for the requested layout preset.
func (m *Model) renderPresetLayout(preset string, width, height int) string {
	switch preset {
	case layoutPresetWide:
		return m.renderWideLayout(width, height)
	default:
		return m.renderNarrowLayout(width, height)
	}
}

// renderWideLayout renders the wide two-column shell layout.
func (m *Model) renderWideLayout(width, height int) string {
	leftWidth, rightWidth := m.wideColumnWidths(width)
	heights := computeWidePaneHeights(height)
	requestHeight, responseHeight := m.rightPaneHeights(heights)

	operationsPane := m.renderSizedPane(paneLayoutSpec{
		pane:   model.FocusedPaneOperations,
		width:  leftWidth,
		height: height,
	})
	rightColumn := m.renderPaneStack([]paneLayoutSpec{
		{pane: model.FocusedPaneDetails, width: rightWidth, height: heights.Details},
		{pane: model.FocusedPaneRequest, width: rightWidth, height: requestHeight},
		{pane: model.FocusedPaneResponse, width: rightWidth, height: responseHeight},
	})

	return lipgloss.JoinHorizontal(lipgloss.Top, operationsPane, rightColumn)
}

// renderNarrowLayout renders the vertically stacked shell layout.
func (m *Model) renderNarrowLayout(width, height int) string {
	heights := computeNarrowPaneHeights(height)
	requestHeight, responseHeight := m.rightPaneHeights(heights)

	return m.renderPaneStack([]paneLayoutSpec{
		{pane: model.FocusedPaneOperations, width: width, height: heights.Operations},
		{pane: model.FocusedPaneDetails, width: width, height: heights.Details},
		{pane: model.FocusedPaneRequest, width: width, height: requestHeight},
		{pane: model.FocusedPaneResponse, width: width, height: responseHeight},
	})
}

// renderZoomLayout renders the focused pane as a single full-shell pane.
func (m *Model) renderZoomLayout(width, height int) string {
	return m.renderSizedPane(paneLayoutSpec{
		pane:   m.viewState.FocusedPane,
		width:  width,
		height: height,
	})
}

// renderPaneStack renders a vertical stack of pane layout specs, skipping collapsed panes.
func (m *Model) renderPaneStack(specs []paneLayoutSpec) string {
	parts := make([]string, 0, len(specs))
	for _, spec := range specs {
		if spec.height <= 0 {
			continue
		}
		parts = append(parts, m.renderSizedPane(spec))
	}

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// renderSizedPane resolves and renders one pane for the provided shell size.
func (m *Model) renderSizedPane(spec paneLayoutSpec) string {
	view := m.sizedPaneView(spec.pane, spec.width, spec.height)
	return m.renderPaneView(view, spec.width, spec.height)
}

// sizedPaneView resolves one pane view with its body content for the provided size.
func (m *Model) sizedPaneView(pane model.FocusedPane, width, height int) paneView {
	view := m.paneView(pane)
	view.Body = m.paneBodyForSize(pane, width, height)
	return view
}

// paneBodyForSize resolves pane body content for the provided shell size.
func (m *Model) paneBodyForSize(pane model.FocusedPane, width, height int) string {
	switch pane {
	case model.FocusedPaneDetails:
		return m.detailsPaneContentForSize(width, height)
	case model.FocusedPaneRequest:
		return m.requestPaneContentForSize(width, height)
	case model.FocusedPaneResponse:
		return m.responsePaneContentForSize(width, height)
	default:
		return m.operationsPaneContentForSizeAndHeight(width, height)
	}
}

// renderPaneView renders one prepared pane view inside the shell frame.
func (m *Model) renderPaneView(view paneView, width, height int) string {
	return m.renderPane(view.TitleLeft, view.Title, view.TitleRight, view.Body, view.Footer, width, height, view.Focused)
}

// wideColumnWidths returns the left and right column widths for the wide shell layout.
func (m *Model) wideColumnWidths(width int) (int, int) {
	// keep the operations pane around one third of the screen, but inside readable bounds.
	leftWidth := util.Clamp(int(float64(width)*0.32), 30, 40)
	// preserve a minimum right-side working area for the stacked panes.
	leftWidth = min(leftWidth, width-20)
	return leftWidth, max(width-leftWidth, 20)
}

// renderPane renders one pane frame with optional footer content.
func (m *Model) renderPane(titleLeft, title, titleRight, body, footer string, width, height int, focused bool) string {
	width = max(width, 12)
	height = max(height, 4)

	// subtract the pane frame padding and borders before sizing the content blocks.
	contentWidth := max(width-4, 1)
	contentHeight := max(height-2, 1)
	footerBlock := ""
	footerHeight := 0
	if strings.TrimSpace(footer) != "" {
		footerBlock = widgets.PaneFooterStyle(contentWidth).Render(footer)
		footerHeight = lipgloss.Height(footerBlock)
	}

	bodyHeight := max(contentHeight-footerHeight, 1)
	bodyBlock := widgets.PaneBodyStyle(contentWidth, bodyHeight).Render(body)

	content := bodyBlock
	if footerBlock != "" {
		content = lipgloss.JoinVertical(lipgloss.Left, bodyBlock, footerBlock)
	}

	return lipgloss.Place(width, height, lipgloss.Left, lipgloss.Top, widgets.RenderPaneFrame(titleLeft, title, titleRight, content, width, focused))
}

// renderStatusBar renders the shell status bar for the current width.
func (m *Model) renderStatusBar(width int) string {
	line := statusbarui.Render(m.projectStatusBar(), width)
	return widgets.StatusBarStyle(width).Render(line)
}

// statusBarHeight returns the rendered shell status-bar height for the current width.
func (m *Model) statusBarHeight(width int) int {
	return lipgloss.Height(m.renderStatusBar(width))
}

// resolvedDimensions returns the current shell size, falling back to sensible defaults.
func (m *Model) resolvedDimensions() (int, int) {
	width := m.shell.width
	height := m.shell.height
	if width <= 0 {
		width = defaultWidth
	}
	if height <= 0 {
		height = defaultHeight
	}

	return width, height
}

// resolvedExpandedRightPane returns the right pane that should receive the expanded height.
func (m *Model) resolvedExpandedRightPane() model.FocusedPane {
	if m.viewState.ExpandedRightPane == model.FocusedPaneResponse {
		return model.FocusedPaneResponse
	}

	return model.FocusedPaneRequest
}

// rightPaneHeights maps expanded and folded heights to request and response panes.
func (m *Model) rightPaneHeights(heights stackedPaneHeights) (int, int) {
	if m.resolvedExpandedRightPane() == model.FocusedPaneResponse {
		return heights.Folded, heights.Expanded
	}

	return heights.Expanded, heights.Folded
}

// responsePaneSize returns the rendered response pane size for the current shell layout.
func (m *Model) responsePaneSize() (int, int) {
	width, height := m.resolvedDimensions()
	bodyHeight := max(height-m.statusBarHeight(width), 12)
	if m.viewState.ZoomedPane && m.viewState.FocusedPane == model.FocusedPaneResponse {
		return width, bodyHeight
	}

	preset := m.viewState.RightPaneLayoutPreset
	if preset == "" {
		preset = chooseLayoutPreset(width)
	}

	if preset == layoutPresetWide {
		_, rightWidth := m.wideColumnWidths(width)
		_, responseHeight := m.rightPaneHeights(computeWidePaneHeights(bodyHeight))
		return rightWidth, responseHeight
	}

	_, responseHeight := m.rightPaneHeights(computeNarrowPaneHeights(bodyHeight))
	return width, responseHeight
}
