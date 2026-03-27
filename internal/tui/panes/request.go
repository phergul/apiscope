package panes

type RequestData struct {
	LoadInFlight  bool
	Sections      []Section
	ActiveSection string
	EmptyState    string
}

func RenderRequest(data RequestData) string {
	if data.LoadInFlight {
		return "Loading spec..."
	}

	return RenderSectionView(data.Sections, data.ActiveSection, data.EmptyState)
}
