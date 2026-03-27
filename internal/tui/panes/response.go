package panes

type ResponseData struct {
	LoadInFlight bool
}

func RenderResponse(data ResponseData) string {
	if data.LoadInFlight {
		return "Loading spec..."
	}

	return "Response inspection arrives later.\nThis pane will hold response details and examples after execution."
}
