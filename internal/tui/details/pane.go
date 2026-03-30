package details

import (
	"strings"

	"github.com/phergul/apiscope/internal/model"
	"github.com/phergul/apiscope/internal/tui/widgets"
)

// PaneInput contains the root-owned state needed to project the details pane.
type PaneInput struct {
	LoadInFlight  bool
	LoadErrorBody string
	Selected      *model.Operation
	FilterText    string
	ActiveSection string
	Security      *model.SecurityRequirement
	Warnings      []model.SpecWarning
	ContentWidth  int
	ContentHeight int
	ScrollOffset  int
}

// PaneProjection contains the projected details pane plus scroll metadata.
type PaneProjection struct {
	Data            Data
	MaxScrollOffset int
}

// ProjectPane projects root details state into a render-ready details pane model.
func ProjectPane(input PaneInput) PaneProjection {
	data := Data{
		LoadInFlight:  input.LoadInFlight,
		LoadErrorBody: input.LoadErrorBody,
		Selected:      input.Selected,
		FilterText:    input.FilterText,
		ActiveSection: ResolveActiveSection(input.ActiveSection, input.Selected, input.Security, input.Warnings),
		Security:      input.Security,
		Warnings:      append([]model.SpecWarning(nil), input.Warnings...),
	}
	if data.LoadInFlight || strings.TrimSpace(data.LoadErrorBody) != "" || data.Selected == nil {
		return PaneProjection{Data: data}
	}

	projected := widgets.ProjectClippedSectionView(widgets.ClippedSectionViewInput{
		Sections:      Sections(data),
		Active:        data.ActiveSection,
		EmptyState:    "",
		ContentWidth:  input.ContentWidth,
		ContentHeight: input.ContentHeight,
		ScrollOffset:  input.ScrollOffset,
	})
	data.ActiveSection = projected.Active
	data.Sections = projected.Data.Sections

	return PaneProjection{
		Data:            data,
		MaxScrollOffset: projected.MaxScrollOffset,
	}
}

// MaxScrollOffset returns the maximum details scroll offset for the active section.
func MaxScrollOffset(data Data, visibleLines int) int {
	return widgets.MaxSectionScrollOffset(dataSections(data), data.ActiveSection, visibleLines)
}
