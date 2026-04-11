package app

import "github.com/phergul/apiscope/internal/model"

type CurlExportResult struct {
	OperationKey model.OperationKey
	Validation   RequestValidationResult
	Command      string
	Error        string
}

// ExportCurl validates and renders a curl command for the active request.
func (s *Service) ExportCurl(session model.SessionState) CurlExportResult {
	operation := selectedOperation(session)
	if operation == nil {
		return CurlExportResult{Error: "no operation selected"}
	}

	draft := EnsureRequestDraft(&session, operation)
	validation := ValidateExecutableRequest(session, operation, draft)
	if validation.HasIssues() {
		return CurlExportResult{
			OperationKey: operation.Key,
			Validation:   validation,
		}
	}

	command, err := s.executor.ExportCurl(
		operation,
		draft,
		session.SelectedServerURL,
		EffectiveSecurityRequirement(session, operation),
		session.Spec.SecuritySchemes,
		session.AuthState,
	)
	if err != nil {
		return CurlExportResult{
			OperationKey: operation.Key,
			Error:        err.Error(),
		}
	}

	return CurlExportResult{
		OperationKey: operation.Key,
		Command:      command,
	}
}
