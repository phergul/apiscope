package tui

import (
	"fmt"
	"strings"

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
		return "Load failed.\nSee details pane for the error."
	case m.session.Spec == nil:
		return "No spec loaded."
	}

	title := m.session.Spec.Title
	if strings.TrimSpace(title) == "" {
		title = "(untitled API)"
	}

	lines := []string{
		fmt.Sprintf("API: %s", title),
		fmt.Sprintf("Operations: %d", len(m.session.Spec.Operations)),
	}
	if selected := m.selectedOperation(); selected != nil {
		lines = append(lines, fmt.Sprintf("Current: %s %s", selected.Method, selected.Path))
	} else {
		lines = append(lines, "", "No operations in spec.")
	}

	return strings.Join(lines, "\n")
}

func (m *Model) detailsPaneContent() string {
	switch {
	case m.viewState.LoadInFlight:
		return "Loading spec..."
	case m.loadErr != nil:
		return fmt.Sprintf("Failed to load spec.\n\n%s", m.loadErr.Error())
	}

	selected := m.selectedOperation()
	if selected == nil {
		return "No operations in spec."
	}

	summary := selected.Summary
	if strings.TrimSpace(summary) == "" {
		summary = "No summary yet."
	}

	return strings.Join([]string{
		fmt.Sprintf("Operation: %s %s", selected.Method, selected.Path),
		fmt.Sprintf("Summary: %s", summary),
		"Read-only explorer rendering arrives in M2.2.",
	}, "\n")
}

func (m *Model) requestPaneContent() string {
	if m.viewState.LoadInFlight {
		return "Loading spec..."
	}

	return "Request editor arrives in M3."
}

func (m *Model) responsePaneContent() string {
	if m.viewState.LoadInFlight {
		return "Loading spec..."
	}

	return "No response yet."
}

func (m *Model) renderStatusBar(width int) string {
	line := fmt.Sprintf(
		"Source: %s | State: %s | Focus: %s | Keys: 1-4 switch Tab cycle q quit",
		m.source,
		m.loadStateLabel(),
		focusedPaneLabel(m.viewState.FocusedPane),
	)

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
		return "load failed: " + summarizeError(m.loadErr.Error(), 40)
	case m.session.Spec != nil:
		return "loaded"
	default:
		return "idle"
	}
}

func (m *Model) selectedOperation() *model.Operation {
	if m.session.Spec == nil {
		return nil
	}

	for index := range m.session.Spec.Operations {
		operation := &m.session.Spec.Operations[index]
		if operation.Key == m.session.SelectedOperationKey {
			return operation
		}
	}

	return nil
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
