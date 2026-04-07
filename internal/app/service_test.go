package app

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/phergul/apiscope/internal/model"
	"github.com/phergul/apiscope/internal/persist"
	"github.com/phergul/apiscope/internal/spec"
	"github.com/phergul/apiscope/internal/transport"
)

type stubLoader struct {
	result    *model.APISpec
	err       error
	gotSource spec.Source
}

func (l *stubLoader) Load(ctx context.Context, source spec.Source) (*model.APISpec, error) {
	l.gotSource = source
	if l.err != nil {
		return nil, l.err
	}

	return l.result, nil
}

func TestServiceLoadSourceInitializesSessionAndView(t *testing.T) {
	t.Parallel()

	loader := &stubLoader{
		result: &model.APISpec{
			Fingerprint: "spec-123",
			Title:       "Demo API",
			Servers: []model.Server{
				{URL: "https://api.example.com"},
			},
			Operations: []model.Operation{
				{Key: model.NewOperationKey("GET", "/pets"), Method: "GET", Path: "/pets"},
				{Key: model.NewOperationKey("POST", "/pets"), Method: "POST", Path: "/pets"},
			},
		},
	}

	result, err := NewService(loader, nil, nil, nil).LoadSource(context.Background(), "spec.yaml")
	if err != nil {
		t.Fatalf("LoadSource returned error: %v", err)
	}

	if loader.gotSource.Value != "spec.yaml" {
		t.Fatalf("expected raw source to be forwarded, got %#v", loader.gotSource)
	}
	if result.Session.Spec == nil {
		t.Fatal("expected normalised spec to be stored in session")
	}
	if result.Session.SpecFingerprint != "spec-123" {
		t.Fatalf("expected fingerprint spec-123, got %q", result.Session.SpecFingerprint)
	}
	if result.Session.SelectedOperationKey != model.NewOperationKey("GET", "/pets") {
		t.Fatalf("expected first operation to be selected, got %q", result.Session.SelectedOperationKey)
	}
	if result.Session.SelectedServerURL != "https://api.example.com" {
		t.Fatalf("expected first server to be selected, got %q", result.Session.SelectedServerURL)
	}
	if result.View.FocusedPane != model.FocusedPaneOperations {
		t.Fatalf("expected operations pane focus, got %q", result.View.FocusedPane)
	}
	if result.View.ExpandedRightPane != model.FocusedPaneRequest {
		t.Fatalf("expected request pane to start expanded, got %q", result.View.ExpandedRightPane)
	}
	if result.View.ActiveEditorMode != model.EditorModeBrowse {
		t.Fatalf("expected browse mode, got %q", result.View.ActiveEditorMode)
	}
	if !result.View.OperationsPaneVisible {
		t.Fatal("expected operations pane to remain visible")
	}
	if result.View.ZoomedPane {
		t.Fatal("expected zoom mode to start disabled")
	}
	if len(result.View.VisibleOperationKeys) != 2 {
		t.Fatalf("expected 2 visible operations, got %d", len(result.View.VisibleOperationKeys))
	}
}

func TestServiceLoadSourceAllowsSpecsWithNoOperations(t *testing.T) {
	t.Parallel()

	loader := &stubLoader{
		result: &model.APISpec{
			Fingerprint: "empty-spec",
			Title:       "Empty API",
		},
	}

	result, err := NewService(loader, nil, nil, nil).LoadSource(context.Background(), "empty.yaml")
	if err != nil {
		t.Fatalf("LoadSource returned error: %v", err)
	}

	if result.Session.SelectedOperationKey != "" {
		t.Fatalf("expected no selected operation, got %q", result.Session.SelectedOperationKey)
	}
	if len(result.View.VisibleOperationKeys) != 0 {
		t.Fatalf("expected no visible operations, got %d", len(result.View.VisibleOperationKeys))
	}
	if result.Session.RequestDrafts == nil {
		t.Fatal("expected request drafts map to be initialized")
	}
	if result.Session.AuthState == nil {
		t.Fatal("expected auth state map to be initialized")
	}
}

func TestServiceLoadSourceReturnsLoaderErrors(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("boom")
	loader := &stubLoader{err: wantErr}

	_, err := NewService(loader, nil, nil, nil).LoadSource(context.Background(), "broken.yaml")
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
}

func TestServiceExecuteCurrentReturnsValidationIssuesBeforeTransport(t *testing.T) {
	t.Parallel()

	session := model.SessionState{
		SelectedServerURL:    "https://api.example.com",
		SelectedOperationKey: model.NewOperationKey("POST", "/pets/{petId}"),
		Spec: &model.APISpec{
			Operations: []model.Operation{
				{
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
				},
			},
		},
		RequestDrafts: map[model.DraftKey]*model.RequestDraft{},
	}

	result := NewService(nil, nil, nil, nil).ExecuteCurrent(context.Background(), session)
	if !result.Validation.HasIssues() {
		t.Fatal("expected validation errors before execution")
	}
	if result.Response != nil {
		t.Fatalf("expected no response when validation fails, got %#v", result.Response)
	}
}

func TestServiceExecuteCurrentBuildsAndExecutesRequest(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got != "/pets/abc" {
			t.Fatalf("expected path /pets/abc, got %q", got)
		}
		if got := r.URL.Query().Get("limit"); got != "10" {
			t.Fatalf("expected query limit=10, got %q", got)
		}
		if got := r.Header.Get("X-Trace-ID"); got != "trace-1" {
			t.Fatalf("expected X-Trace-ID header, got %q", got)
		}
		if got := r.Header.Get("X-API-Key"); got != "secret" {
			t.Fatalf("expected X-API-Key header, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
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
	draft.QueryParams["limit"] = "10"
	draft.HeaderParams["X-Trace-ID"] = "trace-1"
	draft.BodyMediaType = "application/json"
	draft.BodyRaw = `{"name":"fido"}`

	service := NewService(nil, transport.NewExecutor(server.Client(), nil), nil, nil)
	result := service.ExecuteCurrent(context.Background(), session)
	if result.Validation.HasIssues() {
		t.Fatalf("expected execution without validation issues, got %#v", result.Validation.Issues)
	}
	if result.Response == nil {
		t.Fatal("expected execution response")
	}
	if result.Response.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", result.Response.StatusCode)
	}
	if result.Response.OperationKey != operation.Key {
		t.Fatalf("expected response to track operation key, got %q", result.Response.OperationKey)
	}
}

func TestServiceExecuteCurrentReturnsAuthValidationIssuesBeforeTransport(t *testing.T) {
	t.Parallel()

	operation := model.Operation{
		Key:    model.NewOperationKey("GET", "/me"),
		Method: "GET",
		Path:   "/me",
		Security: &model.SecurityRequirement{
			Alternatives: []model.SecurityAlternative{
				{Schemes: []model.SecurityRequirementRef{{Name: "bearer_auth"}}},
			},
		},
	}
	session := model.SessionState{
		SelectedServerURL:    "https://api.example.com",
		SelectedOperationKey: operation.Key,
		Spec: &model.APISpec{
			Operations: []model.Operation{operation},
			SecuritySchemes: map[string]model.SecurityScheme{
				"bearer_auth": {
					Name:   "bearer_auth",
					Type:   model.SecuritySchemeTypeHTTP,
					Scheme: model.HTTPAuthSchemeBearer,
				},
			},
		},
		RequestDrafts: map[model.DraftKey]*model.RequestDraft{},
	}

	result := NewService(nil, nil, nil, nil).ExecuteCurrent(context.Background(), session)
	if !result.Validation.HasIssues() {
		t.Fatal("expected auth validation errors before execution")
	}
	if _, ok := result.Validation.IssueForTarget(AuthAlternativeFieldTarget(0, "bearer_auth", AuthFieldBearerToken)); !ok {
		t.Fatalf("expected missing bearer token issue, got %#v", result.Validation.Issues)
	}
	if result.Response != nil {
		t.Fatalf("expected no response when auth validation fails, got %#v", result.Response)
	}
}

func TestServiceLoadSourceBuildsStablePersistenceScopeKeyAcrossSpecEdits(t *testing.T) {
	t.Parallel()

	rawSource := "spec.yaml"
	loaderA := &stubLoader{
		result: &model.APISpec{
			Fingerprint:  "spec-a",
			SourceFamily: model.SourceFamilyOpenAPI3,
			Operations:   []model.Operation{{Key: model.NewOperationKey("GET", "/pets"), Method: "GET", Path: "/pets"}},
		},
	}
	loaderB := &stubLoader{
		result: &model.APISpec{
			Fingerprint:  "spec-b",
			SourceFamily: model.SourceFamilyOpenAPI3,
			Operations:   []model.Operation{{Key: model.NewOperationKey("POST", "/pets"), Method: "POST", Path: "/pets"}},
		},
	}

	resultA, err := NewService(loaderA, nil, nil, nil).LoadSource(context.Background(), rawSource)
	if err != nil {
		t.Fatalf("first LoadSource returned error: %v", err)
	}
	resultB, err := NewService(loaderB, nil, nil, nil).LoadSource(context.Background(), rawSource)
	if err != nil {
		t.Fatalf("second LoadSource returned error: %v", err)
	}

	if resultA.Session.PersistenceScopeKey == "" {
		t.Fatal("expected persistence scope key to be set")
	}
	if resultA.Session.PersistenceScopeKey != resultB.Session.PersistenceScopeKey {
		t.Fatalf("expected persistence scope key to survive spec edits, got %q and %q", resultA.Session.PersistenceScopeKey, resultB.Session.PersistenceScopeKey)
	}
}

func TestServiceLoadSourceHydratesPersistedStateAndRecentSpecs(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	store := persist.NewStore(tempDir)
	scopeKey := model.NewPersistenceScopeKey("spec.yaml", model.SourceFamilyOpenAPI3)
	operationKey := model.NewOperationKey("GET", "/pets")

	if err := store.SaveEnvironments([]model.SavedEnvironment{{
		Name:              "staging",
		ScopeKey:          scopeKey,
		SelectedServerURL: "https://staging.example.com",
		AuthBindings: map[string]model.SavedAuthBinding{
			"api_key": {
				FieldEnvVars: map[model.AuthField]string{
					model.AuthFieldAPIKey: "APISCOPE_TEST_API_KEY",
				},
			},
		},
	}}); err != nil {
		t.Fatalf("SaveEnvironments returned error: %v", err)
	}
	if err := store.SaveHistory([]model.PersistedHistoryBucket{{
		ScopeKey:     scopeKey,
		OperationKey: operationKey,
		Entries: []model.HistoryEntry{{
			RequestID:    41,
			OperationKey: operationKey,
			Request: model.ExecutedRequestSnapshot{
				OperationKey: operationKey,
				ServerURL:    "https://history.example.com",
				Draft: model.RequestDraft{
					Key:             model.NewDraftKey("old-spec", operationKey),
					SpecFingerprint: "old-spec",
					OperationKey:    operationKey,
				},
			},
			Response: &model.HTTPResponse{
				OperationKey: operationKey,
				RequestID:    41,
				StatusCode:   http.StatusOK,
				Status:       "200 OK",
			},
		}},
	}, {
		ScopeKey:     scopeKey,
		OperationKey: model.NewOperationKey("GET", "/missing"),
		Entries:      []model.HistoryEntry{{RequestID: 99, OperationKey: model.NewOperationKey("GET", "/missing")}},
	}}); err != nil {
		t.Fatalf("SaveHistory returned error: %v", err)
	}

	service := NewService(&stubLoader{
		result: &model.APISpec{
			Fingerprint:  "new-spec",
			Title:        "Demo API",
			SourceFamily: model.SourceFamilyOpenAPI3,
			Operations: []model.Operation{{
				Key:    operationKey,
				Method: "GET",
				Path:   "/pets",
			}},
		},
	}, nil, store, nil)
	service.lookupEnv = func(name string) (string, bool) {
		if name == "APISCOPE_TEST_API_KEY" {
			return "secret", true
		}
		return "", false
	}
	now := time.Date(2026, time.April, 2, 11, 22, 33, 0, time.UTC)
	service.now = func() time.Time { return now }

	result, err := service.LoadSource(context.Background(), "spec.yaml")
	if err != nil {
		t.Fatalf("LoadSource returned error: %v", err)
	}

	if len(result.Environments) != 1 || result.Environments[0].Name != "staging" {
		t.Fatalf("expected hydrated environments, got %#v", result.Environments)
	}
	if len(result.Session.RequestHistory) != 1 {
		t.Fatalf("expected one compatible history entry, got %#v", result.Session.RequestHistory)
	}
	if got := result.Session.RequestHistory[0].Request.Draft.SpecFingerprint; got != "new-spec" {
		t.Fatalf("expected hydrated history draft fingerprint to rekey to the loaded spec, got %q", got)
	}
	if got := result.Session.RequestHistory[0].Request.Draft.Key; got != model.NewDraftKey("new-spec", operationKey) {
		t.Fatalf("expected hydrated history draft key to rekey to the loaded spec, got %q", got)
	}
	if result.Session.RequestHistory[0].Request.AuthState != nil {
		t.Fatalf("expected hydrated history auth snapshot to stay empty, got %#v", result.Session.RequestHistory[0].Request.AuthState)
	}
	if result.Session.ActiveExecRequestID != 41 {
		t.Fatalf("expected next request id baseline 41, got %d", result.Session.ActiveExecRequestID)
	}
	if result.View.ActiveExecuteRequestID != 41 {
		t.Fatalf("expected view request id baseline 41, got %d", result.View.ActiveExecuteRequestID)
	}

	config, err := store.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}
	if len(config.RecentSpecs) != 1 {
		t.Fatalf("expected one recent spec, got %#v", config.RecentSpecs)
	}
	if config.RecentSpecs[0].Source != "spec.yaml" {
		t.Fatalf("expected recent spec source spec.yaml, got %#v", config.RecentSpecs[0])
	}
	if !config.RecentSpecs[0].LastOpenedAt.Equal(now) {
		t.Fatalf("expected recent spec timestamp %v, got %v", now, config.RecentSpecs[0].LastOpenedAt)
	}
}

func TestUpdateRecentSpecsDedupesAndCaps(t *testing.T) {
	t.Parallel()

	existing := make([]model.RecentSpec, 0, 10)
	for index := 0; index < 10; index++ {
		existing = append(existing, model.RecentSpec{Source: "spec-" + strconv.Itoa(index)})
	}

	updated := updateRecentSpecs(existing, model.RecentSpec{Source: "spec-4"})
	if len(updated) != 10 {
		t.Fatalf("expected capped recent specs, got %d", len(updated))
	}
	if updated[0].Source != "spec-4" {
		t.Fatalf("expected most recent source at the front, got %#v", updated[0])
	}

	count := 0
	for _, spec := range updated {
		if spec.Source == "spec-4" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected spec-4 to be deduped, got %#v", updated)
	}
}

func TestServicePersistHistoryEntryPrunesPerOperationBucket(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	store := persist.NewStore(tempDir)
	service := NewService(nil, nil, store, nil)
	session := model.SessionState{
		PersistenceScopeKey: model.NewPersistenceScopeKey("spec.yaml", model.SourceFamilyOpenAPI3),
	}
	operationKey := model.NewOperationKey("GET", "/pets")

	for requestID := 1; requestID <= 17; requestID++ {
		if err := service.PersistHistoryEntry(session, model.HistoryEntry{
			RequestID:    uint64(requestID),
			OperationKey: operationKey,
		}); err != nil {
			t.Fatalf("PersistHistoryEntry returned error: %v", err)
		}
	}

	buckets, err := store.LoadHistory()
	if err != nil {
		t.Fatalf("LoadHistory returned error: %v", err)
	}
	if len(buckets) != 1 {
		t.Fatalf("expected one history bucket, got %#v", buckets)
	}
	if len(buckets[0].Entries) != maxPersistedHistoryEntries {
		t.Fatalf("expected %d pruned entries, got %d", maxPersistedHistoryEntries, len(buckets[0].Entries))
	}
	if buckets[0].Entries[0].RequestID != 3 || buckets[0].Entries[len(buckets[0].Entries)-1].RequestID != 17 {
		t.Fatalf("expected pruning to keep the newest entries, got %#v", buckets[0].Entries)
	}
}
