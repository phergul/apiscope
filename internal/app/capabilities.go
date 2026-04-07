package app

import (
	"fmt"
	"strings"

	"github.com/phergul/apiscope/internal/model"
)

// ProjectCapabilityWarnings surfaces spec-version limits through the normalised capability set.
func ProjectCapabilityWarnings(apiSpec *model.APISpec) []model.SpecWarning {
	if apiSpec == nil {
		return nil
	}

	warnings := make([]model.SpecWarning, 0, 2)
	if !apiSpec.Capabilities.SupportsCookieParameters {
		warnings = append(warnings, model.SpecWarning{
			Code:    model.SpecWarningUnsupportedFeature,
			Message: fmt.Sprintf("cookie parameters are unavailable for %s specs", capabilitySourceLabel(apiSpec)),
			Path:    "source version",
		})
	}
	if !apiSpec.Capabilities.SupportsServerVariables {
		warnings = append(warnings, model.SpecWarning{
			Code:    model.SpecWarningUnsupportedFeature,
			Message: fmt.Sprintf("server variables are unavailable for %s specs", capabilitySourceLabel(apiSpec)),
			Path:    "source version",
		})
	}

	return warnings
}

// ProjectCapabilityRequestSupportNotes reports capability-gated request-pane limitations.
func ProjectCapabilityRequestSupportNotes(apiSpec *model.APISpec, operation *model.Operation, draft *model.RequestDraft, servers []model.Server) []RequestSupportNote {
	if apiSpec == nil {
		return nil
	}

	notes := make([]RequestSupportNote, 0, 2)
	if apiSpec.Capabilities.SupportsServerVariables && hasServerVariables(servers) {
		notes = append(notes, RequestSupportNote{
			Section:  "Server",
			Severity: RequestSupportSeverityDowngraded,
			Summary:  "Server variables are not editable yet.",
			Detail:   "This OpenAPI 3.x spec defines templated server URLs. Pane 3 can show the current template, but it cannot edit server variable values yet.",
		})
	}
	if apiSpec.Capabilities.SupportsOpenAPI3 && currentBodyMediaTypeHasEncodingWarning(apiSpec, operation, draft) {
		notes = append(notes, RequestSupportNote{
			Section:  "Body",
			Target:   ValidationTargetBodyMediaType,
			Severity: RequestSupportSeverityDowngraded,
			Summary:  "Body encoding details are not preserved yet.",
			Detail:   "This media type uses OpenAPI encoding metadata. Pane 3 can edit the raw body, but it cannot preserve or author those per-part encoding rules yet.",
		})
	}

	return notes
}

func capabilitySourceLabel(apiSpec *model.APISpec) string {
	if apiSpec == nil {
		return "this"
	}

	switch apiSpec.SourceFamily {
	case model.SourceFamilySwagger2:
		return "Swagger 2.0"
	case model.SourceFamilyOpenAPI3:
		if version := strings.TrimSpace(apiSpec.SourceVersion); version != "" {
			return "OpenAPI " + version
		}
		return "OpenAPI 3.x"
	default:
		if version := strings.TrimSpace(apiSpec.SourceVersion); version != "" {
			return version
		}
		return "this"
	}
}

func hasServerVariables(servers []model.Server) bool {
	for _, server := range servers {
		if len(server.Variables) > 0 {
			return true
		}
	}

	return false
}

func currentBodyMediaTypeHasEncodingWarning(apiSpec *model.APISpec, operation *model.Operation, draft *model.RequestDraft) bool {
	if apiSpec == nil {
		return false
	}

	mediaType, ok := currentBodyMediaType(operation, draft)
	if !ok {
		return false
	}

	path := "requestBody:" + mediaType
	for _, warning := range apiSpec.Warnings {
		if warning.Path != path {
			continue
		}
		if strings.Contains(warning.Message, "encoding details for media type") {
			return true
		}
	}

	return false
}

func currentBodyMediaType(operation *model.Operation, draft *model.RequestDraft) (string, bool) {
	if operation == nil || operation.RequestBody == nil || len(operation.RequestBody.Content) == 0 {
		return "", false
	}

	mediaType := strings.TrimSpace(operation.SelectedContentType)
	if draft != nil && strings.TrimSpace(draft.BodyMediaType) != "" {
		mediaType = strings.TrimSpace(draft.BodyMediaType)
	}
	if mediaType == "" {
		mediaType = operation.RequestBody.Content[0].MediaType
	}

	return mediaType, true
}
