package request

import (
	"testing"

	"github.com/phergul/apiscope/internal/app"
	"github.com/phergul/apiscope/internal/model"
)

func TestStartEditReturnsFieldEditorStateForEditableParameter(t *testing.T) {
	t.Parallel()

	parameter := model.Parameter{
		Name: "petId",
		In:   model.ParameterLocationPath,
	}
	got := StartEdit(
		&model.Operation{Parameters: []model.Parameter{parameter}},
		&model.RequestDraft{PathParams: map[string]string{"petId": "42"}},
		[]RowDescriptor{{
			ID:        "path:petId",
			Kind:      RowKindParameter,
			Parameter: &parameter,
			Editable:  true,
		}},
		0,
		nil,
		nil,
	)

	if got.Kind != model.RequestEditKindField {
		t.Fatalf("expected field edit kind, got %q", got.Kind)
	}
	if got.Buffer != "42" {
		t.Fatalf("expected draft value to seed edit buffer, got %q", got.Buffer)
	}
	if !got.FocusField {
		t.Fatal("expected field editor to request focus")
	}
}

func TestCycleBodyMediaTypeAdvancesDraftSelection(t *testing.T) {
	t.Parallel()

	selected := &model.Operation{
		Key: model.NewOperationKey("POST", "/pets"),
		RequestBody: &model.RequestBodySpec{
			Content: []model.MediaTypeSpec{
				{MediaType: "application/json"},
				{MediaType: "application/xml"},
			},
		},
	}
	session := model.SessionState{
		RequestDrafts: map[model.DraftKey]*model.RequestDraft{},
	}
	app.EnsureRequestDraft(&session, selected)

	ok := CycleBodyMediaType(&session, selected)
	if !ok {
		t.Fatal("expected body media type cycle to succeed")
	}

	draft := app.EnsureRequestDraft(&session, selected)
	if draft.BodyMediaType != "application/xml" {
		t.Fatalf("expected body media type to advance, got %q", draft.BodyMediaType)
	}
}

func TestCycleBodyExampleAdvancesDraftSelection(t *testing.T) {
	t.Parallel()

	selected := &model.Operation{
		Key: model.NewOperationKey("POST", "/pets"),
		RequestBody: &model.RequestBodySpec{
			Content: []model.MediaTypeSpec{{
				MediaType: "application/json",
				Examples: map[string]model.Example{
					"a-first":  {Value: map[string]any{"name": "first"}},
					"b-second": {Value: map[string]any{"name": "second"}},
				},
			}},
		},
	}
	session := model.SessionState{
		RequestDrafts: map[model.DraftKey]*model.RequestDraft{},
	}
	app.EnsureRequestDraft(&session, selected)

	ok := CycleBodyExample(&session, selected)
	if !ok {
		t.Fatal("expected body example cycle to succeed")
	}

	draft := app.EnsureRequestDraft(&session, selected)
	if got := draft.SelectedExamples["body:application/json"]; got != "b-second" {
		t.Fatalf("expected body example to advance, got %q", got)
	}
}

func TestStartEditReturnsBodyExampleCycleAction(t *testing.T) {
	t.Parallel()

	got := StartEdit(
		&model.Operation{Key: model.NewOperationKey("POST", "/pets")},
		nil,
		[]RowDescriptor{{
			ID:       "body:example",
			Kind:     RowKindBodyExample,
			Editable: true,
		}},
		0,
		nil,
		nil,
	)

	if !got.CycleBodyExample {
		t.Fatal("expected body example row to trigger an example cycle action")
	}
	if got.Kind != model.RequestEditKindNone {
		t.Fatalf("expected no editor to open for body example row, got %q", got.Kind)
	}
}

func TestStartEditReturnsServerCycleAction(t *testing.T) {
	t.Parallel()

	got := StartEdit(
		&model.Operation{Key: model.NewOperationKey("GET", "/pets")},
		nil,
		[]RowDescriptor{{
			ID:       "server:url",
			Kind:     RowKindServer,
			Editable: true,
		}},
		0,
		nil,
		nil,
	)

	if !got.CycleServerURL {
		t.Fatal("expected server row to trigger a server cycle action")
	}
	if got.Kind != model.RequestEditKindNone {
		t.Fatalf("expected no editor to open for server row, got %q", got.Kind)
	}
}

func TestStartEditReturnsFieldEditorStateForEnvironmentBinding(t *testing.T) {
	t.Parallel()

	got := StartEdit(
		&model.Operation{Key: model.NewOperationKey("GET", "/pets")},
		nil,
		[]RowDescriptor{{
			ID:       "environment:binding:api_key:api_key",
			Kind:     RowKindEnvironmentBinding,
			Value:    "API_KEY_ENV",
			Editable: true,
		}},
		0,
		nil,
		nil,
	)

	if got.Kind != model.RequestEditKindField {
		t.Fatalf("expected field edit kind, got %q", got.Kind)
	}
	if got.Buffer != "API_KEY_ENV" {
		t.Fatalf("expected env var name to seed edit buffer, got %q", got.Buffer)
	}
	if !got.FocusField {
		t.Fatal("expected binding editor to request focus")
	}
}

func TestCycleServerURLAdvancesSelectedServer(t *testing.T) {
	t.Parallel()

	session := model.SessionState{
		SelectedServerURL: "https://api.example.com",
	}

	ok := CycleServerURL(&session, []model.Server{
		{URL: "https://api.example.com"},
		{URL: "https://staging.example.com"},
	})
	if !ok {
		t.Fatal("expected server cycle to succeed")
	}
	if session.SelectedServerURL != "https://staging.example.com" {
		t.Fatalf("expected selected server to advance, got %q", session.SelectedServerURL)
	}
}

func TestSaveEditWritesBodyBufferToDraft(t *testing.T) {
	t.Parallel()

	selected := &model.Operation{Key: model.NewOperationKey("POST", "/pets")}
	session := model.SessionState{
		RequestDrafts: map[model.DraftKey]*model.RequestDraft{},
	}

	ok := SaveEdit(
		&session,
		selected,
		[]RowDescriptor{{ID: "body:raw", Kind: RowKindBodyText, Editable: true}},
		0,
		model.RequestEditKindBody,
		"{\"name\":\"fido\"}",
		nil,
	)
	if !ok {
		t.Fatal("expected save to succeed")
	}

	draft := app.EnsureRequestDraft(&session, selected)
	if draft.BodyRaw != "{\"name\":\"fido\"}" {
		t.Fatalf("expected body draft to be saved, got %q", draft.BodyRaw)
	}
}

func TestSaveEditWritesFormFieldToDraft(t *testing.T) {
	t.Parallel()

	session := model.SessionState{
		RequestDrafts: map[model.DraftKey]*model.RequestDraft{},
	}
	selected := &model.Operation{Key: model.NewOperationKey("POST", "/pets")}
	parameter := model.Parameter{Name: "name", In: model.ParameterLocationForm}

	ok := SaveEdit(
		&session,
		selected,
		[]RowDescriptor{{
			ID:        "form:name",
			Kind:      RowKindParameter,
			Parameter: &parameter,
			Editable:  true,
		}},
		0,
		model.RequestEditKindField,
		"fido",
		nil,
	)
	if !ok {
		t.Fatal("expected save to succeed")
	}

	draft := app.EnsureRequestDraft(&session, selected)
	if got := draft.FormParams["name"]; got != "fido" {
		t.Fatalf("expected form draft to be saved, got %q", got)
	}
}

func TestSaveEditWritesFormFilePathToDraft(t *testing.T) {
	t.Parallel()

	session := model.SessionState{
		RequestDrafts: map[model.DraftKey]*model.RequestDraft{},
	}
	selected := &model.Operation{Key: model.NewOperationKey("POST", "/upload")}
	parameter := model.Parameter{Name: "file", In: model.ParameterLocationForm, FormInputKind: model.FormInputKindFile}

	ok := SaveEdit(
		&session,
		selected,
		[]RowDescriptor{{
			ID:        "form:file",
			Kind:      RowKindParameter,
			Parameter: &parameter,
			Editable:  true,
		}},
		0,
		model.RequestEditKindField,
		"/tmp/demo.txt",
		nil,
	)
	if !ok {
		t.Fatal("expected save to succeed")
	}

	draft := app.EnsureRequestDraft(&session, selected)
	if got := draft.FormFileParams["file"]; got != "/tmp/demo.txt" {
		t.Fatalf("expected form file draft to be saved, got %q", got)
	}
}

func TestStartEditSeedsAuthFieldBufferFromSessionAuthState(t *testing.T) {
	t.Parallel()

	scheme := model.SecurityScheme{
		Name:          "api_key",
		Type:          model.SecuritySchemeTypeAPIKey,
		In:            model.ParameterLocationHeader,
		ParameterName: "X-API-Key",
	}
	got := StartEdit(
		&model.Operation{Key: model.NewOperationKey("GET", "/pets")},
		nil,
		[]RowDescriptor{{
			ID:             "auth:api_key:api_key",
			Kind:           RowKindAuthField,
			AuthSchemeName: "api_key",
			AuthField:      app.AuthFieldAPIKey,
			Editable:       true,
		}},
		0,
		map[string]model.SecurityScheme{"api_key": scheme},
		map[string]model.AuthValue{"api_key": {Type: model.AuthSchemeValueTypeAPIKey, APIKey: "secret"}},
	)

	if got.Kind != model.RequestEditKindField {
		t.Fatalf("expected field edit kind, got %q", got.Kind)
	}
	if got.Buffer != "secret" {
		t.Fatalf("expected auth value to seed edit buffer, got %q", got.Buffer)
	}
}

func TestSaveEditWritesAuthFieldToSessionState(t *testing.T) {
	t.Parallel()

	session := model.SessionState{}
	selected := &model.Operation{Key: model.NewOperationKey("GET", "/pets")}
	scheme := model.SecurityScheme{
		Name:   "bearer_auth",
		Type:   model.SecuritySchemeTypeHTTP,
		Scheme: model.HTTPAuthSchemeBearer,
	}

	ok := SaveEdit(
		&session,
		selected,
		[]RowDescriptor{{
			ID:             "auth:bearer_auth:bearer_token",
			Kind:           RowKindAuthField,
			AuthSchemeName: "bearer_auth",
			AuthField:      app.AuthFieldBearerToken,
			Editable:       true,
		}},
		0,
		model.RequestEditKindField,
		"token-123",
		map[string]model.SecurityScheme{"bearer_auth": scheme},
	)
	if !ok {
		t.Fatal("expected auth save to succeed")
	}
	if got := session.AuthState["bearer_auth"].BearerToken; got != "token-123" {
		t.Fatalf("expected bearer token to be saved, got %q", got)
	}
}
