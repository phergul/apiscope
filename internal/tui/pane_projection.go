package tui

import (
	"github.com/phergul/apiscope/internal/model"
	detailsui "github.com/phergul/apiscope/internal/tui/details"
	operationsui "github.com/phergul/apiscope/internal/tui/operations"
	responseui "github.com/phergul/apiscope/internal/tui/response"
	statusbarui "github.com/phergul/apiscope/internal/tui/statusbar"
)

// projectOperationsPaneForState projects root state into the operations pane for a specific size and offset.
func (m *Model) projectOperationsPaneForState(contentWidth, maxLines, scrollOffset int) operationsui.PaneProjection {
	selectedKey := model.OperationKey("")
	if selected := m.resolvedSelectedOperation(); selected != nil {
		selectedKey = selected.Key
	}

	input := operationsui.PaneInput{
		LoadInFlight: m.viewState.LoadInFlight,
		LoadFailed:   m.shell.loadErr != nil,
		HasSpec:      m.session.Spec != nil,
		VisibleKeys:  append([]model.OperationKey(nil), m.viewState.VisibleOperationKeys...),
		SelectedKey:  selectedKey,
		ContentWidth: contentWidth,
		ScrollOffset: scrollOffset,
		MaxLines:     maxLines,
	}
	if m.session.Spec != nil {
		input.Operations = append([]model.Operation(nil), m.session.Spec.Operations...)
	}

	return operationsui.ProjectPane(input)
}

// projectDetailsPane projects root state into the details pane render model without viewport clipping.
func (m *Model) projectDetailsPane() detailsui.Data {
	return m.projectDetailsPaneForSize(0, 0).Data
}

// projectDetailsPaneForSize projects root state into the details pane for a specific pane size.
func (m *Model) projectDetailsPaneForSize(width, height int) detailsui.PaneProjection {
	contentWidth := 0
	contentHeight := 0
	if width > 0 {
		// subtract the pane frame padding and borders before projecting feature content.
		contentWidth = max(width-4, 1)
	}
	if height > 0 {
		// reserve space for the pane frame, section strip, and the blank spacer before the body.
		contentHeight = max(height-4, 1)
	}

	data := detailsui.PaneInput{
		LoadInFlight:  m.viewState.LoadInFlight,
		LoadErrorBody: "",
		Selected:      m.resolvedSelectedOperation(),
		FilterText:    m.viewState.FilterText,
		ActiveSection: m.panes.activeDetailsSection,
		ContentWidth:  contentWidth,
		ContentHeight: contentHeight,
		ScrollOffset:  m.viewState.DetailsScrollOffset,
	}
	if m.shell.loadErr != nil {
		data.LoadErrorBody = m.renderLoadErrorContent()
		return detailsui.ProjectPane(data)
	}
	if data.Selected != nil {
		data.Security = m.effectiveSecurityRequirement(data.Selected)
	}
	if m.session.Spec != nil {
		data.Warnings = append([]model.SpecWarning{}, m.session.Spec.Warnings...)
	}

	return detailsui.ProjectPane(data)
}

// projectResponsePaneForSize projects root state into the response pane for a specific pane size.
func (m *Model) projectResponsePaneForSize(width, height int) responseui.PaneProjection {
	contentWidth := 0
	contentHeight := 0
	if width > 0 {
		// subtract the pane frame padding and borders before projecting feature content.
		contentWidth = max(width-4, 1)
	}
	if height > 0 {
		// reserve space for the pane frame, section strip, and the blank spacer before the body.
		contentHeight = max(height-4, 1)
	}

	return responseui.ProjectPane(responseui.PaneInput{
		LoadInFlight:  m.viewState.LoadInFlight,
		Selected:      m.resolvedSelectedOperation(),
		LastResponse:  m.session.LastResponse,
		ActiveSection: m.panes.activeResponseSection,
		ContentWidth:  contentWidth,
		ContentHeight: contentHeight,
		ScrollOffset:  m.viewState.ResponseScrollOffset,
	})
}

// projectStatusBar projects root state into the status bar render model.
func (m *Model) projectStatusBar() statusbarui.Data {
	return statusbarui.Data{
		Status:   m.viewState.Notice,
		HelpHint: "Help - ?",
	}
}
