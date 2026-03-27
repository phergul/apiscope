package panes

type RequestData struct {
	LoadInFlight bool
}

func RenderRequest(data RequestData) string {
	if data.LoadInFlight {
		return "Loading spec..."
	}

	return "Request editing arrives later.\nThis pane will hold path/query/header params, auth, and request body input."
}
