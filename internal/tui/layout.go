package tui

import (
	"fmt"
	"slices"
	"strings"

	"api-tui/internal/app"
	"api-tui/internal/model"

	"github.com/charmbracelet/lipgloss"
)

var paneBorder = lipgloss.NormalBorder()

func chooseLayoutPreset(width int) string {
	if width >= 100 {
		return layoutPresetWide
	}

	return layoutPresetNarrow
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
	view := app.DescribeLoadError(m.loadErr, m.source)
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

	detailsHeight := clampInt(height/4, 7, 10)
	responseHeight := clampInt(height/5, 5, 7)
	requestHeight := maxInt(height-detailsHeight-responseHeight, 8)

	operationsPane := m.renderPane(
		"1 Operations",
		m.operationsPaneContent(),
		leftWidth,
		height,
		m.viewState.FocusedPane == model.FocusedPaneOperations,
	)
	detailsPane := m.renderPane(
		"2 Details",
		m.detailsPaneContent(),
		rightWidth,
		detailsHeight,
		m.viewState.FocusedPane == model.FocusedPaneDetails,
	)
	requestPane := m.renderPane(
		"3 Request",
		m.requestPaneContent(),
		rightWidth,
		requestHeight,
		m.viewState.FocusedPane == model.FocusedPaneRequest,
	)
	responsePane := m.renderPane(
		"4 Response",
		m.responsePaneContent(),
		rightWidth,
		responseHeight,
		m.viewState.FocusedPane == model.FocusedPaneResponse,
	)

	rightColumn := lipgloss.JoinVertical(lipgloss.Left, detailsPane, requestPane, responsePane)

	return lipgloss.JoinHorizontal(lipgloss.Top, operationsPane, rightColumn)
}

func (m *Model) renderNarrowLayout(width, height int) string {
	operationsHeight := clampInt(height/3, 7, 10)
	detailsHeight := clampInt(height/5, 7, 10)
	responseHeight := clampInt(height/6, 5, 7)
	requestHeight := maxInt(height-operationsHeight-detailsHeight-responseHeight, 8)

	operationsPane := m.renderPane(
		"1 Operations",
		m.operationsPaneContent(),
		width,
		operationsHeight,
		m.viewState.FocusedPane == model.FocusedPaneOperations,
	)
	detailsPane := m.renderPane(
		"2 Details",
		m.detailsPaneContent(),
		width,
		detailsHeight,
		m.viewState.FocusedPane == model.FocusedPaneDetails,
	)
	requestPane := m.renderPane(
		"3 Request",
		m.requestPaneContent(),
		width,
		requestHeight,
		m.viewState.FocusedPane == model.FocusedPaneRequest,
	)
	responsePane := m.renderPane(
		"4 Response",
		m.responsePaneContent(),
		width,
		responseHeight,
		m.viewState.FocusedPane == model.FocusedPaneResponse,
	)

	return lipgloss.JoinVertical(lipgloss.Left, operationsPane, detailsPane, requestPane, responsePane)
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

func (m *Model) operationsPaneContent() string {
	switch {
	case m.viewState.LoadInFlight:
		return "Loading spec..."
	case m.loadErr != nil:
		return "Spec load failed.\nSee pane 2 for details and recovery steps."
	case m.session.Spec == nil:
		return "No spec loaded."
	}

	filterValue := fallbackText(m.viewState.FilterText, "None")
	if m.viewState.ActiveEditorMode == model.EditorModeFilter {
		filterValue += " (editing)"
	}

	lines := []string{
		fmt.Sprintf("Filter: %s", filterValue),
		"",
	}

	if len(m.session.Spec.Operations) == 0 {
		lines = append(lines, "This spec loaded successfully, but it does not define any operations.")
		return strings.Join(lines, "\n")
	}
	if len(m.viewState.VisibleOperationKeys) == 0 {
		lines = append(lines, "No operations match the current filter.", "Press Esc to clear the filter.")
		return strings.Join(lines, "\n")
	}

	selected := m.resolvedSelectedOperation()
	for _, group := range m.groupedVisibleOperations() {
		lines = append(lines, strings.ToUpper(group.Name))
		for _, key := range group.Keys {
			operation := m.operationByKey(key)
			if operation == nil {
				continue
			}

			prefix := "  "
			if selected != nil && operation.Key == selected.Key {
				prefix = "> "
			}

			line := fmt.Sprintf("%s%-6s %s", prefix, strings.ToUpper(operation.Method), operation.Path)
			lines = append(lines, line)
		}
		lines = append(lines, "")
	}
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	return strings.Join(lines, "\n")
}

func (m *Model) detailsPaneContent() string {
	switch {
	case m.viewState.LoadInFlight:
		return "Loading spec..."
	case m.loadErr != nil:
		return m.renderLoadErrorContent()
	}

	selected := m.resolvedSelectedOperation()
	if selected == nil {
		lines := []string{
			"No operation selected.",
			"Choose an operation in pane 1 to inspect its details.",
		}
		if strings.TrimSpace(m.viewState.FilterText) != "" {
			lines = append(lines, "If the list is empty, press Esc to clear the filter.")
		}
		return strings.Join(lines, "\n")
	}

	return strings.Join([]string{
		m.detailsSectionStrip(),
		"",
		m.activeDetailsSectionContent(selected),
	}, "\n")
}

func (m *Model) activeDetailsSectionContent(selected *model.Operation) string {
	switch m.activeDetailsSection {
	case detailsSectionSecurity:
		return formatSecurityRequirement(m.effectiveSecurityRequirement(selected))
	case detailsSectionWarnings:
		return formatWarnings(m.session.Spec.Warnings)
	default:
		return strings.Join([]string{
			fmt.Sprintf("Operation: %s %s", strings.ToUpper(selected.Method), selected.Path),
			fmt.Sprintf("Summary: %s", fallbackText(selected.Summary, "None")),
			fmt.Sprintf("Description: %s", fallbackText(selected.Description, "None")),
			fmt.Sprintf("Tags: %s", formatTags(selected.Tags)),
			fmt.Sprintf("Deprecated: %s", yesNo(selected.Deprecated)),
		}, "\n")
	}
}

func (m *Model) requestPaneContent() string {
	if m.viewState.LoadInFlight {
		return "Loading spec..."
	}

	return "Request editing arrives in M3.\nThis pane will hold path/query/header params, auth, and request body input."
}

func (m *Model) responsePaneContent() string {
	if m.viewState.LoadInFlight {
		return "Loading spec..."
	}

	return "Response inspection arrives in M3.\nThis pane will hold response details and examples after execution."
}

func (m *Model) renderStatusBar(width int) string {
	parts := []string{
		fmt.Sprintf("Source: %s", m.source),
		fmt.Sprintf("State: %s", m.loadStateLabel()),
		fmt.Sprintf("Focus: %s", focusedPaneLabel(m.viewState.FocusedPane)),
	}
	if selected := m.resolvedSelectedOperation(); selected != nil {
		parts = append(parts, fmt.Sprintf("Operation: %s %s", strings.ToUpper(selected.Method), selected.Path))
	}
	if m.session.Spec != nil {
		parts = append(parts, fmt.Sprintf("Count: %d", len(m.session.Spec.Operations)))
		parts = append(parts, fmt.Sprintf("Visible: %d", len(m.viewState.VisibleOperationKeys)))
		if len(m.session.Spec.Warnings) > 0 {
			parts = append(parts, fmt.Sprintf("Warnings: %d", len(m.session.Spec.Warnings)))
		}
	}
	if strings.TrimSpace(m.viewState.FilterText) != "" {
		parts = append(parts, fmt.Sprintf("Filter: %s", m.viewState.FilterText))
	}
	parts = append(parts, "Keys: 1-4 switch Tab cycle q quit")

	line := strings.Join(parts, " | ")

	return lipgloss.NewStyle().
		BorderTop(true).
		BorderStyle(paneBorder).
		Width(width).
		Render(line)
}

func (m *Model) loadStateLabel() string {
	switch {
	case m.viewState.LoadInFlight:
		return "loading"
	case m.loadErr != nil:
		return "load failed"
	case m.session.Spec != nil:
		return "loaded"
	default:
		return "idle"
	}
}

func (m *Model) operationByKey(key model.OperationKey) *model.Operation {
	if m.session.Spec == nil {
		return nil
	}

	for index := range m.session.Spec.Operations {
		operation := &m.session.Spec.Operations[index]
		if operation.Key == key {
			return operation
		}
	}

	return nil
}

func (m *Model) resolvedSelectedOperation() *model.Operation {
	if operation := m.operationByKey(m.session.SelectedOperationKey); operation != nil {
		if len(m.viewState.VisibleOperationKeys) == 0 || slices.Contains(m.viewState.VisibleOperationKeys, operation.Key) {
			return operation
		}
	}
	if len(m.viewState.VisibleOperationKeys) == 0 {
		return nil
	}

	return m.operationByKey(m.viewState.VisibleOperationKeys[0])
}

func (m *Model) effectiveSecurityRequirement(operation *model.Operation) *model.SecurityRequirement {
	if operation != nil && operation.Security != nil {
		return operation.Security
	}

	if m.session.Spec == nil {
		return nil
	}

	return m.session.Spec.Security
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

func focusedPaneLabel(pane model.FocusedPane) string {
	switch pane {
	case model.FocusedPaneDetails:
		return "details"
	case model.FocusedPaneRequest:
		return "request"
	case model.FocusedPaneResponse:
		return "response"
	default:
		return "operations"
	}
}

func summarizeError(value string, limit int) string {
	value = strings.TrimSpace(value)
	if len(value) <= limit {
		return value
	}

	return value[:limit-3] + "..."
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

func fallbackText(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}

	return value
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}

	return "no"
}

func formatTags(tags []string) string {
	if len(tags) == 0 {
		return "None"
	}

	return strings.Join(tags, ", ")
}

func formatParameterSections(parameters []model.Parameter) string {
	locations := []model.ParameterLocation{
		model.ParameterLocationPath,
		model.ParameterLocationQuery,
		model.ParameterLocationHeader,
		model.ParameterLocationCookie,
	}

	lines := make([]string, 0, len(locations)*2)
	for _, location := range locations {
		lines = append(lines, strings.ToUpper(string(location))+":")

		count := 0
		for _, parameter := range parameters {
			if parameter.In != location {
				continue
			}

			count++
			lines = append(lines, fmt.Sprintf(
				"- %s (%s, %s)",
				parameter.Name,
				requiredLabel(parameter.Required),
				formatParameterTypeHint(parameter),
			))
		}
		if count == 0 {
			lines = append(lines, "- none")
		}
	}

	return strings.Join(lines, "\n")
}

func requiredLabel(required bool) string {
	if required {
		return "required"
	}

	return "optional"
}

func formatParameterTypeHint(parameter model.Parameter) string {
	if parameter.Schema != nil {
		return formatSchemaType(parameter.Schema)
	}
	if len(parameter.Content) > 0 {
		return "content"
	}

	return "unknown"
}

func formatSchemaType(schema *model.Schema) string {
	if schema == nil {
		return "unknown"
	}

	parts := make([]string, 0, 2)
	if schema.Type != "" {
		parts = append(parts, schema.Type)
	}
	if schema.Format != "" {
		parts = append(parts, schema.Format)
	}
	if len(parts) > 0 {
		return strings.Join(parts, "/")
	}
	if schema.Ref != "" {
		return schema.Ref
	}
	if len(schema.OneOf) > 0 {
		return "oneOf"
	}
	if len(schema.AnyOf) > 0 {
		return "anyOf"
	}
	if len(schema.AllOf) > 0 {
		return "allOf"
	}

	return "object"
}

func formatRequestBody(body *model.RequestBodySpec) string {
	if body == nil {
		return "None"
	}

	required := "optional"
	if body.Required {
		required = "required"
	}

	mediaTypes := make([]string, 0, len(body.Content))
	for _, content := range body.Content {
		mediaTypes = append(mediaTypes, content.MediaType)
	}
	if len(mediaTypes) == 0 {
		mediaTypes = append(mediaTypes, "none")
	}

	lines := []string{
		fmt.Sprintf("Required: %s", required),
		fmt.Sprintf("Media types: %s", strings.Join(mediaTypes, ", ")),
	}
	if description := strings.TrimSpace(body.Description); description != "" {
		lines = append(lines, fmt.Sprintf("Description: %s", description))
	}

	return strings.Join(lines, "\n")
}

func formatResponses(responses []model.ResponseSpec) string {
	if len(responses) == 0 {
		return "None"
	}

	lines := make([]string, 0, len(responses))
	for _, response := range responses {
		mediaTypes := make([]string, 0, len(response.Content))
		for _, content := range response.Content {
			mediaTypes = append(mediaTypes, content.MediaType)
		}
		if len(mediaTypes) == 0 {
			mediaTypes = append(mediaTypes, "none")
		}

		description := fallbackText(response.Description, "None")
		lines = append(lines, fmt.Sprintf("- %s: %s [%s]", response.StatusCode, description, strings.Join(mediaTypes, ", ")))
	}

	return strings.Join(lines, "\n")
}

func formatSecurityRequirement(requirement *model.SecurityRequirement) string {
	if requirement == nil || len(requirement.Alternatives) == 0 {
		return "None"
	}

	lines := make([]string, 0, len(requirement.Alternatives))
	for _, alternative := range requirement.Alternatives {
		parts := make([]string, 0, len(alternative.Schemes))
		for _, scheme := range alternative.Schemes {
			part := scheme.Name
			if len(scheme.Scopes) > 0 {
				part += " (" + strings.Join(scheme.Scopes, ", ") + ")"
			}
			parts = append(parts, part)
		}
		if len(parts) == 0 {
			continue
		}
		lines = append(lines, "- "+strings.Join(parts, " AND "))
	}
	if len(lines) == 0 {
		return "None"
	}

	return strings.Join(lines, "\nOR\n")
}

func (m *Model) renderLoadErrorContent() string {
	view := app.DescribeLoadError(m.loadErr, m.source)
	lines := []string{
		view.Title,
		"",
		fmt.Sprintf("Category: %s", view.Category),
		fmt.Sprintf("Source: %s", fallbackText(view.Source, m.source)),
		"",
		view.Summary,
		"",
		fmt.Sprintf("Try this: %s", view.Hint),
	}

	return strings.Join(lines, "\n")
}

func (m *Model) hasBlockingLoadError() bool {
	return m.loadErr != nil
}

func formatWarnings(warnings []model.SpecWarning) string {
	if len(warnings) == 0 {
		return "No warnings."
	}

	lines := make([]string, 0, len(warnings)*3)
	for _, warning := range warnings {
		lines = append(lines, fmt.Sprintf("- %s: %s", warning.Code, warning.Message))
		if strings.TrimSpace(warning.Path) != "" {
			lines = append(lines, fmt.Sprintf("  path: %s", warning.Path))
		}
	}

	return strings.Join(lines, "\n")
}
