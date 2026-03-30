package app

import (
	"context"

	"github.com/phergul/apiscope/internal/model"
	"github.com/phergul/apiscope/internal/spec"
	"github.com/phergul/apiscope/internal/transport"
)

type specLoader interface {
	Load(ctx context.Context, source spec.Source) (*model.APISpec, error)
}

type Service struct {
	loader   specLoader
	executor *transport.Executor
}

type LoadResult struct {
	Session model.SessionState
	View    model.ViewState
}

func NewService(loader specLoader) *Service {
	return NewServiceWithExecutor(loader, nil)
}

func NewServiceWithExecutor(loader specLoader, executor *transport.Executor) *Service {
	if loader == nil {
		loader = spec.NewLoader(nil)
	}
	if executor == nil {
		executor = transport.NewExecutor(nil)
	}

	return &Service{
		loader:   loader,
		executor: executor,
	}
}

func (s *Service) LoadSource(ctx context.Context, rawSource string) (LoadResult, error) {
	apiSpec, err := s.loader.Load(ctx, spec.Source{Value: rawSource})
	if err != nil {
		return LoadResult{}, err
	}

	return newLoadResult(apiSpec, rawSource), nil
}

type ExecuteResult struct {
	OperationKey model.OperationKey
	ServerURL    string
	Response     *model.HTTPResponse
	Validation   RequestValidationResult
}

func (s *Service) ExecuteCurrent(ctx context.Context, session model.SessionState) ExecuteResult {
	operation := selectedOperation(session)
	if operation == nil {
		return ExecuteResult{
			Response: &model.HTTPResponse{
				TransportError: "no operation selected",
			},
		}
	}

	draft := EnsureRequestDraft(&session, operation)
	validation := ValidateRequest(operation, draft)
	if validation.HasIssues() {
		return ExecuteResult{
			OperationKey: operation.Key,
			ServerURL:    session.SelectedServerURL,
			Validation:   validation,
		}
	}

	request, err := s.executor.PrepareRequest(operation, draft, session.SelectedServerURL)
	if err != nil {
		return ExecuteResult{
			OperationKey: operation.Key,
			ServerURL:    session.SelectedServerURL,
			Response: &model.HTTPResponse{
				OperationKey:   operation.Key,
				TransportError: err.Error(),
			},
		}
	}

	response := s.executor.Execute(ctx, operation.Key, request)
	return ExecuteResult{
		OperationKey: operation.Key,
		ServerURL:    session.SelectedServerURL,
		Response:     response,
	}
}

func selectedOperation(session model.SessionState) *model.Operation {
	if session.Spec == nil {
		return nil
	}

	for index := range session.Spec.Operations {
		if session.Spec.Operations[index].Key == session.SelectedOperationKey {
			return &session.Spec.Operations[index]
		}
	}

	return nil
}
