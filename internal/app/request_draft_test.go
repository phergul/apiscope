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
	if session.RequestDrafts[wantKey] != draft {
		t.Fatal("expected draft to be stored in session map")
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
