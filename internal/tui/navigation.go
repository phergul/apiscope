package tui

import (
	"strings"
	"unicode/utf8"

	"github.com/phergul/apiscope/internal/model"
	"github.com/phergul/apiscope/internal/tui/panes"

	"github.com/charmbracelet/lipgloss"
)

type detailsSection string

const (
	detailsSectionSummary  detailsSection = "Summary"
	detailsSectionSecurity detailsSection = "Security"
	detailsSectionWarnings detailsSection = "Warnings"
)

type operationGroup struct {
	Name string
	Keys []model.OperationKey
}

const (
	requestSectionBody = "Body"
	requestSectionAuth = "Auth"
)

var requestParameterLocations = []model.ParameterLocation{
	model.ParameterLocationPath,
	model.ParameterLocationQuery,
	model.ParameterLocationHeader,
	model.ParameterLocationCookie,
}

func (m *Model) syncVisibleOperations() {
	if m.session.Spec == nil {
		m.viewState.VisibleOperationKeys = nil
		m.viewState.OperationsCursor = 0
		m.session.SelectedOperationKey = ""
		m.syncActivePaneSections()
		return
	}

	filter := strings.TrimSpace(strings.ToLower(m.viewState.FilterText))
	visible := make([]model.OperationKey, 0, len(m.session.Spec.Operations))
	for _, operation := range m.session.Spec.Operations {
		if filter == "" || operationMatchesFilter(operation, filter) {
			visible = append(visible, operation.Key)
		}
	}

	m.viewState.VisibleOperationKeys = m.groupOperationKeys(visible)
	m.syncSelectedOperationAfterVisibilityChange()
}

func operationMatchesFilter(operation model.Operation, filter string) bool {
	if filter == "" {
		return true
	}

	fields := []string{
		operation.Method,
		operation.Path,
		operation.Summary,
	}
	fields = append(fields, operation.Tags...)

	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), filter) {
			return true
		}
	}

	return false
}

func (m *Model) syncSelectedOperationAfterVisibilityChange() {
	previous := m.session.SelectedOperationKey
	if len(m.viewState.VisibleOperationKeys) == 0 {
		m.session.SelectedOperationKey = ""
		m.viewState.OperationsCursor = 0
		m.onSelectionChanged(previous, "")
		return
	}

	for index, key := range m.viewState.VisibleOperationKeys {
		if key == m.session.SelectedOperationKey {
			m.viewState.OperationsCursor = index
			m.syncActiveDetailsSection()
			return
		}
	}

	m.session.SelectedOperationKey = m.viewState.VisibleOperationKeys[0]
	m.viewState.OperationsCursor = 0
	m.onSelectionChanged(previous, m.session.SelectedOperationKey)
}

func (m *Model) setSelectedOperationByVisibleIndex(index int) {
	previous := m.session.SelectedOperationKey
	if len(m.viewState.VisibleOperationKeys) == 0 {
		m.session.SelectedOperationKey = ""
		m.viewState.OperationsCursor = 0
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
	m.onSelectionChanged(previous, m.session.SelectedOperationKey)
}

func (m *Model) groupedVisibleOperations() []operationGroup {
	return m.groupedOperationKeys(m.viewState.VisibleOperationKeys)
}

func (m *Model) groupedOperationKeys(keys []model.OperationKey) []operationGroup {
	if len(keys) == 0 {
		return nil
	}

	groups := make([]operationGroup, 0)
	indexByName := make(map[string]int)
	for _, key := range keys {
		operation := m.operationByKey(key)
		if operation == nil {
			continue
		}

		groupName := operationGroupName(operation)
		groupIndex, ok := indexByName[groupName]
		if !ok {
			groupIndex = len(groups)
			indexByName[groupName] = groupIndex
			groups = append(groups, operationGroup{Name: groupName})
		}

		groups[groupIndex].Keys = append(groups[groupIndex].Keys, key)
	}

	return groups
}

func (m *Model) groupOperationKeys(keys []model.OperationKey) []model.OperationKey {
	groups := m.groupedOperationKeys(keys)
	ordered := make([]model.OperationKey, 0, len(keys))
	for _, group := range groups {
		ordered = append(ordered, group.Keys...)
	}

	return ordered
}

func operationGroupName(operation *model.Operation) string {
	if operation == nil || len(operation.Tags) == 0 || strings.TrimSpace(operation.Tags[0]) == "" {
		return "Untagged"
	}

	return operation.Tags[0]
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

func (m *Model) availableDetailsSections() []detailsSection {
	selected := m.resolvedSelectedOperation()
	if selected == nil {
		return []detailsSection{detailsSectionSummary}
	}

	sections := []detailsSection{detailsSectionSummary}
	requirement := m.effectiveSecurityRequirement(selected)
	if requirement != nil && len(requirement.Alternatives) > 0 {
		sections = append(sections, detailsSectionSecurity)
	}
	if m.session.Spec != nil && len(m.session.Spec.Warnings) > 0 {
		sections = append(sections, detailsSectionWarnings)
	}

	return sections
}

func (m *Model) syncActiveDetailsSection() {
	available := m.availableDetailsSections()
	if len(available) == 0 {
		m.activeDetailsSection = detailsSectionSummary
		return
	}

	for _, section := range available {
		if section == m.activeDetailsSection {
			return
		}
	}

	m.activeDetailsSection = available[0]
}

func (m *Model) availableRequestSections() []string {
	selected := m.resolvedSelectedOperation()
	if selected == nil {
		return nil
	}

	sections := make([]string, 0, len(requestParameterLocations)+2)
	for _, location := range requestParameterLocations {
		if hasParametersInLocation(selected.Parameters, location) {
			sections = append(sections, requestLocationSectionLabel(location))
		}
	}
	if selected.RequestBody != nil {
		sections = append(sections, requestSectionBody)
	}
	requirement := m.effectiveSecurityRequirement(selected)
	if requirement != nil && len(requirement.Alternatives) > 0 {
		sections = append(sections, requestSectionAuth)
	}

	return sections
}

func (m *Model) syncActiveRequestSection() {
	available := m.availableRequestSections()
	if len(available) == 0 {
		m.activeRequestSection = ""
		m.resetRequestCursorAndScroll()
		return
	}

	for _, section := range available {
		if section == m.activeRequestSection {
			m.syncActiveRequestRow()
			return
		}
	}

	m.activeRequestSection = available[0]
	m.resetRequestCursorAndScroll()
}

func (m *Model) resetActiveRequestSection() {
	available := m.availableRequestSections()
	if len(available) == 0 {
		m.activeRequestSection = ""
		m.resetRequestCursorAndScroll()
		return
	}

	m.activeRequestSection = available[0]
	m.resetRequestCursorAndScroll()
}

func (m *Model) moveRequestSection(direction int) {
	m.activeRequestSection = moveStringSection(m.activeRequestSection, m.availableRequestSections(), direction)
	m.resetRequestCursorAndScroll()
}

func (m *Model) setRequestSectionBoundary(last bool) {
	m.activeRequestSection = boundaryStringSection(m.availableRequestSections(), last)
	m.resetRequestCursorAndScroll()
}

func (m *Model) availableResponseSections() []string {
	selected := m.resolvedSelectedOperation()
	if selected == nil {
		return nil
	}

	sections := make([]string, 0, len(selected.Responses))
	for _, response := range selected.Responses {
		sections = append(sections, response.StatusCode)
	}

	return sections
}

func (m *Model) syncActiveResponseSection() {
	available := m.availableResponseSections()
	if len(available) == 0 {
		m.activeResponseSection = ""
		return
	}

	for _, section := range available {
		if section == m.activeResponseSection {
			return
		}
	}

	m.activeResponseSection = available[0]
}

func (m *Model) resetActiveResponseSection() {
	available := m.availableResponseSections()
	if len(available) == 0 {
		m.activeResponseSection = ""
		return
	}

	m.activeResponseSection = available[0]
}

func (m *Model) moveResponseSection(direction int) {
	m.activeResponseSection = moveStringSection(m.activeResponseSection, m.availableResponseSections(), direction)
}

func (m *Model) setResponseSectionBoundary(last bool) {
	m.activeResponseSection = boundaryStringSection(m.availableResponseSections(), last)
}

func (m *Model) syncActivePaneSections() {
	m.syncActiveDetailsSection()
	m.resetActiveRequestSection()
	m.resetActiveResponseSection()
	m.viewState.DetailsScrollOffset = 0
	m.ensureSelectedRequestDraft()
}

func (m *Model) onSelectionChanged(previous, current model.OperationKey) {
	m.syncActiveDetailsSection()
	if previous != current {
		m.ensureSelectedRequestDraft()
		m.resetActiveRequestSection()
		m.resetActiveResponseSection()
		m.viewState.DetailsScrollOffset = 0
	}
}

func hasParametersInLocation(parameters []model.Parameter, location model.ParameterLocation) bool {
	for _, parameter := range parameters {
		if parameter.In == location {
			return true
		}
	}

	return false
}

func requestLocationSectionLabel(location model.ParameterLocation) string {
	switch location {
	case model.ParameterLocationPath:
		return "Path"
	case model.ParameterLocationQuery:
		return "Query"
	case model.ParameterLocationHeader:
		return "Header"
	case model.ParameterLocationCookie:
		return "Cookie"
	default:
		return string(location)
	}
}

func moveStringSection(current string, available []string, direction int) string {
	if len(available) == 0 {
		return ""
	}

	currentIndex := 0
	for index, section := range available {
		if section == current {
			currentIndex = index
			break
		}
	}

	targetIndex := currentIndex + direction
	if targetIndex < 0 || targetIndex >= len(available) {
		return available[currentIndex]
	}

	return available[targetIndex]
}

func boundaryStringSection(available []string, last bool) string {
	if len(available) == 0 {
		return ""
	}
	if last {
		return available[len(available)-1]
	}

	return available[0]
}

func (m *Model) moveDetailsSection(direction int) {
	available := m.availableDetailsSections()
	if len(available) == 0 {
		m.activeDetailsSection = detailsSectionSummary
		return
	}

	currentIndex := 0
	for index, section := range available {
		if section == m.activeDetailsSection {
			currentIndex = index
			break
		}
	}

	targetIndex := currentIndex + direction
	if targetIndex < 0 || targetIndex >= len(available) {
		return
	}

	m.activeDetailsSection = available[targetIndex]
	m.viewState.DetailsScrollOffset = 0
}

func (m *Model) setDetailsSectionBoundary(last bool) {
	available := m.availableDetailsSections()
	if len(available) == 0 {
		m.activeDetailsSection = detailsSectionSummary
		return
	}

	if last {
		m.activeDetailsSection = available[len(available)-1]
		m.viewState.DetailsScrollOffset = 0
		return
	}

	m.activeDetailsSection = available[0]
	m.viewState.DetailsScrollOffset = 0
}

func (m *Model) detailsSectionStrip() string {
	available := m.availableDetailsSections()
	parts := make([]string, 0, len(available))
	for _, section := range available {
		label := string(section)
		parts = append(parts, label)
	}

	return strings.Join(parts, "  ")
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

func (m *Model) maxDetailsScrollOffset() int {
	data := m.projectDetailsPane()
	lines := len(splitLines(panes.RenderActiveDetailsSectionForProjection(data)))
	visible := m.detailsVisibleBodyLines()
	if lines <= visible {
		return 0
	}

	return lines - visible
}

func (m *Model) detailsVisibleBodyLines() int {
	width, height := m.resolvedDimensions()
	bodyHeight := maxInt(height-lipgloss.Height(m.renderStatusBar(width)), 12)
	var paneHeight int
	if m.viewState.RightPaneLayoutPreset == layoutPresetWide {
		paneHeight = computeWidePaneHeights(bodyHeight).Details
	} else {
		paneHeight = computeNarrowPaneHeights(bodyHeight).Details
	}

	return maxInt(paneHeight-6, 1)
}

func appendFilterInput(existing string, runes []rune) string {
	if len(runes) == 0 {
		return existing
	}

	return existing + string(runes)
}

func trimLastRune(value string) string {
	if value == "" {
		return ""
	}

	_, size := utf8.DecodeLastRuneInString(value)
	if size <= 0 {
		return ""
	}

	return value[:len(value)-size]
}
