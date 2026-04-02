package app

import (
	"testing"

	"github.com/phergul/apiscope/internal/model"
)

func TestEnsureRequestDraftSeedsKeyAndFirstBodyMediaType(t *testing.T) {
	t.Parallel()

	session := model.SessionState{
		SpecFingerprint:   "spec-123",
		SelectedServerURL: "https://api.example.com",
		RequestDrafts:     make(map[model.DraftKey]*model.RequestDraft),
	}
	operation := &model.Operation{
		Key: model.NewOperationKey("POST", "/pets"),
		RequestBody: &model.RequestBodySpec{
			Content: []model.MediaTypeSpec{
				{MediaType: "application/json"},
				{MediaType: "application/xml"},
			},
		},
	}

	draft := EnsureRequestDraft(&session, operation)

	wantKey := model.NewDraftKey("spec-123", operation.Key)
	if draft.Key != wantKey {
		t.Fatalf("expected draft key %q, got %q", wantKey, draft.Key)
	}
	if draft.SpecFingerprint != "spec-123" {
		t.Fatalf("expected fingerprint spec-123, got %q", draft.SpecFingerprint)
	}
	if draft.OperationKey != operation.Key {
		t.Fatalf("expected operation key %q, got %q", operation.Key, draft.OperationKey)
	}
	if draft.ServerURL != "https://api.example.com" {
		t.Fatalf("expected selected server to seed draft, got %q", draft.ServerURL)
	}
	if draft.BodyMediaType != "application/json" {
		t.Fatalf("expected first request body media type to be selected, got %q", draft.BodyMediaType)
	}
	if draft.BodyRaw != "" {
		t.Fatalf("expected body to stay empty when no examples or defaults exist, got %q", draft.BodyRaw)
	}
	if session.RequestDrafts[wantKey] != draft {
		t.Fatal("expected draft to be stored in session map")
	}
}

func TestEnsureRequestDraftSeedsParameterExamplesAndDefaults(t *testing.T) {
	t.Parallel()

	session := model.SessionState{
		SpecFingerprint: "spec-123",
		RequestDrafts:   make(map[model.DraftKey]*model.RequestDraft),
	}
	operation := &model.Operation{
		Key: model.NewOperationKey("POST", "/pets"),
		Parameters: []model.Parameter{
			{
				Name:    "petId",
				In:      model.ParameterLocationPath,
				Example: "abc123",
			},
			{
				Name:    "limit",
				In:      model.ParameterLocationQuery,
				Default: 25,
			},
			{
				Name:          "name",
				In:            model.ParameterLocationForm,
				FormInputKind: model.FormInputKindValue,
				Schema: &model.Schema{
					Example: "fido",
				},
			},
			{
				Name:          "upload",
				In:            model.ParameterLocationForm,
				FormInputKind: model.FormInputKindFile,
				Example:       "/tmp/demo.txt",
			},
		},
	}

	draft := EnsureRequestDraft(&session, operation)

	if got := draft.PathParams["petId"]; got != "abc123" {
		t.Fatalf("expected path example to seed draft, got %q", got)
	}
	if got := draft.QueryParams["limit"]; got != "25" {
		t.Fatalf("expected query default to seed draft, got %q", got)
	}
	if got := draft.FormParams["name"]; got != "fido" {
		t.Fatalf("expected form schema example to seed draft, got %q", got)
	}
	if got := draft.FormFileParams["upload"]; got != "" {
		t.Fatalf("expected file inputs to stay unset, got %q", got)
	}
}

func TestEnsureRequestDraftSeedsBodyFromMediaTypeExample(t *testing.T) {
	t.Parallel()

	session := model.SessionState{
		SpecFingerprint: "spec-123",
		RequestDrafts:   make(map[model.DraftKey]*model.RequestDraft),
	}
	operation := &model.Operation{
		Key: model.NewOperationKey("POST", "/pets"),
		RequestBody: &model.RequestBodySpec{
			Content: []model.MediaTypeSpec{
				{
					MediaType: "application/json",
					Example: map[string]any{
						"name": "fido",
					},
				},
			},
		},
	}

	draft := EnsureRequestDraft(&session, operation)

	if got := draft.BodyRaw; got != "{\n  \"name\": \"fido\"\n}" {
		t.Fatalf("expected media type example to seed body, got %q", got)
	}
}

func TestEnsureRequestDraftSeedsBodyFromNamedExampleAndTracksSelection(t *testing.T) {
	t.Parallel()

	session := model.SessionState{
		SpecFingerprint: "spec-123",
		RequestDrafts:   make(map[model.DraftKey]*model.RequestDraft),
	}
	operation := &model.Operation{
		Key: model.NewOperationKey("POST", "/pets"),
		RequestBody: &model.RequestBodySpec{
			Content: []model.MediaTypeSpec{
				{
					MediaType: "application/json",
					Examples: map[string]model.Example{
						"b-second": {Value: map[string]any{"name": "second"}},
						"a-first":  {Value: map[string]any{"name": "first"}},
					},
				},
			},
		},
	}

	draft := EnsureRequestDraft(&session, operation)

	if got := draft.BodyRaw; got != "{\n  \"name\": \"first\"\n}" {
		t.Fatalf("expected first named example to seed body deterministically, got %q", got)
	}
	if got := draft.SelectedExamples["body:application/json"]; got != "a-first" {
		t.Fatalf("expected named example selection to be tracked, got %q", got)
	}
}

func TestEnsureRequestDraftSeedsBodyFromSchemaDefaults(t *testing.T) {
	t.Parallel()

	session := model.SessionState{
		SpecFingerprint: "spec-123",
		RequestDrafts:   make(map[model.DraftKey]*model.RequestDraft),
	}
	operation := &model.Operation{
		Key: model.NewOperationKey("POST", "/pets"),
		RequestBody: &model.RequestBodySpec{
			Content: []model.MediaTypeSpec{
				{
					MediaType: "application/json",
					Schema: &model.Schema{
						Type: "object",
						Properties: map[string]*model.Schema{
							"name": {
								Type:    "string",
								Default: "fido",
							},
							"metadata": {
								Type: "object",
								Properties: map[string]*model.Schema{
									"region": {
										Type:    "string",
										Example: "ie",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	draft := EnsureRequestDraft(&session, operation)

	if got := draft.BodyRaw; got != "{\n  \"metadata\": {\n    \"region\": \"ie\"\n  },\n  \"name\": \"fido\"\n}" {
		t.Fatalf("expected schema defaults/examples to seed JSON body, got %q", got)
	}
}

func TestEnsureRequestDraftReusesExistingDraft(t *testing.T) {
	t.Parallel()

	key := model.NewDraftKey("spec-123", model.NewOperationKey("GET", "/pets"))
	existing := &model.RequestDraft{
		Key:             key,
		SpecFingerprint: "spec-123",
		OperationKey:    model.NewOperationKey("GET", "/pets"),
		QueryParams: map[string]string{
			"market": "IE",
		},
	}
	session := model.SessionState{
		SpecFingerprint: "spec-123",
		RequestDrafts: map[model.DraftKey]*model.RequestDraft{
			key: existing,
		},
	}
	operation := &model.Operation{Key: model.NewOperationKey("GET", "/pets")}

	draft := EnsureRequestDraft(&session, operation)

	if draft != existing {
		t.Fatal("expected existing draft to be reused")
	}
	if draft.QueryParams["market"] != "IE" {
		t.Fatalf("expected existing values to be preserved, got %#v", draft.QueryParams)
	}
}

func TestSetDraftParameterStoresAndClearsValuesByLocation(t *testing.T) {
	t.Parallel()

	session := model.SessionState{
		SpecFingerprint: "spec-123",
		RequestDrafts:   make(map[model.DraftKey]*model.RequestDraft),
	}
	operation := &model.Operation{Key: model.NewOperationKey("GET", "/pets")}
	parameter := model.Parameter{Name: "market", In: model.ParameterLocationQuery}

	draft := SetDraftParameter(&session, operation, parameter, "IE")
	if got := draft.QueryParams["market"]; got != "IE" {
		t.Fatalf("expected query param to be stored, got %q", got)
	}

	draft = SetDraftParameter(&session, operation, parameter, "")
	if _, ok := draft.QueryParams["market"]; ok {
		t.Fatalf("expected cleared query param to be removed, got %#v", draft.QueryParams)
	}
}

func TestSetDraftParameterStoresFormValues(t *testing.T) {
	t.Parallel()

	session := model.SessionState{
		SpecFingerprint: "spec-123",
		RequestDrafts:   make(map[model.DraftKey]*model.RequestDraft),
	}
	operation := &model.Operation{Key: model.NewOperationKey("POST", "/pets")}
	parameter := model.Parameter{Name: "name", In: model.ParameterLocationForm}

	draft := SetDraftParameter(&session, operation, parameter, "fido")
	if got := draft.FormParams["name"]; got != "fido" {
		t.Fatalf("expected form param to be stored, got %q", got)
	}

	draft = SetDraftParameter(&session, operation, parameter, "")
	if _, ok := draft.FormParams["name"]; ok {
		t.Fatalf("expected cleared form param to be removed, got %#v", draft.FormParams)
	}
}

func TestSetDraftParameterStoresFormFilePaths(t *testing.T) {
	t.Parallel()

	session := model.SessionState{
		SpecFingerprint: "spec-123",
		RequestDrafts:   make(map[model.DraftKey]*model.RequestDraft),
	}
	operation := &model.Operation{Key: model.NewOperationKey("POST", "/upload")}
	parameter := model.Parameter{Name: "file", In: model.ParameterLocationForm, FormInputKind: model.FormInputKindFile}

	draft := SetDraftParameter(&session, operation, parameter, "/tmp/demo.txt")
	if got := draft.FormFileParams["file"]; got != "/tmp/demo.txt" {
		t.Fatalf("expected file form param to be stored, got %q", got)
	}

	draft = SetDraftParameter(&session, operation, parameter, "")
	if _, ok := draft.FormFileParams["file"]; ok {
		t.Fatalf("expected cleared file form param to be removed, got %#v", draft.FormFileParams)
	}
}

func TestSetDraftBodyRawPreservesEmptyString(t *testing.T) {
	t.Parallel()

	session := model.SessionState{
		SpecFingerprint: "spec-123",
		RequestDrafts:   make(map[model.DraftKey]*model.RequestDraft),
	}
	operation := &model.Operation{Key: model.NewOperationKey("POST", "/pets")}

	draft := SetDraftBodyRaw(&session, operation, "")
	if draft.BodyRaw != "" {
		t.Fatalf("expected empty body string to be preserved, got %q", draft.BodyRaw)
	}

	draft = SetDraftBodyRaw(&session, operation, "{\"name\":\"fido\"}")
	if draft.BodyRaw != "{\"name\":\"fido\"}" {
		t.Fatalf("expected body text to be stored, got %q", draft.BodyRaw)
	}
}

func TestSetDraftBodyMediaTypeReseedsWhenBodyStillMatchesGeneratedSeed(t *testing.T) {
	t.Parallel()

	session := model.SessionState{
		SpecFingerprint: "spec-123",
		RequestDrafts:   make(map[model.DraftKey]*model.RequestDraft),
	}
	operation := &model.Operation{
		Key: model.NewOperationKey("POST", "/pets"),
		RequestBody: &model.RequestBodySpec{
			Content: []model.MediaTypeSpec{
				{
					MediaType: "application/json",
					Example:   map[string]any{"name": "json"},
				},
				{
					MediaType: "application/xml",
					Example:   "<pet><name>xml</name></pet>",
				},
			},
		},
	}

	draft := EnsureRequestDraft(&session, operation)
	if got := draft.BodyRaw; got != "{\n  \"name\": \"json\"\n}" {
		t.Fatalf("expected initial body seed, got %q", got)
	}

	draft = SetDraftBodyMediaType(&session, operation, "application/xml")
	if got := draft.BodyRaw; got != "<pet><name>xml</name></pet>" {
		t.Fatalf("expected seeded body to update with media type change, got %q", got)
	}
}

func TestSetDraftBodyMediaTypePreservesUserEditedBody(t *testing.T) {
	t.Parallel()

	session := model.SessionState{
		SpecFingerprint: "spec-123",
		RequestDrafts:   make(map[model.DraftKey]*model.RequestDraft),
	}
	operation := &model.Operation{
		Key: model.NewOperationKey("POST", "/pets"),
		RequestBody: &model.RequestBodySpec{
			Content: []model.MediaTypeSpec{
				{
					MediaType: "application/json",
					Example:   map[string]any{"name": "json"},
				},
				{
					MediaType: "application/xml",
					Example:   "<pet><name>xml</name></pet>",
				},
			},
		},
	}

	draft := EnsureRequestDraft(&session, operation)
	draft.BodyRaw = "{\"name\":\"custom\"}"

	draft = SetDraftBodyMediaType(&session, operation, "application/xml")
	if got := draft.BodyRaw; got != "{\"name\":\"custom\"}" {
		t.Fatalf("expected user-edited body to be preserved, got %q", got)
	}
}
