package panes

import "github.com/phergul/apiscope/internal/tui/widgets"

type ResponseData struct {
	LoadInFlight  bool
	Sections      []widgets.Section
	ActiveSection string
	EmptyState    string
}

func RenderResponse(data ResponseData) string {
	if data.LoadInFlight {
		return "Loading spec..."
	}

	return widgets.RenderSectionView(widgets.SectionViewData{
		Sections:   data.Sections,
		Active:     data.ActiveSection,
		EmptyState: data.EmptyState,
	})
}
