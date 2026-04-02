package app

import (
	"fmt"

	"github.com/phergul/apiscope/internal/model"
)

// RequestSupportSeverity describes how strongly the UI should call out a request-input limitation.
type RequestSupportSeverity string

const (
	RequestSupportSeverityUnsupported RequestSupportSeverity = "unsupported"
	RequestSupportSeverityDowngraded  RequestSupportSeverity = "downgraded"
)

// RequestSupportNote describes one non-blocking request-input support note for pane 3.
type RequestSupportNote struct {
	Section  string
	Target   string
	Severity RequestSupportSeverity
	Summary  string
	Detail   string
}

// ProjectRequestSupportNotes reports request-input limitations for the selected operation.
func ProjectRequestSupportNotes(operation *model.Operation) []RequestSupportNote {
	if operation == nil {
		return nil
	}

	notes := make([]RequestSupportNote, 0, len(operation.Parameters))
	for _, parameter := range operation.Parameters {
		target := string(parameter.In) + ":" + parameter.Name
		section := requestSectionForLocation(parameter.In)
		if len(parameter.Content) > 0 {
			notes = append(notes, RequestSupportNote{
				Section:  section,
				Target:   target,
				Severity: RequestSupportSeverityUnsupported,
				Summary:  "Content-based parameter is read-only.",
				Detail:   "This parameter uses media-type content. Pane 3 cannot edit or send it yet.",
			})
		}
		if parameter.CollectionFormat != "" {
			notes = append(notes, RequestSupportNote{
				Section:  section,
				Target:   target,
				Severity: RequestSupportSeverityDowngraded,
				Summary:  fmt.Sprintf("Swagger collectionFormat %q needs manual formatting.", parameter.CollectionFormat),
				Detail:   "Enter the fully formatted value yourself. Request preparation does not serialize arrays from preserved Swagger collectionFormat values yet.",
			})
		}
	}

	return notes
}
