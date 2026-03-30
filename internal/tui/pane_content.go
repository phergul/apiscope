package tui

import (
	"strings"

	"github.com/phergul/apiscope/internal/model"
	detailsui "github.com/phergul/apiscope/internal/tui/details"
	operationsui "github.com/phergul/apiscope/internal/tui/operations"
	requestui "github.com/phergul/apiscope/internal/tui/request"
	"github.com/phergul/apiscope/internal/tui/widgets"
	"github.com/phergul/apiscope/internal/util"

	"github.com/charmbracelet/lipgloss"
)

type paneView struct {
	TitleLeft  string
	Title      string
	TitleRight string
	Body       string
	Footer     string
	Focused    bool
}

func (m *Model) operationsPaneContent() string {
	width, _ := m.resolvedDimensions()
	if m.viewState.RightPaneLayoutPreset == layoutPresetWide {
		totalWidth := width
		width = util.Clamp(int(float64(totalWidth)*0.32), 30, 40)
		width = min(width, totalWidth-20)
	}

	return m.operationsPaneContentForSize(width)
}

func (m *Model) operationsPaneContentForSize(width int) string {
	data := m.projectOperationsPane()
	data.ContentWidth = max(width-4, 1)
	data.ScrollOffset = m.viewState.OperationsScrollOffset
	return operationsui.Render(data)
}

func (m *Model) operationsPaneContentForHeight(height int) string {
	width, _ := m.resolvedDimensions()
	if m.viewState.RightPaneLayoutPreset == layoutPresetWide {
		totalWidth := width
		width = util.Clamp(int(float64(totalWidth)*0.32), 30, 40)
		width = min(width, totalWidth-20)
	}

	return m.operationsPaneContentForSizeAndHeight(width, height)
}

func (m *Model) operationsPaneContentForSizeAndHeight(width, height int) string {
	data := m.projectOperationsPane()
	data.ContentWidth = max(width-4, 1)
	data.ScrollOffset = m.viewState.OperationsScrollOffset
	data.MaxLines = max(height-4, 1)
	return operationsui.Render(data)
}

func (m *Model) operationsPaneMetrics() (int, int) {
	width, height := m.resolvedDimensions()
	bodyHeight := max(height-lipgloss.Height(m.renderStatusBar(width)), 12)
	paneWidth := width
	paneHeight := bodyHeight

	if !(m.viewState.ZoomedPane && m.viewState.FocusedPane == model.FocusedPaneOperations) {
		if m.viewState.RightPaneLayoutPreset == layoutPresetWide {
			paneWidth = util.Clamp(int(float64(width)*0.32), 30, 40)
			paneWidth = min(paneWidth, width-20)
		} else {
			paneHeight = computeNarrowPaneHeights(bodyHeight).Operations
		}
	}

	return max(paneWidth-4, 1), max(paneHeight-4, 1)
}

func (m *Model) visibleOperationRowCount(offset int) int {
	contentWidth, maxLines := m.operationsPaneMetrics()
	data := m.projectOperationsPane()
	data.ContentWidth = contentWidth
	data.ScrollOffset = offset
	data.MaxLines = maxLines
	return operationsui.VisibleRowCount(data)
}

func (m *Model) detailsPaneContent() string {
	return detailsui.Render(m.projectDetailsPane())
}

func (m *Model) detailsPaneContentForHeight(height int) string {
	width, _ := m.resolvedDimensions()
	if m.viewState.RightPaneLayoutPreset == layoutPresetWide {
		leftWidth := util.Clamp(int(float64(width)*0.32), 30, 40)
		leftWidth = min(leftWidth, width-20)
		width = max(width-leftWidth, 20)
	}

	return m.detailsPaneContentForSize(width, height)
}

func (m *Model) detailsPaneContentForSize(width, height int) string {
	data := m.projectDetailsPane()
	if data.LoadInFlight || strings.TrimSpace(data.LoadErrorBody) != "" || data.Selected == nil {
		return detailsui.Render(data)
	}

	visibleLines := max(height-6, 1)
	contentWidth := max(width-4, 1)
	viewport := widgets.NewViewport(contentWidth, visibleLines)
	viewport.SetContent(detailsui.RenderActiveSection(data))
	viewport.SetYOffset(m.viewState.DetailsScrollOffset)
	clipped := viewport.View()
	sections := detailsui.Sections(data)
	for index := range sections {
		if sections[index].Label == data.ActiveSection {
			sections[index].Body = clipped
			return widgets.RenderSectionView(widgets.SectionViewData{
				Sections:   sections,
				Active:     data.ActiveSection,
				EmptyState: "",
			})
		}
	}

	if len(sections) > 0 {
		sections[0].Body = clipped
	}

	return widgets.RenderSectionView(widgets.SectionViewData{
		Sections:   sections,
		Active:     data.ActiveSection,
		EmptyState: "",
	})
}

func (m *Model) requestPaneContent() string {
	return requestui.Render(m.projectRequestPane())
}

func (m *Model) responsePaneContent() string {
	width, height := m.resolvedDimensions()
	if m.viewState.RightPaneLayoutPreset == layoutPresetWide {
		leftWidth := util.Clamp(int(float64(width)*0.32), 30, 40)
		leftWidth = min(leftWidth, width-20)
		width = max(width-leftWidth, 20)
	}

	return m.responsePaneContentForSize(width, height)
}

func (m *Model) paneView(pane model.FocusedPane) paneView {
	switch pane {
	case model.FocusedPaneDetails:
		return paneView{
			Title:   "2 Details",
			Body:    m.detailsPaneContent(),
			Focused: m.viewState.FocusedPane == pane,
		}
	case model.FocusedPaneRequest:
		return paneView{
			Title:      "3 Request",
			TitleRight: m.requestPaneTitleRight(),
			Body:       m.requestPaneContent(),
			Focused:    m.viewState.FocusedPane == pane && !m.requestEditActive(),
		}
	case model.FocusedPaneResponse:
		return paneView{
			Title:   "4 Response",
			Body:    m.responsePaneContent(),
			Focused: m.viewState.FocusedPane == pane,
		}
	default:
		return paneView{
			Title:   "1 Operations",
			Body:    m.operationsPaneContent(),
			Footer:  m.operationsPaneFooter(),
			Focused: m.viewState.FocusedPane == pane,
		}
	}
}

func (m *Model) operationsPaneFooter() string {
	m.ensureWidgetDefaults()

	if m.viewState.ActiveEditorMode != model.EditorModeFilter && strings.TrimSpace(m.viewState.FilterText) == "" {
		return ""
	}

	if m.viewState.ActiveEditorMode == model.EditorModeFilter {
		return m.filterInput.View()
	}

	return widgets.InputFrameStyle(false).
		Render("Filter: " + m.viewState.FilterText)
}

func (m *Model) requestPaneTitleRight() string {
	if m.viewState.FocusedPane != model.FocusedPaneRequest {
		return ""
	}

	return "Send request Ctrl+R"
}
