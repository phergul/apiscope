package app

import (
	"testing"

	"github.com/phergul/apiscope/internal/model"
)

func TestDiffSpecsReportsOperationCapabilityAndWarningChanges(t *testing.T) {
	t.Parallel()

	previous := &model.APISpec{
		Fingerprint:   "a",
		SourceFamily:  model.SourceFamilySwagger2,
		SourceVersion: "2.0",
		Capabilities: model.CapabilitySet{
			SupportsSwagger2Conversion: true,
			SupportsRequestBodies:      true,
			SupportsSecuritySchemes:    true,
		},
		Warnings: []model.SpecWarning{{
			Code:    model.SpecWarningUnsupportedFeature,
			Path:    "source version",
			Message: "cookie parameters are unavailable for Swagger 2.0 specs",
		}},
		Operations: []model.Operation{
			{Key: model.NewOperationKey("GET", "/pets"), Method: "GET", Path: "/pets", Summary: "list"},
			{Key: model.NewOperationKey("POST", "/pets"), Method: "POST", Path: "/pets"},
		},
	}
	next := &model.APISpec{
		Fingerprint:   "b",
		SourceFamily:  model.SourceFamilyOpenAPI3,
		SourceVersion: "3.0.3",
		Capabilities: model.CapabilitySet{
			SupportsOpenAPI3:         true,
			SupportsCookieParameters: true,
			SupportsRequestBodies:    true,
			SupportsServerVariables:  true,
			SupportsSecuritySchemes:  true,
		},
		Warnings: []model.SpecWarning{{
			Code:    model.SpecWarningDowngradedFeature,
			Path:    "requestBody:multipart/form-data",
			Message: `encoding details for media type "multipart/form-data" were not preserved in the normalised model`,
		}},
		Operations: []model.Operation{
			{Key: model.NewOperationKey("GET", "/pets"), Method: "GET", Path: "/pets", Summary: "list all pets"},
			{Key: model.NewOperationKey("GET", "/admin"), Method: "GET", Path: "/admin"},
		},
	}

	diff := DiffSpecs(previous, next)
	if !diff.Changed {
		t.Fatal("expected diff to report changes")
	}
	if len(diff.AddedOperations) != 1 || diff.AddedOperations[0] != model.NewOperationKey("GET", "/admin") {
		t.Fatalf("expected added operation GET /admin, got %#v", diff.AddedOperations)
	}
	if len(diff.RemovedOperations) != 1 || diff.RemovedOperations[0] != model.NewOperationKey("POST", "/pets") {
		t.Fatalf("expected removed operation POST /pets, got %#v", diff.RemovedOperations)
	}
	if len(diff.ChangedOperations) != 1 || diff.ChangedOperations[0] != model.NewOperationKey("GET", "/pets") {
		t.Fatalf("expected changed operation GET /pets, got %#v", diff.ChangedOperations)
	}
	if len(diff.CapabilityChanges) == 0 {
		t.Fatalf("expected capability changes, got %#v", diff.CapabilityChanges)
	}
	if len(diff.AddedWarnings) != 1 || len(diff.RemovedWarnings) != 1 {
		t.Fatalf("expected warning deltas, got added=%#v removed=%#v", diff.AddedWarnings, diff.RemovedWarnings)
	}
}

func TestDiffSpecsReportsUnchangedForEquivalentSpecs(t *testing.T) {
	t.Parallel()

	spec := &model.APISpec{
		Fingerprint:   "same",
		SourceFamily:  model.SourceFamilyOpenAPI3,
		SourceVersion: "3.1.0",
		Capabilities: model.CapabilitySet{
			SupportsOpenAPI3:         true,
			SupportsCookieParameters: true,
			SupportsRequestBodies:    true,
			SupportsServerVariables:  true,
			SupportsSecuritySchemes:  true,
		},
		Operations: []model.Operation{{
			Key:    model.NewOperationKey("GET", "/pets"),
			Method: "GET",
			Path:   "/pets",
		}},
	}

	diff := DiffSpecs(spec, spec)
	if diff.Changed {
		t.Fatalf("expected unchanged diff, got %#v", diff)
	}
	if len(diff.AddedOperations) != 0 || len(diff.RemovedOperations) != 0 || len(diff.ChangedOperations) != 0 {
		t.Fatalf("expected no operation deltas, got %#v", diff)
	}
}
