package tui

import (
	"fmt"
	"strings"

	"github.com/phergul/apiscope/internal/model"
	statusbarui "github.com/phergul/apiscope/internal/tui/statusbar"
	"github.com/phergul/apiscope/internal/tui/widgets"
	"github.com/phergul/apiscope/internal/util"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) render() string {
	width, height := m.resolvedDimensions()
	if m.hasBlockingLoadError() {
		return m.renderBlockingLoadError(width, height)
	}

	preset := m.viewState.RightPaneLayoutPreset
	if preset == "" {
		preset = chooseLayoutPreset(width)
	}

	statusBar := m.renderStatusBar(width)
	bodyHeight := max(height-lipgloss.Height(statusBar), 12)

	if m.viewState.ZoomedPane {
		return lipgloss.JoinVertical(lipgloss.Left, m.renderZoomLayout(width, bodyHeight), statusBar)
	}

	var body string
	switch preset {
	case layoutPresetWide:
		body = m.renderWideLayout(width, bodyHeight)
	default:
		body = m.renderNarrowLayout(width, bodyHeight)
	}

	return lipgloss.JoinVertical(lipgloss.Left, body, statusBar)
}

func (m *Model) renderBlockingLoadError(width, height int) string {
	view := m.loadErrorView()
	popupWidth := util.Clamp(int(float64(width)*0.68), 56, 92)

	body := strings.Join([]string{
		view.Title,
		"",
		fmt.Sprintf("Category: %s", view.Category),
		fmt.Sprintf("Source: %s", fallbackText(view.Source, m.source)),
		"",
		view.Summary,
		"",
		fmt.Sprintf("Try this: %s", view.Hint),
		"",
		"[ Quit ]",
	}, "\n")

	popup := widgets.ModalStyle(max(popupWidth-4, 1)).Render(body)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, popup)
}

func (m *Model) renderWideLayout(width, height int) string {
	leftWidth := util.Clamp(int(float64(width)*0.32), 30, 40)
	leftWidth = min(leftWidth, width-20)
	rightWidth := max(width-leftWidth, 20)

	heights := computeWidePaneHeights(height)
	requestHeight, responseHeight := m.rightPaneHeights(heights)

	operationsView := m.paneView(model.FocusedPaneOperations)
	operationsView.Body = m.operationsPaneContentForSizeAndHeight(leftWidth, height)
	detailsView := m.paneView(model.FocusedPaneDetails)
	detailsView.Body = m.detailsPaneContentForSize(rightWidth, heights.Details)
	requestView := m.paneView(model.FocusedPaneRequest)
	requestView.Body = m.requestPaneContentForSize(rightWidth, requestHeight)
	responseView := m.paneView(model.FocusedPaneResponse)
	responseView.Body = m.responsePaneContentForSize(rightWidth, responseHeight)

	operationsPane := m.renderPane(operationsView.Title, operationsView.Body, operationsView.Footer, leftWidth, height, operationsView.Focused)

	rightParts := []string{
		m.renderPane(detailsView.Title, detailsView.Body, detailsView.Footer, rightWidth, heights.Details, detailsView.Focused),
		m.renderPane(requestView.Title, requestView.Body, requestView.Footer, rightWidth, requestHeight, requestView.Focused),
	}
	if responseHeight > 0 {
		rightParts = append(rightParts, m.renderPane(responseView.Title, responseView.Body, responseView.Footer, rightWidth, responseHeight, responseView.Focused))
	}

	rightColumn := lipgloss.JoinVertical(lipgloss.Left, rightParts...)

	return lipgloss.JoinHorizontal(lipgloss.Top, operationsPane, rightColumn)
}

func (m *Model) renderNarrowLayout(width, height int) string {
	heights := computeNarrowPaneHeights(height)
	requestHeight, responseHeight := m.rightPaneHeights(heights)

	operationsView := m.paneView(model.FocusedPaneOperations)
	operationsView.Body = m.operationsPaneContentForSizeAndHeight(width, heights.Operations)
	detailsView := m.paneView(model.FocusedPaneDetails)
	detailsView.Body = m.detailsPaneContentForSize(width, heights.Details)
	requestView := m.paneView(model.FocusedPaneRequest)
	requestView.Body = m.requestPaneContentForSize(width, requestHeight)
	responseView := m.paneView(model.FocusedPaneResponse)
	responseView.Body = m.responsePaneContentForSize(width, responseHeight)

	parts := []string{
		m.renderPane(operationsView.Title, operationsView.Body, operationsView.Footer, width, heights.Operations, operationsView.Focused),
		m.renderPane(detailsView.Title, detailsView.Body, detailsView.Footer, width, heights.Details, detailsView.Focused),
		m.renderPane(requestView.Title, requestView.Body, requestView.Footer, width, requestHeight, requestView.Focused),
	}
	if responseHeight > 0 {
		parts = append(parts, m.renderPane(responseView.Title, responseView.Body, responseView.Footer, width, responseHeight, responseView.Focused))
	}

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (m *Model) renderZoomLayout(width, height int) string {
	view := m.paneView(m.viewState.FocusedPane)
	if m.viewState.FocusedPane == model.FocusedPaneOperations {
		view.Body = m.operationsPaneContentForSizeAndHeight(width, height)
	}
	if m.viewState.FocusedPane == model.FocusedPaneDetails {
		view.Body = m.detailsPaneContentForSize(width, height)
	}
	if m.viewState.FocusedPane == model.FocusedPaneRequest {
		view.Body = m.requestPaneContentForSize(width, height)
	}
	if m.viewState.FocusedPane == model.FocusedPaneResponse {
		view.Body = m.responsePaneContentForSize(width, height)
	}
	return m.renderPane(view.Title, view.Body, view.Footer, width, height, true)
}

func (m *Model) renderPane(title, body, footer string, width, height int, focused bool) string {
	width = max(width, 12)
	height = max(height, 4)

	contentWidth := max(width-4, 1)
	contentHeight := max(height-2, 1)
	footerBlock := ""
	footerHeight := 0
	if strings.TrimSpace(footer) != "" {
		footerBlock = widgets.PaneFooterStyle(contentWidth).Render(footer)
		footerHeight = lipgloss.Height(footerBlock)
	}

	bodyHeight := max(contentHeight-footerHeight, 1)
	bodyBlock := widgets.PaneBodyStyle(contentWidth, bodyHeight).
		Render(body)

	content := bodyBlock
	if footerBlock != "" {
		content = lipgloss.JoinVertical(lipgloss.Left, bodyBlock, footerBlock)
	}

	return lipgloss.Place(width, height, lipgloss.Left, lipgloss.Top, widgets.RenderPaneFrame(title, content, width, focused))
}

func (m *Model) renderStatusBar(width int) string {
	line := statusbarui.Render(m.projectStatusBar())
	return widgets.StatusBarStyle(width).Render(line)
}

func (m *Model) resolvedDimensions() (int, int) {
	width := m.width
	height := m.height
	if width <= 0 {
		width = defaultWidth
	}
	if height <= 0 {
		height = defaultHeight
	}

	return width, height
}

func (m *Model) resolvedExpandedRightPane() model.FocusedPane {
	if m.viewState.ExpandedRightPane == model.FocusedPaneResponse {
		return model.FocusedPaneResponse
	}

	return model.FocusedPaneRequest
}

func (m *Model) rightPaneHeights(heights stackedPaneHeights) (int, int) {
	if m.resolvedExpandedRightPane() == model.FocusedPaneResponse {
		return heights.Folded, heights.Expanded
	}

	return heights.Expanded, heights.Folded
}
