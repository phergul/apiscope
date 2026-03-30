package tui

import (
	"github.com/phergul/apiscope/internal/model"
	detailsui "github.com/phergul/apiscope/internal/tui/details"
	operationsui "github.com/phergul/apiscope/internal/tui/operations"
	requestui "github.com/phergul/apiscope/internal/tui/request"
	responseui "github.com/phergul/apiscope/internal/tui/response"
)

// syncVisibleOperations refreshes the visible operations list from the current filter text.
func (m *Model) syncVisibleOperations() {
	if m.session.Spec == nil {
		m.viewState.VisibleOperationKeys = nil
		listState := operationsui.ResetListState()
		m.viewState.OperationsCursor = listState.Cursor
		m.viewState.OperationsScrollOffset = listState.ScrollOffset
		m.session.SelectedOperationKey = ""
		m.syncActivePaneSections()
		return
	}

	m.viewState.VisibleOperationKeys = operationsui.FilterVisibleKeys(m.session.Spec.Operations, m.viewState.FilterText)
	m.syncSelectedOperationAfterVisibilityChange()
}

// syncSelectedOperationAfterVisibilityChange keeps selection and cursor state valid after filtering.
func (m *Model) syncSelectedOperationAfterVisibilityChange() {
	previous := m.session.SelectedOperationKey
	if len(m.viewState.VisibleOperationKeys) == 0 {
		m.session.SelectedOperationKey = ""
		listState := operationsui.ResetListState()
		m.viewState.OperationsCursor = listState.Cursor
		m.viewState.OperationsScrollOffset = listState.ScrollOffset
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

// setSelectedOperationByVisibleIndex selects the operation at the requested visible index.
func (m *Model) setSelectedOperationByVisibleIndex(index int) {
	previous := m.session.SelectedOperationKey
	if len(m.viewState.VisibleOperationKeys) == 0 {
		m.session.SelectedOperationKey = ""
		listState := operationsui.ResetListState()
		m.viewState.OperationsCursor = listState.Cursor
		m.viewState.OperationsScrollOffset = listState.ScrollOffset
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

// jumpToAdjacentOperationGroup moves selection to the first row of the adjacent operations group.
func (m *Model) jumpToAdjacentOperationGroup(direction int) {
	if m.session.Spec == nil {
		return
	}

	targetKey := operationsui.AdjacentGroupTarget(m.session.Spec.Operations, m.viewState.VisibleOperationKeys, m.session.SelectedOperationKey, direction)
	if targetKey == "" {
		return
	}

	for visibleIndex, key := range m.viewState.VisibleOperationKeys {
		if key == targetKey {
			m.setSelectedOperationByVisibleIndex(visibleIndex)
			return
		}
	}
}

// ensureActiveOperationVisible keeps the selected operations row within the rendered list window.
func (m *Model) ensureActiveOperationVisible() {
	if m.session.Spec == nil {
		listState := operationsui.ResetListState()
		m.viewState.OperationsCursor = listState.Cursor
		m.viewState.OperationsScrollOffset = listState.ScrollOffset
		return
	}

	contentWidth, maxLines := m.operationsPaneMetrics()
	listState := operationsui.SyncListState(operationsui.StateInput{
		Operations:   m.session.Spec.Operations,
		VisibleKeys:  m.viewState.VisibleOperationKeys,
		ContentWidth: contentWidth,
		MaxLines:     maxLines,
	}, operationsui.ListState{
		Cursor:       m.viewState.OperationsCursor,
		ScrollOffset: m.viewState.OperationsScrollOffset,
	})
	m.viewState.OperationsCursor = listState.Cursor
	m.viewState.OperationsScrollOffset = listState.ScrollOffset
}

// maxOperationsScrollOffset returns the largest valid operations scroll offset.
func (m *Model) maxOperationsScrollOffset() int {
	if m.session.Spec == nil {
		return 0
	}

	contentWidth, maxLines := m.operationsPaneMetrics()
	return operationsui.MaxScrollOffset(operationsui.PaneInput{
		HasSpec:      true,
		Operations:   m.session.Spec.Operations,
		VisibleKeys:  m.viewState.VisibleOperationKeys,
		ContentWidth: contentWidth,
		MaxLines:     maxLines,
	})
}

// syncActiveDetailsSection clamps the active details section to the currently visible sections.
func (m *Model) syncActiveDetailsSection() {
	var warnings []model.SpecWarning
	if m.session.Spec != nil {
		warnings = m.session.Spec.Warnings
	}

	m.panes.activeDetailsSection = detailsui.ResolveActiveSection(
		m.panes.activeDetailsSection,
		m.resolvedSelectedOperation(),
		m.effectiveSecurityRequirement(m.resolvedSelectedOperation()),
		warnings,
	)
}

// availableRequestSections returns the visible request sections for the selected operation.
func (m *Model) availableRequestSections() []string {
	return requestui.AvailableSections(m.resolvedSelectedOperation(), m.effectiveSecurityRequirement(m.resolvedSelectedOperation()))
}

// resetActiveRequestSection resets the active request section to the first available section.
func (m *Model) resetActiveRequestSection() {
	m.panes.activeRequestSection = requestui.ResolveActiveSection("", m.resolvedSelectedOperation(), m.effectiveSecurityRequirement(m.resolvedSelectedOperation()))
	m.resetRequestCursorAndScroll()
	m.clearRequestValidation()
}

// moveRequestSection moves the active request section by the given direction.
func (m *Model) moveRequestSection(direction int) {
	m.panes.activeRequestSection = requestui.MoveActiveSection(m.panes.activeRequestSection, direction, m.resolvedSelectedOperation(), m.effectiveSecurityRequirement(m.resolvedSelectedOperation()))
	m.resetRequestCursorAndScroll()
	m.clearRequestValidation()
}

// setRequestSectionBoundary moves the active request section to the first or last section.
func (m *Model) setRequestSectionBoundary(last bool) {
	m.panes.activeRequestSection = requestui.BoundaryActiveSection(last, m.resolvedSelectedOperation(), m.effectiveSecurityRequirement(m.resolvedSelectedOperation()))
	m.resetRequestCursorAndScroll()
	m.clearRequestValidation()
}

// resetActiveResponseSection resets the active response section to the first available section.
func (m *Model) resetActiveResponseSection() {
	selected := m.resolvedSelectedOperation()
	if selected == nil {
		m.panes.activeResponseSection = ""
		m.viewState.ResponseScrollOffset = 0
		return
	}

	m.panes.activeResponseSection = responseui.ResolveActiveSection("", selected.Responses)
	m.viewState.ResponseScrollOffset = 0
}

// moveResponseSection moves the active response section by the given direction.
func (m *Model) moveResponseSection(direction int) {
	selected := m.resolvedSelectedOperation()
	if selected == nil {
		m.panes.activeResponseSection = ""
		m.viewState.ResponseScrollOffset = 0
		return
	}

	m.panes.activeResponseSection = responseui.MoveActiveSection(m.panes.activeResponseSection, direction, selected.Responses)
	m.viewState.ResponseScrollOffset = 0
}

// setResponseSectionBoundary moves the active response section to the first or last section.
func (m *Model) setResponseSectionBoundary(last bool) {
	selected := m.resolvedSelectedOperation()
	if selected == nil {
		m.panes.activeResponseSection = ""
		m.viewState.ResponseScrollOffset = 0
		return
	}

	m.panes.activeResponseSection = responseui.BoundaryActiveSection(last, selected.Responses)
	m.viewState.ResponseScrollOffset = 0
}

// syncActivePaneSections resets pane-local section state after a selection change.
func (m *Model) syncActivePaneSections() {
	m.syncActiveDetailsSection()
	m.resetActiveRequestSection()
	m.resetActiveResponseSection()
	m.viewState.DetailsScrollOffset = 0
	m.ensureSelectedRequestDraft()
	m.clearRequestValidation()
}

// onSelectionChanged updates pane-local state when the selected operation changes.
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

// moveDetailsSection moves the active details section by the given direction.
func (m *Model) moveDetailsSection(direction int) {
	var warnings []model.SpecWarning
	if m.session.Spec != nil {
		warnings = m.session.Spec.Warnings
	}

	m.panes.activeDetailsSection = detailsui.MoveActiveSection(
		m.panes.activeDetailsSection,
		direction,
		m.resolvedSelectedOperation(),
		m.effectiveSecurityRequirement(m.resolvedSelectedOperation()),
		warnings,
	)
	m.viewState.DetailsScrollOffset = 0
}

// setDetailsSectionBoundary moves the active details section to the first or last section.
func (m *Model) setDetailsSectionBoundary(last bool) {
	var warnings []model.SpecWarning
	if m.session.Spec != nil {
		warnings = m.session.Spec.Warnings
	}

	m.panes.activeDetailsSection = detailsui.BoundaryActiveSection(
		last,
		m.resolvedSelectedOperation(),
		m.effectiveSecurityRequirement(m.resolvedSelectedOperation()),
		warnings,
	)
	m.viewState.DetailsScrollOffset = 0
}

// scrollDetailsBy scrolls the active details section by the requested delta.
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

// scrollDetailsToBoundary scrolls the active details section to the first or last line.
func (m *Model) scrollDetailsToBoundary(last bool) {
	if last {
		m.viewState.DetailsScrollOffset = m.maxDetailsScrollOffset()
		return
	}

	m.viewState.DetailsScrollOffset = 0
}

// scrollResponseBy scrolls the active response section by the requested delta.
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

// scrollResponseToBoundary scrolls the active response section to the first or last line.
func (m *Model) scrollResponseToBoundary(last bool) {
	if last {
		m.viewState.ResponseScrollOffset = m.maxResponseScrollOffset()
		return
	}

	m.viewState.ResponseScrollOffset = 0
}

// maxResponseScrollOffset returns the largest valid response scroll offset for the active section.
func (m *Model) maxResponseScrollOffset() int {
	return responseui.MaxScrollOffset(m.projectResponsePane(), m.responseVisibleBodyLines())
}

// responseVisibleBodyLines returns the visible response body height for the current layout.
func (m *Model) responseVisibleBodyLines() int {
	width, height := m.resolvedDimensions()
	bodyHeight := max(height-m.statusBarHeight(width), 12)

	var paneHeight int
	if m.viewState.RightPaneLayoutPreset == layoutPresetWide {
		_, responseHeight := m.rightPaneHeights(computeWidePaneHeights(bodyHeight))
		paneHeight = responseHeight
	} else {
		_, responseHeight := m.rightPaneHeights(computeNarrowPaneHeights(bodyHeight))
		paneHeight = responseHeight
	}

	// reserve six lines for the pane frame, section strip, and spacing above the body content.
	return max(paneHeight-6, 1)
}

// maxDetailsScrollOffset returns the largest valid details scroll offset for the active section.
func (m *Model) maxDetailsScrollOffset() int {
	return detailsui.MaxScrollOffset(m.projectDetailsPane(), m.detailsVisibleBodyLines())
}

// detailsVisibleBodyLines returns the visible details body height for the current layout.
func (m *Model) detailsVisibleBodyLines() int {
	width, height := m.resolvedDimensions()
	bodyHeight := max(height-m.statusBarHeight(width), 12)
	var paneHeight int
	if m.viewState.RightPaneLayoutPreset == layoutPresetWide {
		paneHeight = computeWidePaneHeights(bodyHeight).Details
	} else {
		paneHeight = computeNarrowPaneHeights(bodyHeight).Details
	}

	// reserve six lines for the pane frame, section strip, and spacing above the body content.
	return max(paneHeight-6, 1)
}
