package tui

import (
	"github.com/phergul/apiscope/internal/model"
	detailsui "github.com/phergul/apiscope/internal/tui/details"
	operationsui "github.com/phergul/apiscope/internal/tui/operations"
	requestui "github.com/phergul/apiscope/internal/tui/request"
	responseui "github.com/phergul/apiscope/internal/tui/response"
	schemaexplorerui "github.com/phergul/apiscope/internal/tui/schemaexplorer"
)

type paneView struct {
	TitleLeft  string
	Title      string
	TitleRight string
	Body       string
	Footer     string
	Focused    bool
}

// operationsPaneContent renders the operations pane using the current window-derived width.
func (m *Model) operationsPaneContent() string {
	width, _ := m.resolvedDimensions()
	if m.viewState.RightPaneLayoutPreset == layoutPresetWide {
		width, _ = m.wideColumnWidths(width)
	}

	return m.operationsPaneContentForSize(width)
}

// operationsPaneContentForSize renders the operations pane for the provided pane width.
func (m *Model) operationsPaneContentForSize(width int) string {
	// subtract the pane frame padding and borders before passing width to the renderer.
	return operationsui.Render(m.projectOperationsPaneForState(max(width-4, 1), 0, m.viewState.OperationsScrollOffset).Data)
}

// operationsPaneContentForHeight renders the operations pane using the current width and a fixed height.
func (m *Model) operationsPaneContentForHeight(height int) string {
	width, _ := m.resolvedDimensions()
	if m.viewState.RightPaneLayoutPreset == layoutPresetWide {
		width, _ = m.wideColumnWidths(width)
	}

	return m.operationsPaneContentForSizeAndHeight(width, height)
}

// operationsPaneContentForSizeAndHeight renders the operations pane for the provided size.
func (m *Model) operationsPaneContentForSizeAndHeight(width, height int) string {
	// subtract the pane frame padding and borders before passing width to the renderer.
	contentWidth := max(width-4, 1)
	// the operations pane only loses the top and bottom frame rows before content begins.
	maxLines := max(height-2, 1)
	return operationsui.Render(m.projectOperationsPaneForState(contentWidth, maxLines, m.viewState.OperationsScrollOffset).Data)
}

// operationsPaneMetrics returns the effective operations pane content width and height.
func (m *Model) operationsPaneMetrics() (int, int) {
	width, height := m.resolvedDimensions()
	// reserve the rendered status bar before splitting the remaining shell height across panes.
	bodyHeight := max(height-m.statusBarHeight(width), 12)
	paneWidth := width
	paneHeight := bodyHeight

	if !(m.viewState.ZoomedPane && m.viewState.FocusedPane == model.FocusedPaneOperations) {
		if m.viewState.RightPaneLayoutPreset == layoutPresetWide {
			paneWidth, _ = m.wideColumnWidths(width)
		} else {
			paneHeight = computeNarrowPaneHeights(bodyHeight).Operations
		}
	}

	// keep the existing navigation metrics so scrolloff behavior does not change with the visual fill tweak.
	return max(paneWidth-4, 1), max(paneHeight-4, 1)
}

// visibleOperationRowCount returns the number of rendered operation rows visible at the given offset.
func (m *Model) visibleOperationRowCount(offset int) int {
	contentWidth, maxLines := m.operationsPaneMetrics()
	return m.projectOperationsPaneForState(contentWidth, maxLines, offset).VisibleRows
}

// detailsPaneContent renders the unwindowed details pane for default rendering paths and tests.
func (m *Model) detailsPaneContent() string {
	return detailsui.Render(m.projectDetailsPane())
}

// detailsPaneContentForHeight renders the details pane using the current width and a fixed height.
func (m *Model) detailsPaneContentForHeight(height int) string {
	width, _ := m.resolvedDimensions()
	if m.viewState.RightPaneLayoutPreset == layoutPresetWide {
		_, width = m.wideColumnWidths(width)
	}

	return m.detailsPaneContentForSize(width, height)
}

// detailsPaneContentForSize renders the details pane for the provided size.
func (m *Model) detailsPaneContentForSize(width, height int) string {
	return detailsui.Render(m.projectDetailsPaneForSize(width, height).Data)
}

// requestPaneContent renders the request pane with its current projected state.
func (m *Model) requestPaneContent() string {
	return requestui.Render(m.projectRequestPane())
}

// responsePaneContent renders the response pane using the current window-derived size.
func (m *Model) responsePaneContent() string {
	width, height := m.resolvedDimensions()
	if m.viewState.RightPaneLayoutPreset == layoutPresetWide {
		_, width = m.wideColumnWidths(width)
	}

	return m.responsePaneContentForSize(width, height)
}

// responsePaneContentForSize renders the response pane for the provided size.
func (m *Model) responsePaneContentForSize(width, height int) string {
	return responseui.Render(m.projectResponsePaneForSize(width, height).Data)
}

// paneView returns the frame data for the requested pane.
func (m *Model) paneView(pane model.FocusedPane) paneView {
	switch pane {
	case model.FocusedPaneDetails:
		return paneView{
			Title:      "2 Details",
			TitleRight: m.detailsPaneTitleRight(),
			Body:       m.detailsPaneContent(),
			Focused:    m.paneOuterFocused(pane),
		}
	case model.FocusedPaneRequest:
		return paneView{
			Title:      "3 Request",
			TitleRight: m.requestPaneTitleRight(),
			Body:       m.requestPaneContent(),
			Focused:    m.paneOuterFocused(pane) && !m.requestEditActive(),
		}
	case model.FocusedPaneResponse:
		return paneView{
			Title:   "4 Response",
			Body:    m.responsePaneContent(),
			Focused: m.paneOuterFocused(pane),
		}
	default:
		return paneView{
			Title:   "1 Operations",
			Body:    m.operationsPaneContent(),
			Footer:  m.operationsPaneFooter(),
			Focused: m.paneOuterFocused(pane),
		}
	}
}

func (m *Model) detailsPaneTitleRight() string {
	if m.viewState.FocusedPane != model.FocusedPaneDetails {
		return ""
	}
	if schemaexplorerui.Available(m.resolvedSelectedOperation()) {
		return "Open schemas 's'"
	}

	return ""
}

// paneOuterFocused reports whether a shell pane should render its focused border state.
func (m *Model) paneOuterFocused(pane model.FocusedPane) bool {
	// shell-level popups take the only visible focus ring while they are open.
	if m.historyPopupOpen() || m.helpOverlayOpen() {
		return false
	}

	return m.viewState.FocusedPane == pane
}

// operationsPaneFooter renders the filter footer shown below the operations pane when needed.
func (m *Model) operationsPaneFooter() string {
	m.ensureWidgetDefaults()
	m.widgets.filterInput.SetWidth(m.operationsFooterWidth())
	return operationsui.RenderFooter(operationsui.FilterFooterInput{
		Editing:    m.viewState.ActiveEditorMode == model.EditorModeFilter,
		FilterText: m.viewState.FilterText,
		EditorView: m.widgets.filterInput.BareFilledView(),
		Width:      m.operationsFooterWidth(),
	})
}

// operationsFooterWidth returns the current full footer width for the operations pane content area.
func (m *Model) operationsFooterWidth() int {
	width, _ := m.resolvedDimensions()
	if m.viewState.RightPaneLayoutPreset == layoutPresetWide {
		width, _ = m.wideColumnWidths(width)
	}

	return max(width-4, 1)
}

// requestPaneTitleRight returns the contextual title-right hint for the request pane.
func (m *Model) requestPaneTitleRight() string {
	if m.viewState.FocusedPane != model.FocusedPaneRequest {
		return ""
	}

	return "Send request Ctrl+R"
}
