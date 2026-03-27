package tui

import (
	"fmt"
	"strings"

	"api-tui/internal/model"
	"api-tui/internal/tui/panes"

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

	popup := lipgloss.NewStyle().
		Width(maxInt(popupWidth-4, 1)).
		Border(paneBorder).
		Padding(1, 2).
		Render(body)

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
	requestView := m.paneView(model.FocusedPaneRequest)
	responseView := m.paneView(model.FocusedPaneResponse)

	operationsPane := m.renderPane(operationsView.Title, operationsView.Body, leftWidth, height, operationsView.Focused)

	rightParts := []string{
		m.renderPane(detailsView.Title, detailsView.Body, rightWidth, heights.Details, detailsView.Focused),
		m.renderPane(requestView.Title, requestView.Body, rightWidth, requestHeight, requestView.Focused),
	}
	if responseHeight > 0 {
		rightParts = append(rightParts, m.renderPane(responseView.Title, responseView.Body, rightWidth, responseHeight, responseView.Focused))
	}

	rightColumn := lipgloss.JoinVertical(lipgloss.Left, rightParts...)

	return lipgloss.JoinHorizontal(lipgloss.Top, operationsPane, rightColumn)
}

func (m *Model) renderNarrowLayout(width, height int) string {
	heights := computeNarrowPaneHeights(height)
	requestHeight, responseHeight := m.rightPaneHeights(heights)

	operationsView := m.paneView(model.FocusedPaneOperations)
	detailsView := m.paneView(model.FocusedPaneDetails)
	requestView := m.paneView(model.FocusedPaneRequest)
	responseView := m.paneView(model.FocusedPaneResponse)

	parts := []string{
		m.renderPane(operationsView.Title, operationsView.Body, width, heights.Operations, operationsView.Focused),
		m.renderPane(detailsView.Title, detailsView.Body, width, heights.Details, detailsView.Focused),
		m.renderPane(requestView.Title, requestView.Body, width, requestHeight, requestView.Focused),
	}
	if responseHeight > 0 {
		parts = append(parts, m.renderPane(responseView.Title, responseView.Body, width, responseHeight, responseView.Focused))
	}

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (m *Model) renderZoomLayout(width, height int) string {
	view := m.paneView(m.viewState.FocusedPane)
	return m.renderPane(view.Title, view.Body, width, height, true)
}

func (m *Model) renderPane(title, body string, width, height int, focused bool) string {
	width = maxInt(width, 12)
	height = maxInt(height, 4)

	titleLine := title
	if focused {
		titleLine = "> " + title
	}

	contentWidth := maxInt(width-4, 1)
	contentHeight := maxInt(height-2, 1)
	content := lipgloss.NewStyle().
		Width(contentWidth).
		Height(contentHeight).
		MaxWidth(contentWidth).
		MaxHeight(contentHeight).
		Render(titleLine + "\n\n" + body)

	style := lipgloss.NewStyle().
		Border(paneBorder).
		Padding(0, 1)
	if focused {
		style = style.Bold(true)
	}

	return lipgloss.Place(width, height, lipgloss.Left, lipgloss.Top, style.Render(content))
}

func (m *Model) renderStatusBar(width int) string {
	line := panes.RenderStatusBar(m.projectStatusBar())

	return lipgloss.NewStyle().
		BorderTop(true).
		BorderStyle(paneBorder).
		Width(width).
		Render(line)
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
