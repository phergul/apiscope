package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/phergul/apiscope/internal/model"
)

func TestCloneExecutionSessionCopiesMutableInputs(t *testing.T) {
	t.Parallel()

	operationKey := model.NewOperationKey("POST", "/pets")
	draftKey := model.NewDraftKey("spec-1", operationKey)
	original := model.SessionState{
		SpecFingerprint:   "spec-1",
		SelectedServerURL: "https://api.example.com",
		RequestDrafts: map[model.DraftKey]*model.RequestDraft{
			draftKey: {
				Key:             draftKey,
				SpecFingerprint: "spec-1",
				OperationKey:    operationKey,
				ServerURL:       "https://api.example.com",
				PathParams:      map[string]string{"petId": "abc"},
				FormParams:      map[string]string{"name": "fido"},
				FormFileParams:  map[string]string{"file": "/tmp/demo.txt"},
			},
		},
		AuthState: map[string]model.AuthValue{
			"api_key": {Type: model.AuthSchemeValueTypeAPIKey, APIKey: "secret"},
		},
	}

	cloned := CloneExecutionSession(original)
	cloned.RequestDrafts[draftKey].PathParams["petId"] = "changed"
	cloned.RequestDrafts[draftKey].FormParams["name"] = "spot"
	cloned.RequestDrafts[draftKey].FormFileParams["file"] = "/tmp/other.txt"
	cloned.AuthState["api_key"] = model.AuthValue{Type: model.AuthSchemeValueTypeAPIKey, APIKey: "other"}

	if got := original.RequestDrafts[draftKey].PathParams["petId"]; got != "abc" {
		t.Fatalf("expected original draft to stay unchanged, got %q", got)
	}
	if got := original.RequestDrafts[draftKey].FormParams["name"]; got != "fido" {
		t.Fatalf("expected original form draft to stay unchanged, got %q", got)
	}
	if got := original.RequestDrafts[draftKey].FormFileParams["file"]; got != "/tmp/demo.txt" {
		t.Fatalf("expected original file draft to stay unchanged, got %q", got)
	}
	if got := original.AuthState["api_key"].APIKey; got != "secret" {
		t.Fatalf("expected original auth state to stay unchanged, got %q", got)
	}
}

func TestServiceExecuteCurrentCapturesExecutedRequestSnapshot(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	operation := model.Operation{
		Key:    model.NewOperationKey("POST", "/pets/{petId}"),
		Method: "POST",
		Path:   "/pets/{petId}",
		Parameters: []model.Parameter{
			{Name: "petId", In: model.ParameterLocationPath, Required: true},
		},
		RequestBody: &model.RequestBodySpec{
			Required: true,
			Content:  []model.MediaTypeSpec{{MediaType: "application/json"}},
		},
		Security: &model.SecurityRequirement{
			Alternatives: []model.SecurityAlternative{
				{Schemes: []model.SecurityRequirementRef{{Name: "api_key"}}},
			},
		},
	}
	session := model.SessionState{
		SelectedServerURL:    server.URL,
		SelectedOperationKey: operation.Key,
		Spec: &model.APISpec{
			Operations: []model.Operation{operation},
			SecuritySchemes: map[string]model.SecurityScheme{
				"api_key": {
					Name:          "api_key",
					Type:          model.SecuritySchemeTypeAPIKey,
					In:            model.ParameterLocationHeader,
					ParameterName: "X-API-Key",
				},
			},
		},
		RequestDrafts: map[model.DraftKey]*model.RequestDraft{},
		AuthState: map[string]model.AuthValue{
			"api_key": {Type: model.AuthSchemeValueTypeAPIKey, APIKey: "secret"},
		},
	}
	draft := EnsureRequestDraft(&session, &operation)
	draft.PathParams["petId"] = "abc"
	draft.FormParams["name"] = "fido"
	draft.FormFileParams["file"] = "/tmp/demo.txt"
	draft.BodyMediaType = "application/json"
	draft.BodyRaw = `{"name":"fido"}`

	result := NewService(nil).ExecuteCurrent(context.Background(), CloneExecutionSession(session))
	if got := result.Snapshot.ServerURL; got != server.URL {
		t.Fatalf("expected snapshot server url, got %q", got)
	}
	if got := result.Snapshot.Draft.PathParams["petId"]; got != "abc" {
		t.Fatalf("expected snapshot path param, got %q", got)
	}
	if got := result.Snapshot.Draft.FormParams["name"]; got != "fido" {
		t.Fatalf("expected snapshot form param, got %q", got)
	}
	if got := result.Snapshot.Draft.FormFileParams["file"]; got != "/tmp/demo.txt" {
		t.Fatalf("expected snapshot file param, got %q", got)
	}
	if got := result.Snapshot.Draft.BodyRaw; got != `{"name":"fido"}` {
		t.Fatalf("expected snapshot body, got %q", got)
	}
	if got := result.Snapshot.AuthState["api_key"].APIKey; got != "secret" {
		t.Fatalf("expected snapshot auth state, got %q", got)
	}
}

func TestHistoryForOperationReturnsNewestFirst(t *testing.T) {
	t.Parallel()

	session := model.SessionState{
		RequestHistory: []model.HistoryEntry{
			{RequestID: 1, OperationKey: model.NewOperationKey("GET", "/pets")},
			{RequestID: 2, OperationKey: model.NewOperationKey("GET", "/health")},
			{RequestID: 3, OperationKey: model.NewOperationKey("GET", "/pets")},
		},
	}

	entries := HistoryForOperation(session, model.NewOperationKey("GET", "/pets"))
	if len(entries) != 2 {
		t.Fatalf("expected 2 filtered entries, got %d", len(entries))
	}
	if entries[0].RequestID != 3 || entries[1].RequestID != 1 {
		t.Fatalf("expected newest-first ordering, got %#v", entries)
	}
}

func TestLoadHistoryResponseOnlyUpdatesLastResponse(t *testing.T) {
	t.Parallel()

	operationKey := model.NewOperationKey("GET", "/pets")
	draftKey := model.NewDraftKey("spec-1", operationKey)
	session := model.SessionState{
		SelectedServerURL: "https://current.example.com",
		RequestDrafts: map[model.DraftKey]*model.RequestDraft{
			draftKey: {
				Key:          draftKey,
				OperationKey: operationKey,
				ServerURL:    "https://current.example.com",
				PathParams:   map[string]string{"petId": "current"},
			},
		},
		AuthState: map[string]model.AuthValue{
			"api_key": {Type: model.AuthSchemeValueTypeAPIKey, APIKey: "current"},
		},
	}

	ok := LoadHistoryResponse(&session, model.HistoryEntry{
		Response: &model.HTTPResponse{Status: "200 OK", PrettyBody: "old"},
		Request: model.ExecutedRequestSnapshot{
			ServerURL: "https://history.example.com",
		},
	})
	if !ok {
		t.Fatal("expected response recall to succeed")
	}
	if session.LastResponse == nil || session.LastResponse.Status != "200 OK" {
		t.Fatalf("expected recalled response, got %#v", session.LastResponse)
	}
	if got := session.SelectedServerURL; got != "https://current.example.com" {
		t.Fatalf("expected server selection to stay unchanged, got %q", got)
	}
	if got := session.RequestDrafts[draftKey].PathParams["petId"]; got != "current" {
		t.Fatalf("expected draft to stay unchanged, got %q", got)
	}
	if got := session.AuthState["api_key"].APIKey; got != "current" {
		t.Fatalf("expected auth state to stay unchanged, got %q", got)
	}
}

func TestRestoreHistoryRequestRestoresCurrentOperationInputs(t *testing.T) {
	t.Parallel()

	currentOp := model.NewOperationKey("GET", "/pets")
	otherOp := model.NewOperationKey("GET", "/health")
	currentDraftKey := model.NewDraftKey("spec-1", currentOp)
	otherDraftKey := model.NewDraftKey("spec-1", otherOp)
	session := model.SessionState{
		SelectedServerURL: "https://current.example.com",
		RequestDrafts: map[model.DraftKey]*model.RequestDraft{
			currentDraftKey: {
				Key:             currentDraftKey,
				SpecFingerprint: "spec-1",
				OperationKey:    currentOp,
				ServerURL:       "https://current.example.com",
				PathParams:      map[string]string{"petId": "current"},
			},
			otherDraftKey: {
				Key:             otherDraftKey,
				SpecFingerprint: "spec-1",
				OperationKey:    otherOp,
				ServerURL:       "https://current.example.com",
				QueryParams:     map[string]string{"check": "keep"},
			},
		},
		AuthState: map[string]model.AuthValue{
			"api_key": {Type: model.AuthSchemeValueTypeAPIKey, APIKey: "current"},
		},
	}

	ok := RestoreHistoryRequest(&session, model.HistoryEntry{
		Request: model.ExecutedRequestSnapshot{
			OperationKey: currentOp,
			ServerURL:    "https://history.example.com",
			Draft: model.RequestDraft{
				Key:             currentDraftKey,
				SpecFingerprint: "spec-1",
				OperationKey:    currentOp,
				ServerURL:       "https://history.example.com",
				PathParams:      map[string]string{"petId": "from-history"},
				FormParams:      map[string]string{"name": "from-history"},
				FormFileParams:  map[string]string{"file": "/tmp/from-history.txt"},
				BodyMediaType:   "application/json",
				BodyRaw:         `{"name":"fido"}`,
			},
			AuthState: map[string]model.AuthValue{
				"api_key": {Type: model.AuthSchemeValueTypeAPIKey, APIKey: "restored"},
			},
		},
	})
	if !ok {
		t.Fatal("expected request restore to succeed")
	}
	if got := session.SelectedServerURL; got != "https://history.example.com" {
		t.Fatalf("expected selected server restore, got %q", got)
	}
	if got := session.RequestDrafts[currentDraftKey].PathParams["petId"]; got != "from-history" {
		t.Fatalf("expected current draft restore, got %q", got)
	}
	if got := session.RequestDrafts[currentDraftKey].FormParams["name"]; got != "from-history" {
		t.Fatalf("expected current form draft restore, got %q", got)
	}
	if got := session.RequestDrafts[currentDraftKey].FormFileParams["file"]; got != "/tmp/from-history.txt" {
		t.Fatalf("expected current file draft restore, got %q", got)
	}
	if got := session.RequestDrafts[currentDraftKey].BodyRaw; got != `{"name":"fido"}` {
		t.Fatalf("expected current draft body restore, got %q", got)
	}
	if got := session.AuthState["api_key"].APIKey; got != "restored" {
		t.Fatalf("expected auth state restore, got %q", got)
	}
	if got := session.RequestDrafts[otherDraftKey].QueryParams["check"]; got != "keep" {
		t.Fatalf("expected unrelated draft values to stay unchanged, got %q", got)
	}
}
