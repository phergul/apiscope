package app

import "github.com/phergul/apiscope/internal/model"

// newLoadResult builds the initial app result after a successful spec load.
func newLoadResult(apiSpec *model.APISpec, rawSource string) LoadResult {
	result := LoadResult{
		Session: initialSessionState(apiSpec, rawSource),
		View:    initialViewState(apiSpec),
	}

	if len(result.View.VisibleOperationKeys) > 0 {
		result.Session.SelectedOperationKey = result.View.VisibleOperationKeys[0]
		EnsureRequestDraft(&result.Session, &apiSpec.Operations[0])
	}

	return result
}

// initialSessionState builds the initial session state for a loaded spec.
func initialSessionState(apiSpec *model.APISpec, rawSource string) model.SessionState {
	session := model.SessionState{
		SpecSource:      rawSource,
		SpecFingerprint: apiSpec.Fingerprint,
		Spec:            apiSpec,
		RequestDrafts:   make(map[model.DraftKey]*model.RequestDraft),
		AuthState:       make(map[string]model.AuthValue),
	}
	if len(apiSpec.Servers) > 0 {
		session.SelectedServerURL = apiSpec.Servers[0].URL
	}

	return session
}

// initialViewState builds the initial view state for a loaded spec.
func initialViewState(apiSpec *model.APISpec) model.ViewState {
	view := model.ViewState{
		FocusedPane:           model.FocusedPaneOperations,
		ExpandedRightPane:     model.FocusedPaneRequest,
		VisibleOperationKeys:  make([]model.OperationKey, 0, len(apiSpec.Operations)),
		ActiveEditorMode:      model.EditorModeBrowse,
		OperationsPaneVisible: true,
		ZoomedPane:            false,
	}

	for _, operation := range apiSpec.Operations {
		view.VisibleOperationKeys = append(view.VisibleOperationKeys, operation.Key)
	}

	return view
}
