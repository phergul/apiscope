package response

import (
	"github.com/phergul/apiscope/internal/model"
	"github.com/phergul/apiscope/internal/tui/widgets"
)

// PaneInput contains the root-owned state needed to project the response pane.
type PaneInput struct {
	LoadInFlight  bool
	Selected      *model.Operation
	LastResponse  *model.HTTPResponse
	ActiveSection string
	ContentWidth  int
	ContentHeight int
	ScrollOffset  int
}

// PaneProjection contains the projected response pane plus scroll metadata.
type PaneProjection struct {
	Data            Data
	MaxScrollOffset int
}

// ProjectPane projects root response state into a render-ready response pane model.
func ProjectPane(input PaneInput) PaneProjection {
	data := Data{
		LoadInFlight: input.LoadInFlight,
		ContentWidth: input.ContentWidth,
	}
	if input.Selected == nil {
		data.EmptyState = "No operation selected.\nChoose an operation in pane 1 to inspect response details."
		return PaneProjection{Data: data}
	}

	data.Sections = append(
		[]widgets.Section{LiveSection(input.LastResponse, input.Selected, input.ContentWidth)},
		Sections(input.Selected.Responses, input.ContentWidth)...,
	)
	data.ActiveSection = ResolveActiveSection(input.ActiveSection, input.Selected.Responses)

	projected := widgets.ProjectClippedSectionView(widgets.ClippedSectionViewInput{
		Sections:      data.Sections,
		Active:        data.ActiveSection,
		EmptyState:    data.EmptyState,
		ContentWidth:  input.ContentWidth,
		ContentHeight: input.ContentHeight,
		ScrollOffset:  input.ScrollOffset,
	})
	data.Sections = projected.Data.Sections
	data.ActiveSection = projected.Active

	return PaneProjection{
		Data:            data,
		MaxScrollOffset: projected.MaxScrollOffset,
	}
}

// MaxScrollOffset returns the maximum response scroll offset for the active section.
func MaxScrollOffset(data Data, visibleLines int) int {
	return widgets.MaxSectionScrollOffset(data.Sections, data.ActiveSection, visibleLines)
}
