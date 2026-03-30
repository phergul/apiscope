package tui

import (
	"strings"

	"github.com/phergul/apiscope/internal/model"
	detailsui "github.com/phergul/apiscope/internal/tui/details"
	operationsui "github.com/phergul/apiscope/internal/tui/operations"
	requestui "github.com/phergul/apiscope/internal/tui/request"
	responseui "github.com/phergul/apiscope/internal/tui/response"
	"github.com/phergul/apiscope/internal/tui/widgets"
	"github.com/phergul/apiscope/internal/util"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) syncVisibleOperations() {
	if m.session.Spec == nil {
		m.viewState.VisibleOperationKeys = nil
		m.viewState.OperationsCursor = 0
		m.viewState.OperationsScrollOffset = 0
		m.session.SelectedOperationKey = ""
		m.syncActivePaneSections()
		return
	}

	filter := strings.TrimSpace(strings.ToLower(m.viewState.FilterText))
	visible := make([]model.OperationKey, 0, len(m.session.Spec.Operations))
	for _, operation := range m.session.Spec.Operations {
		if filter == "" || operationsui.MatchFilter(operation, filter) {
			visible = append(visible, operation.Key)
		}
	}

	m.viewState.VisibleOperationKeys = m.groupOperationKeys(visible)
	m.syncSelectedOperationAfterVisibilityChange()
}

func (m *Model) syncSelectedOperationAfterVisibilityChange() {
	previous := m.session.SelectedOperationKey
	if len(m.viewState.VisibleOperationKeys) == 0 {
		m.session.SelectedOperationKey = ""
		m.viewState.OperationsCursor = 0
		m.viewState.OperationsScrollOffset = 0
		m.onSelectionChanged(previous, "")
		return
	}

	for index, key := range m.viewState.VisibleOperationKeys {
		if key == m.session.SelectedOperationKey {
			m.viewState.OperationsCursor = index
			m.ensureActiveOperationVisible()
			m.syncActiveDetailsSection()
			return
		}
	}

	m.session.SelectedOperationKey = m.viewState.VisibleOperationKeys[0]
	m.viewState.OperationsCursor = 0
	m.ensureActiveOperationVisible()
	m.onSelectionChanged(previous, m.session.SelectedOperationKey)
}

func (m *Model) setSelectedOperationByVisibleIndex(index int) {
	previous := m.session.SelectedOperationKey
	if len(m.viewState.VisibleOperationKeys) == 0 {
		m.session.SelectedOperationKey = ""
		m.viewState.OperationsCursor = 0
		m.viewState.OperationsScrollOffset = 0
		m.onSelectionChanged(previous, "")
		return
	}

	if index < 0 {
		index = 0
	}
	if index >= len(m.viewState.VisibleOperationKeys) {
		index = len(m.viewState.VisibleOperationKeys) - 1
	}

	m.viewState.OperationsCursor = index
	m.session.SelectedOperationKey = m.viewState.VisibleOperationKeys[index]
	m.ensureActiveOperationVisible()
	m.onSelectionChanged(previous, m.session.SelectedOperationKey)
}

func (m *Model) groupedVisibleOperations() []operationsui.KeyGroup {
	return operationsui.GroupKeys(m.viewState.VisibleOperationKeys, m.operationByKey)
}

func (m *Model) groupOperationKeys(keys []model.OperationKey) []model.OperationKey {
	return operationsui.FlattenKeys(operationsui.GroupKeys(keys, m.operationByKey))
}

func (m *Model) jumpToAdjacentOperationGroup(direction int) {
	groups := m.groupedVisibleOperations()
	if len(groups) == 0 {
		return
	}

	currentKey := m.session.SelectedOperationKey
	currentGroupIndex := -1
	for index, group := range groups {
		for _, key := range group.Keys {
			if key == currentKey {
				currentGroupIndex = index
				break
			}
		}
		if currentGroupIndex >= 0 {
			break
		}
	}
	if currentGroupIndex < 0 {
		currentGroupIndex = 0
	}

	targetIndex := currentGroupIndex + direction
	if targetIndex < 0 || targetIndex >= len(groups) || len(groups[targetIndex].Keys) == 0 {
		return
	}

	targetKey := groups[targetIndex].Keys[0]
	for visibleIndex, key := range m.viewState.VisibleOperationKeys {
		if key == targetKey {
			m.setSelectedOperationByVisibleIndex(visibleIndex)
			return
		}
	}
}

func (m *Model) ensureActiveOperationVisible() {
	if len(m.viewState.VisibleOperationKeys) == 0 {
		m.viewState.OperationsCursor = 0
		m.viewState.OperationsScrollOffset = 0
		return
	}
	if m.viewState.OperationsCursor < 0 {
		m.viewState.OperationsCursor = 0
	}
	if m.viewState.OperationsCursor >= len(m.viewState.VisibleOperationKeys) {
		m.viewState.OperationsCursor = len(m.viewState.VisibleOperationKeys) - 1
	}

	totalRows := len(m.viewState.VisibleOperationKeys)
	maxOffset := totalRows - 1
	if maxOffset < 0 {
		maxOffset = 0
	}
	endAlignedOffset := m.maxOperationsScrollOffset()
	if endAlignedOffset < maxOffset {
		maxOffset = endAlignedOffset
	}
	m.viewState.OperationsScrollOffset = util.Clamp(m.viewState.OperationsScrollOffset, 0, maxOffset)

	scrolloff := 5
	for range 3 {
		visibleRows := m.visibleOperationRowCount(m.viewState.OperationsScrollOffset)
		if visibleRows <= 0 {
			m.viewState.OperationsScrollOffset = 0
			return
		}

		maxScrolloff := max(visibleRows-1, 0)
		if scrolloff > maxScrolloff {
			scrolloff = maxScrolloff
		}

		minCursor := m.viewState.OperationsScrollOffset + scrolloff
		maxCursor := m.viewState.OperationsScrollOffset + visibleRows - scrolloff - 1
		if maxCursor < minCursor {
			maxCursor = minCursor
		}

		nextOffset := m.viewState.OperationsScrollOffset
		switch {
		case m.viewState.OperationsCursor < minCursor:
			nextOffset = m.viewState.OperationsCursor - scrolloff
		case m.viewState.OperationsCursor > maxCursor:
			nextOffset = m.viewState.OperationsCursor - visibleRows + scrolloff + 1
		default:
			return
		}

		nextOffset = util.Clamp(nextOffset, 0, maxOffset)
		if nextOffset == m.viewState.OperationsScrollOffset {
			return
		}
		m.viewState.OperationsScrollOffset = nextOffset
	}
}

func (m *Model) maxOperationsScrollOffset() int {
	totalRows := len(m.viewState.VisibleOperationKeys)
	if totalRows <= 1 {
		return 0
	}

	for offset := 0; offset < totalRows; offset++ {
		if m.visibleOperationRowCount(offset) == totalRows-offset {
			return offset
		}
	}

	return totalRows - 1
}

func (m *Model) availableDetailsSections() []string {
	var warnings []model.SpecWarning
	if m.session.Spec != nil {
		warnings = m.session.Spec.Warnings
	}

	return detailsui.AvailableSections(
		m.resolvedSelectedOperation(),
		m.effectiveSecurityRequirement(m.resolvedSelectedOperation()),
		warnings,
	)
}

func (m *Model) syncActiveDetailsSection() {
	available := m.availableDetailsSections()
	m.activeDetailsSection = widgets.ResolveActiveSection(m.activeDetailsSection, available, detailsui.SectionSummary)
}

func (m *Model) availableRequestSections() []string {
	return requestui.AvailableSections(m.resolvedSelectedOperation(), m.effectiveSecurityRequirement(m.resolvedSelectedOperation()))
}

func (m *Model) resetActiveRequestSection() {
	available := m.availableRequestSections()
	m.activeRequestSection = widgets.ResolveActiveSection("", available, "")
	m.resetRequestCursorAndScroll()
	m.clearRequestValidation()
}

func (m *Model) moveRequestSection(direction int) {
	available := m.availableRequestSections()
	m.activeRequestSection = widgets.MoveActiveSection(m.activeRequestSection, available, direction, "")
	m.resetRequestCursorAndScroll()
	m.clearRequestValidation()
}

func (m *Model) setRequestSectionBoundary(last bool) {
	available := m.availableRequestSections()
	m.activeRequestSection = widgets.BoundaryActiveSection(available, last, "")
	m.resetRequestCursorAndScroll()
	m.clearRequestValidation()
}

func (m *Model) availableResponseSections() []string {
	selected := m.resolvedSelectedOperation()
	if selected == nil {
		return nil
	}

	return responseui.AvailableSections(selected.Responses)
}

func (m *Model) resetActiveResponseSection() {
	available := m.availableResponseSections()
	m.activeResponseSection = widgets.ResolveActiveSection("", available, "")
	m.viewState.ResponseScrollOffset = 0
}

func (m *Model) moveResponseSection(direction int) {
	available := m.availableResponseSections()
	m.activeResponseSection = widgets.MoveActiveSection(m.activeResponseSection, available, direction, "")
	m.viewState.ResponseScrollOffset = 0
}

func (m *Model) setResponseSectionBoundary(last bool) {
	available := m.availableResponseSections()
	m.activeResponseSection = widgets.BoundaryActiveSection(available, last, "")
	m.viewState.ResponseScrollOffset = 0
}

func (m *Model) syncActivePaneSections() {
	m.syncActiveDetailsSection()
	m.resetActiveRequestSection()
	m.resetActiveResponseSection()
	m.viewState.DetailsScrollOffset = 0
	m.ensureSelectedRequestDraft()
	m.clearRequestValidation()
}

func (m *Model) onSelectionChanged(previous, current model.OperationKey) {
	m.syncActiveDetailsSection()
	if previous != current {
		m.ensureSelectedRequestDraft()
		m.resetActiveRequestSection()
		m.resetActiveResponseSection()
		m.viewState.DetailsScrollOffset = 0
		m.clearRequestValidation()
	}
}

func (m *Model) moveDetailsSection(direction int) {
	available := m.availableDetailsSections()
	m.activeDetailsSection = widgets.MoveActiveSection(m.activeDetailsSection, available, direction, detailsui.SectionSummary)
	m.viewState.DetailsScrollOffset = 0
}

func (m *Model) setDetailsSectionBoundary(last bool) {
	available := m.availableDetailsSections()
	m.activeDetailsSection = widgets.BoundaryActiveSection(available, last, detailsui.SectionSummary)
	m.viewState.DetailsScrollOffset = 0
}

func (m *Model) scrollDetailsBy(delta int) {
	maxOffset := m.maxDetailsScrollOffset()
	target := m.viewState.DetailsScrollOffset + delta
	if target < 0 {
		target = 0
	}
	if target > maxOffset {
		target = maxOffset
	}

	m.viewState.DetailsScrollOffset = target
}

func (m *Model) scrollDetailsToBoundary(last bool) {
	if last {
		m.viewState.DetailsScrollOffset = m.maxDetailsScrollOffset()
		return
	}

	m.viewState.DetailsScrollOffset = 0
}

func (m *Model) scrollResponseBy(delta int) {
	maxOffset := m.maxResponseScrollOffset()
	target := m.viewState.ResponseScrollOffset + delta
	if target < 0 {
		target = 0
	}
	if target > maxOffset {
		target = maxOffset
	}

	m.viewState.ResponseScrollOffset = target
}

func (m *Model) scrollResponseToBoundary(last bool) {
	if last {
		m.viewState.ResponseScrollOffset = m.maxResponseScrollOffset()
		return
	}

	m.viewState.ResponseScrollOffset = 0
}

func (m *Model) maxResponseScrollOffset() int {
	lines := len(splitLines(responseui.ActiveSectionBody(m.projectResponsePane().Sections, m.activeResponseSection)))
	visible := m.responseVisibleBodyLines()
	if lines <= visible {
		return 0
	}

	return lines - visible
}

func (m *Model) responseVisibleBodyLines() int {
	width, height := m.resolvedDimensions()
	bodyHeight := max(height-lipgloss.Height(m.renderStatusBar(width)), 12)

	var paneHeight int
	if m.viewState.RightPaneLayoutPreset == layoutPresetWide {
		_, responseHeight := m.rightPaneHeights(computeWidePaneHeights(bodyHeight))
		paneHeight = responseHeight
	} else {
		_, responseHeight := m.rightPaneHeights(computeNarrowPaneHeights(bodyHeight))
		paneHeight = responseHeight
	}

	return max(paneHeight-6, 1)
}

func (m *Model) maxDetailsScrollOffset() int {
	data := m.projectDetailsPane()
	lines := len(splitLines(detailsui.RenderActiveSection(data)))
	visible := m.detailsVisibleBodyLines()
	if lines <= visible {
		return 0
	}

	return lines - visible
}

func (m *Model) detailsVisibleBodyLines() int {
	width, height := m.resolvedDimensions()
	bodyHeight := max(height-lipgloss.Height(m.renderStatusBar(width)), 12)
	var paneHeight int
	if m.viewState.RightPaneLayoutPreset == layoutPresetWide {
		paneHeight = computeWidePaneHeights(bodyHeight).Details
	} else {
		paneHeight = computeNarrowPaneHeights(bodyHeight).Details
	}

	return max(paneHeight-6, 1)
}
