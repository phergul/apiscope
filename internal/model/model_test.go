package model

import "testing"

func TestNewOperationKey(t *testing.T) {
	t.Parallel()

	got := NewOperationKey(" get ", "pets/{id}")

	if got != OperationKey("GET /pets/{id}") {
		t.Fatalf("unexpected operation key: %q", got)
	}
}

func TestSecurityRequirementPreservesOROfANDSemantics(t *testing.T) {
	t.Parallel()

	requirement := SecurityRequirement{
		Alternatives: []SecurityAlternative{
			{
				Schemes: []SecurityRequirementRef{
					{Name: "api_key"},
					{Name: "secondary_header"},
				},
			},
			{
				Schemes: []SecurityRequirementRef{
					{Name: "oauth", Scopes: []string{"pets:read"}},
				},
			},
		},
	}

	if len(requirement.Alternatives) != 2 {
		t.Fatalf("expected 2 alternatives, got %d", len(requirement.Alternatives))
	}
	if len(requirement.Alternatives[0].Schemes) != 2 {
		t.Fatalf("expected first alternative to preserve two AND-ed schemes, got %d", len(requirement.Alternatives[0].Schemes))
	}
	if requirement.Alternatives[1].Schemes[0].Scopes[0] != "pets:read" {
		t.Fatalf("expected scopes to be preserved, got %#v", requirement.Alternatives[1].Schemes[0].Scopes)
	}
}

func TestAPISpecPreservesSourceMetadataCapabilitiesAndWarnings(t *testing.T) {
	t.Parallel()

	spec := APISpec{
		Fingerprint:   "abc123",
		SourceFamily:  SourceFamilySwagger2,
		SourceVersion: "2.0",
		Capabilities: CapabilitySet{
			SupportsSwagger2Conversion: true,
			SupportsRequestBodies:      true,
		},
		Warnings: []SpecWarning{
			{
				Code:    SpecWarningUnsupportedFeature,
				Message: "callbacks are not supported in v1",
				Path:    "#/paths/~1pets/get/callbacks",
			},
		},
	}

	if spec.SourceFamily != SourceFamilySwagger2 {
		t.Fatalf("unexpected source family: %q", spec.SourceFamily)
	}
	if spec.SourceVersion != "2.0" {
		t.Fatalf("unexpected source version: %q", spec.SourceVersion)
	}
	if !spec.Capabilities.SupportsSwagger2Conversion {
		t.Fatal("expected swagger2 conversion capability to be preserved")
	}
	if len(spec.Warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(spec.Warnings))
	}
}

func TestNewDraftKeyUsesFingerprintAndOperationKey(t *testing.T) {
	t.Parallel()

	key := NewDraftKey("spec-hash", NewOperationKey("post", "/pets"))

	if key != DraftKey("spec-hash::POST /pets") {
		t.Fatalf("unexpected draft key: %q", key)
	}
}
