package tui

import (
	"strings"
	"unicode/utf8"

	"api-tui/internal/model"
)

type detailsSection string

const (
	detailsSectionSummary     detailsSection = "Summary"
	detailsSectionParameters  detailsSection = "Parameters"
	detailsSectionRequestBody detailsSection = "Request Body"
	detailsSectionResponses   detailsSection = "Responses"
	detailsSectionSecurity    detailsSection = "Security"
)

type operationGroup struct {
	Name string
	Keys []model.OperationKey
}

func (m *Model) syncVisibleOperations() {
	if m.session.Spec == nil {
		m.viewState.VisibleOperationKeys = nil
		m.viewState.OperationsCursor = 0
		m.session.SelectedOperationKey = ""
		return
	}

	filter := strings.TrimSpace(strings.ToLower(m.viewState.FilterText))
	visible := make([]model.OperationKey, 0, len(m.session.Spec.Operations))
	for _, operation := range m.session.Spec.Operations {
		if filter == "" || operationMatchesFilter(operation, filter) {
			visible = append(visible, operation.Key)
		}
	}

	m.viewState.VisibleOperationKeys = visible
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
	if len(m.viewState.VisibleOperationKeys) == 0 {
		m.session.SelectedOperationKey = ""
		m.viewState.OperationsCursor = 0
		m.syncActiveDetailsSection()
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
	m.syncActiveDetailsSection()
}

func (m *Model) setSelectedOperationByVisibleIndex(index int) {
	if len(m.viewState.VisibleOperationKeys) == 0 {
		m.session.SelectedOperationKey = ""
		m.viewState.OperationsCursor = 0
		m.syncActiveDetailsSection()
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
	m.syncActiveDetailsSection()
}

func (m *Model) groupedVisibleOperations() []operationGroup {
	if len(m.viewState.VisibleOperationKeys) == 0 {
		return nil
	}

	groups := make([]operationGroup, 0)
	indexByName := make(map[string]int)
	for _, key := range m.viewState.VisibleOperationKeys {
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
	if len(selected.Parameters) > 0 {
		sections = append(sections, detailsSectionParameters)
	}
	if selected.RequestBody != nil {
		sections = append(sections, detailsSectionRequestBody)
	}
	if len(selected.Responses) > 0 {
		sections = append(sections, detailsSectionResponses)
	}
	requirement := m.effectiveSecurityRequirement(selected)
	if requirement != nil && len(requirement.Alternatives) > 0 {
		sections = append(sections, detailsSectionSecurity)
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
}

func (m *Model) setDetailsSectionBoundary(last bool) {
	available := m.availableDetailsSections()
	if len(available) == 0 {
		m.activeDetailsSection = detailsSectionSummary
		return
	}

	if last {
		m.activeDetailsSection = available[len(available)-1]
		return
	}

	m.activeDetailsSection = available[0]
}

func (m *Model) detailsSectionStrip() string {
	available := m.availableDetailsSections()
	parts := make([]string, 0, len(available))
	for _, section := range available {
		label := string(section)
		if section == m.activeDetailsSection {
			label = "[" + label + "]"
		}
		parts = append(parts, label)
	}

	return strings.Join(parts, "  ")
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
