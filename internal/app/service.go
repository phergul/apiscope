package app

import (
	"context"
	"errors"
	"log/slog"

	"github.com/phergul/apiscope/internal/logging"
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
	logger   *slog.Logger
}

type LoadResult struct {
	Session model.SessionState
	View    model.ViewState
}

// NewService builds an app service with the default transport executor.
func NewService(loader specLoader, logger *slog.Logger) *Service {
	return NewServiceWithExecutor(loader, nil, logger)
}

// NewServiceWithExecutor builds an app service with the provided loader and executor.
func NewServiceWithExecutor(loader specLoader, executor *transport.Executor, logger *slog.Logger) *Service {
	logger = logging.OrNop(logger).With("component", "app")
	if loader == nil {
		loader = spec.NewLoader(nil, logger)
	}
	if executor == nil {
		executor = transport.NewExecutor(nil, logger)
	}

	return &Service{
		loader:   loader,
		executor: executor,
		logger:   logger,
	}
}

// LoadSource loads a spec source and returns initialized session and view state.
func (s *Service) LoadSource(ctx context.Context, rawSource string) (LoadResult, error) {
	s.logger.Info("load source started", "event", "load_source_start", "source", logging.SafeSource(rawSource))
	apiSpec, err := s.loader.Load(ctx, spec.Source{Value: rawSource})
	if err != nil {
		args := []any{"event", "load_source_failed", "source", logging.SafeSource(rawSource), "error", err.Error()}
		var specErr *spec.Error
		if errors.As(err, &specErr) {
			args = append(args, "error_kind", specErr.Kind)
			if specErr.Op != "" {
				args = append(args, "error_op", specErr.Op)
			}
		}
		s.logger.Error("load source failed", args...)
		return LoadResult{}, err
	}

	result := newLoadResult(apiSpec, rawSource)
	s.logger.Info(
		"load source completed",
		"event", "load_source_complete",
		"source", logging.SafeSource(rawSource),
		"source_family", apiSpec.SourceFamily,
		"source_version", apiSpec.SourceVersion,
		"operation_count", len(apiSpec.Operations),
		"warning_count", len(apiSpec.Warnings),
	)
	return result, nil
}

type ExecuteResult struct {
	OperationKey model.OperationKey
	ServerURL    string
	Snapshot     model.ExecutedRequestSnapshot
	Response     *model.HTTPResponse
	Validation   RequestValidationResult
}

// ExecuteCurrent prepares and executes the currently selected operation request.
func (s *Service) ExecuteCurrent(ctx context.Context, session model.SessionState) ExecuteResult {
	operation := selectedOperation(session)
	if operation == nil {
		s.logger.Error("execute current failed", "event", "execute_current_failed", "error", "no operation selected")
		return ExecuteResult{
			Response: &model.HTTPResponse{
				TransportError: "no operation selected",
			},
		}
	}

	draft := EnsureRequestDraft(&session, operation)
	snapshot := BuildExecutedRequestSnapshot(session, draft)
	s.logger.Info(
		"execute current started",
		"event", "execute_current_start",
		"operation_key", operation.Key,
		"method", operation.Method,
		"url", logging.SafeURL(session.SelectedServerURL+operation.Path),
	)
	validation := ValidateExecutableRequest(session, operation, draft)
	if validation.HasIssues() {
		s.logger.Error(
			"execute current validation failed",
			"event", "execute_validation_failed",
			"operation_key", operation.Key,
			"issue_count", len(validation.Issues),
			"error", "request validation failed",
		)
		return ExecuteResult{
			OperationKey: operation.Key,
			ServerURL:    session.SelectedServerURL,
			Snapshot:     snapshot,
			Validation:   validation,
		}
	}

	request, err := s.executor.PrepareRequest(
		operation,
		draft,
		session.SelectedServerURL,
		EffectiveSecurityRequirement(session, operation),
		session.Spec.SecuritySchemes,
		session.AuthState,
	)
	if err != nil {
		s.logger.Error(
			"request preparation failed",
			"event", "prepare_request_failed",
			"operation_key", operation.Key,
			"method", operation.Method,
			"url", logging.SafeURL(session.SelectedServerURL+operation.Path),
			"error", err.Error(),
		)
		return ExecuteResult{
			OperationKey: operation.Key,
			ServerURL:    session.SelectedServerURL,
			Snapshot:     snapshot,
			Response: &model.HTTPResponse{
				OperationKey:   operation.Key,
				TransportError: err.Error(),
			},
		}
	}

	s.logger.Info(
		"request handed to transport",
		"event", "execute_handoff",
		"operation_key", operation.Key,
		"method", operation.Method,
		"url", logging.SafeURL(request.URL.String()),
	)
	response := s.executor.Execute(ctx, operation.Key, request)
	if response.TransportError != "" {
		s.logger.Error(
			"execute current failed",
			"event", "execute_current_failed",
			"operation_key", operation.Key,
			"status_code", response.StatusCode,
			"duration_ms", response.Duration.Milliseconds(),
			"error", response.TransportError,
		)
	} else {
		s.logger.Info(
			"execute current completed",
			"event", "execute_current_complete",
			"operation_key", operation.Key,
			"status_code", response.StatusCode,
			"duration_ms", response.Duration.Milliseconds(),
		)
	}
	return ExecuteResult{
		OperationKey: operation.Key,
		ServerURL:    session.SelectedServerURL,
		Snapshot:     snapshot,
		Response:     response,
	}
}

// selectedOperation returns the currently selected operation from session state.
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
