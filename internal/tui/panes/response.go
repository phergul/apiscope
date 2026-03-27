package panes

type ResponseData struct {
	LoadInFlight  bool
	Sections      []Section
	ActiveSection string
	EmptyState    string
}

func RenderResponse(data ResponseData) string {
	if data.LoadInFlight {
		return "Loading spec..."
	}

	return RenderSectionView(data.Sections, data.ActiveSection, data.EmptyState)
}
