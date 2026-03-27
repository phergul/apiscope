package app

import (
	"context"

	"api-tui/internal/model"
	"api-tui/internal/spec"
)

type Service struct {
	loader spec.Loader
}

type LoadResult struct {
	Session model.SessionState
	View    model.ViewState
}

func NewService(loader spec.Loader) *Service {
	if loader == nil {
		loader = spec.NewLoader(nil)
	}

	return &Service{loader: loader}
}

func (s *Service) LoadSource(ctx context.Context, rawSource string) (LoadResult, error) {
	apiSpec, err := s.loader.Load(ctx, spec.Source{Value: rawSource})
	if err != nil {
		return LoadResult{}, err
	}

	result := LoadResult{
		Session: model.SessionState{
			SpecSource:      rawSource,
			SpecFingerprint: apiSpec.Fingerprint,
			Spec:            apiSpec,
			RequestDrafts:   make(map[model.DraftKey]*model.RequestDraft),
			AuthState:       make(map[string]model.AuthValue),
		},
		View: model.ViewState{
			FocusedPane:           model.FocusedPaneOperations,
			VisibleOperationKeys:  make([]model.OperationKey, 0, len(apiSpec.Operations)),
			ActiveEditorMode:      model.EditorModeBrowse,
			OperationsPaneVisible: true,
			ResponsePaneExpanded:  false,
		},
	}

	if len(apiSpec.Servers) > 0 {
		result.Session.SelectedServerURL = apiSpec.Servers[0].URL
	}

	for _, operation := range apiSpec.Operations {
		result.View.VisibleOperationKeys = append(result.View.VisibleOperationKeys, operation.Key)
	}
	if len(result.View.VisibleOperationKeys) > 0 {
		result.Session.SelectedOperationKey = result.View.VisibleOperationKeys[0]
	}

	return result, nil
}
