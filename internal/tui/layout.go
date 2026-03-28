package tui

import (
	"fmt"
	"strings"

	"github.com/phergul/apiscope/internal/model"
	"github.com/phergul/apiscope/internal/tui/panes"
	"github.com/phergul/apiscope/internal/tui/widgets"

	"github.com/charmbracelet/lipgloss"
)

type layoutHeightPreset string

const (
	layoutHeightPresetCompact layoutHeightPreset = "compact"
	layoutHeightPresetNormal  layoutHeightPreset = "normal"
	layoutHeightPresetRoomy   layoutHeightPreset = "roomy"
)

type stackedPaneHeights struct {
	Operations int
	Details    int
	Expanded   int
	Folded     int
}

func chooseLayoutPreset(width int) string {
	if width >= 100 {
		return layoutPresetWide
	}

	return layoutPresetNarrow
}

func chooseHeightPreset(bodyHeight int) layoutHeightPreset {
	switch {
	case bodyHeight >= 28:
		return layoutHeightPresetRoomy
	case bodyHeight >= 20:
		return layoutHeightPresetNormal
	default:
		return layoutHeightPresetCompact
	}
}

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
	bodyHeight := maxInt(height-lipgloss.Height(statusBar), 12)

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
	popupWidth := clampInt(int(float64(width)*0.68), 56, 92)

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

	popup := widgets.ModalStyle(maxInt(popupWidth-4, 1)).Render(body)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, popup)
}

func (m *Model) renderWideLayout(width, height int) string {
	leftWidth := clampInt(int(float64(width)*0.32), 30, 40)
	leftWidth = minInt(leftWidth, width-20)
	rightWidth := maxInt(width-leftWidth, 20)

	heights := computeWidePaneHeights(height)
	requestHeight, responseHeight := m.rightPaneHeights(heights)

	operationsView := m.paneView(model.FocusedPaneOperations)
	detailsView := m.paneView(model.FocusedPaneDetails)
	detailsView.Body = m.detailsPaneContentForSize(rightWidth, heights.Details)
	requestView := m.paneView(model.FocusedPaneRequest)
	requestView.Body = m.requestPaneContentForSize(rightWidth, requestHeight)
	responseView := m.paneView(model.FocusedPaneResponse)

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
	detailsView := m.paneView(model.FocusedPaneDetails)
	detailsView.Body = m.detailsPaneContentForSize(width, heights.Details)
	requestView := m.paneView(model.FocusedPaneRequest)
	requestView.Body = m.requestPaneContentForSize(width, requestHeight)
	responseView := m.paneView(model.FocusedPaneResponse)

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
	if m.viewState.FocusedPane == model.FocusedPaneDetails {
		view.Body = m.detailsPaneContentForSize(width, height)
	}
	if m.viewState.FocusedPane == model.FocusedPaneRequest {
		view.Body = m.requestPaneContentForSize(width, height)
	}
	return m.renderPane(view.Title, view.Body, view.Footer, width, height, true)
}

func (m *Model) renderPane(title, body, footer string, width, height int, focused bool) string {
	width = maxInt(width, 12)
	height = maxInt(height, 4)

	titleLine := title

	contentWidth := maxInt(width-4, 1)
	contentHeight := maxInt(height-2, 1)
	footerBlock := ""
	footerHeight := 0
	if strings.TrimSpace(footer) != "" {
		footerBlock = widgets.PaneFooterStyle(contentWidth).Render(footer)
		footerHeight = lipgloss.Height(footerBlock)
	}

	bodyHeight := maxInt(contentHeight-footerHeight, 1)
	bodyBlock := widgets.PaneBodyStyle(contentWidth, bodyHeight).
		Render(titleLine + "\n\n" + body)

	content := bodyBlock
	if footerBlock != "" {
		content = lipgloss.JoinVertical(lipgloss.Left, bodyBlock, footerBlock)
	}

	return lipgloss.Place(width, height, lipgloss.Left, lipgloss.Top, widgets.PaneFrameStyle(focused).Render(content))
}

func (m *Model) renderStatusBar(width int) string {
	line := panes.RenderStatusBar(m.projectStatusBar())
	return widgets.StatusBarStyle(width).Render(line)
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

func computeWidePaneHeights(total int) stackedPaneHeights {
	var detailsTarget, foldedTarget int
	switch chooseHeightPreset(total) {
	case layoutHeightPresetRoomy:
		detailsTarget, foldedTarget = 9, 6
	case layoutHeightPresetNormal:
		detailsTarget, foldedTarget = 7, 5
	default:
		detailsTarget, foldedTarget = 5, 4
	}

	fixedHeights, expanded := allocateExpandedStackHeights(total, []int{detailsTarget, foldedTarget}, []int{4, 0}, 6, []int{1, 0})
	return stackedPaneHeights{
		Details:  fixedHeights[0],
		Expanded: expanded,
		Folded:   fixedHeights[1],
	}
}

func computeNarrowPaneHeights(total int) stackedPaneHeights {
	var operationsTarget, detailsTarget, foldedTarget int
	switch chooseHeightPreset(total) {
	case layoutHeightPresetRoomy:
		operationsTarget, detailsTarget, foldedTarget = 10, 8, 6
	case layoutHeightPresetNormal:
		operationsTarget, detailsTarget, foldedTarget = 8, 6, 5
	default:
		operationsTarget, detailsTarget, foldedTarget = 6, 5, 4
	}

	fixedHeights, expanded := allocateExpandedStackHeights(total, []int{operationsTarget, detailsTarget, foldedTarget}, []int{4, 4, 0}, 6, []int{2, 1, 0})
	return stackedPaneHeights{
		Operations: fixedHeights[0],
		Details:    fixedHeights[1],
		Expanded:   expanded,
		Folded:     fixedHeights[2],
	}
}

func allocateExpandedStackHeights(total int, fixedTargets, fixedMinimums []int, expandedMinimum int, compressionOrder []int) ([]int, int) {
	fixedHeights := append([]int(nil), fixedTargets...)
	expanded := total - sumInts(fixedHeights)
	if expanded >= expandedMinimum {
		return fixedHeights, expanded
	}

	deficit := expandedMinimum - expanded
	for _, index := range compressionOrder {
		if index < 0 || index >= len(fixedHeights) || index >= len(fixedMinimums) {
			continue
		}

		reducible := fixedHeights[index] - fixedMinimums[index]
		if reducible <= 0 {
			continue
		}

		delta := minInt(deficit, reducible)
		fixedHeights[index] -= delta
		deficit -= delta
		if deficit == 0 {
			break
		}
	}

	return fixedHeights, total - sumInts(fixedHeights)
}

func clampInt(value, minimum, maximum int) int {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}

	return value
}

func minInt(left, right int) int {
	if left < right {
		return left
	}

	return right
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}

	return right
}

func sumInts(values []int) int {
	total := 0
	for _, value := range values {
		total += value
	}

	return total
}
