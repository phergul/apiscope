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

func TestProjectCapabilityWarningsReportSwaggerVersionLimits(t *testing.T) {
	t.Parallel()

	warnings := ProjectCapabilityWarnings(&model.APISpec{
		SourceFamily:  model.SourceFamilySwagger2,
		SourceVersion: "2.0",
		Capabilities: model.CapabilitySet{
			SupportsSwagger2Conversion: true,
			SupportsRequestBodies:      true,
			SupportsSecuritySchemes:    true,
		},
	})
	if len(warnings) != 2 {
		t.Fatalf("expected two capability warnings, got %#v", warnings)
	}
	if warnings[0].Message != "cookie parameters are unavailable for Swagger 2.0 specs" {
		t.Fatalf("expected cookie warning, got %#v", warnings[0])
	}
	if warnings[1].Message != "server variables are unavailable for Swagger 2.0 specs" {
		t.Fatalf("expected server-variable warning, got %#v", warnings[1])
	}
}

func TestProjectCapabilityWarningsReturnsNilForOpenAPI3Spec(t *testing.T) {
	t.Parallel()

	warnings := ProjectCapabilityWarnings(&model.APISpec{
		SourceFamily:  model.SourceFamilyOpenAPI3,
		SourceVersion: "3.0.3",
		Capabilities: model.CapabilitySet{
			SupportsOpenAPI3:         true,
			SupportsCookieParameters: true,
			SupportsRequestBodies:    true,
			SupportsServerVariables:  true,
			SupportsSecuritySchemes:  true,
		},
	})
	if len(warnings) != 0 {
		t.Fatalf("expected no capability warnings, got %#v", warnings)
	}
}

func TestProjectCapabilityRequestSupportNotesReportsTemplatedServers(t *testing.T) {
	t.Parallel()

	notes := ProjectCapabilityRequestSupportNotes(&model.APISpec{
		Capabilities: model.CapabilitySet{
			SupportsOpenAPI3:         true,
			SupportsCookieParameters: true,
			SupportsRequestBodies:    true,
			SupportsServerVariables:  true,
			SupportsSecuritySchemes:  true,
		},
	}, nil, nil, []model.Server{{
		URL: "https://{env}.example.com",
		Variables: map[string]model.ServerVariable{
			"env": {Default: "api"},
		},
	}})
	if len(notes) != 1 {
		t.Fatalf("expected one capability support note, got %#v", notes)
	}
	if notes[0].Section != "Server" {
		t.Fatalf("expected server section note, got %#v", notes[0])
	}
	if notes[0].Severity != RequestSupportSeverityDowngraded {
		t.Fatalf("expected downgraded severity, got %#v", notes[0])
	}
}

func TestProjectCapabilityRequestSupportNotesReportsBodyEncodingWarning(t *testing.T) {
	t.Parallel()

	notes := ProjectCapabilityRequestSupportNotes(&model.APISpec{
		Capabilities: model.CapabilitySet{
			SupportsOpenAPI3: true,
		},
		Warnings: []model.SpecWarning{{
			Code:    model.SpecWarningDowngradedFeature,
			Message: `encoding details for media type "multipart/form-data" were not preserved in the normalised model`,
			Path:    "requestBody:multipart/form-data",
		}},
	}, &model.Operation{
		RequestBody: &model.RequestBodySpec{
			Content: []model.MediaTypeSpec{{MediaType: "multipart/form-data"}},
		},
	}, &model.RequestDraft{BodyMediaType: "multipart/form-data"}, nil)
	if len(notes) != 1 {
		t.Fatalf("expected one encoding note, got %#v", notes)
	}
	if notes[0].Summary != "Body encoding details are not preserved yet." {
		t.Fatalf("unexpected encoding note %#v", notes[0])
	}
	if notes[0].Target != ValidationTargetBodyMediaType {
		t.Fatalf("expected media-type target, got %#v", notes[0])
	}
}
