package app

import (
	"context"

	"github.com/phergul/apiscope/internal/model"
	"github.com/phergul/apiscope/internal/spec"
)

type specLoader interface {
	Load(ctx context.Context, source spec.Source) (*model.APISpec, error)
}

type Service struct {
	loader specLoader
}

type LoadResult struct {
	Session model.SessionState
	View    model.ViewState
}

func NewService(loader specLoader) *Service {
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

	return newLoadResult(apiSpec, rawSource), nil
}
