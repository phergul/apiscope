package app

import (
	"testing"

	"github.com/phergul/apiscope/internal/model"
)

func TestProjectRequestSupportNotesReportsUnsupportedContentParameter(t *testing.T) {
	t.Parallel()

	notes := ProjectRequestSupportNotes(&model.Operation{
		Parameters: []model.Parameter{
			{
				Name:    "legacy",
				In:      model.ParameterLocationQuery,
				Content: []model.MediaTypeSpec{{MediaType: "application/json"}},
			},
		},
	})
	if len(notes) != 1 {
		t.Fatalf("expected one support note, got %d", len(notes))
	}
	if notes[0].Section != "Query" {
		t.Fatalf("expected Query section, got %q", notes[0].Section)
	}
	if notes[0].Target != "query:legacy" {
		t.Fatalf("expected query target, got %q", notes[0].Target)
	}
	if notes[0].Severity != RequestSupportSeverityUnsupported {
		t.Fatalf("expected unsupported severity, got %q", notes[0].Severity)
	}
}

func TestProjectRequestSupportNotesReportsDowngradedCollectionFormat(t *testing.T) {
	t.Parallel()

	notes := ProjectRequestSupportNotes(&model.Operation{
		Parameters: []model.Parameter{
			{
				Name:             "tags",
				In:               model.ParameterLocationQuery,
				CollectionFormat: "pipes",
			},
		},
	})
	if len(notes) != 1 {
		t.Fatalf("expected one support note, got %d", len(notes))
	}
	if notes[0].Section != "Query" {
		t.Fatalf("expected Query section, got %q", notes[0].Section)
	}
	if notes[0].Target != "query:tags" {
		t.Fatalf("expected query target, got %q", notes[0].Target)
	}
	if notes[0].Severity != RequestSupportSeverityDowngraded {
		t.Fatalf("expected downgraded severity, got %q", notes[0].Severity)
	}
}

func TestProjectRequestSupportNotesReturnsNilWhenOperationHasNoRequestLimitations(t *testing.T) {
	t.Parallel()

	notes := ProjectRequestSupportNotes(&model.Operation{
		Parameters: []model.Parameter{
			{Name: "petId", In: model.ParameterLocationPath},
		},
	})
	if len(notes) != 0 {
		t.Fatalf("expected no support notes, got %#v", notes)
	}
}
