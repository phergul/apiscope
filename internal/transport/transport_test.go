package transport

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/phergul/apiscope/internal/model"
)

func TestPrepareRequestBuildsPathQueryHeadersCookiesAndBody(t *testing.T) {
	t.Parallel()

	executor := NewExecutor(nil)
	operation := &model.Operation{
		Method: "POST",
		Path:   "/pets/{petId}",
	}
	draft := &model.RequestDraft{
		PathParams:    map[string]string{"petId": "abc"},
		QueryParams:   map[string]string{"limit": "10"},
		HeaderParams:  map[string]string{"X-Trace-ID": "trace-1"},
		CookieParams:  map[string]string{"session": "cookie-1"},
		BodyMediaType: "application/json",
		BodyRaw:       `{"name":"fido"}`,
	}

	request, err := executor.PrepareRequest(operation, draft, "https://api.example.com", nil, nil, nil)
	if err != nil {
		t.Fatalf("PrepareRequest returned error: %v", err)
	}
	if got := request.URL.String(); got != "https://api.example.com/pets/abc?limit=10" {
		t.Fatalf("unexpected prepared URL %q", got)
	}
	if got := request.Header.Get("X-Trace-ID"); got != "trace-1" {
		t.Fatalf("expected header to be set, got %q", got)
	}
	if got := request.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected content type to be set, got %q", got)
	}
	cookie, err := request.Cookie("session")
	if err != nil {
		t.Fatalf("expected cookie to be set: %v", err)
	}
	if cookie.Value != "cookie-1" {
		t.Fatalf("expected cookie value cookie-1, got %q", cookie.Value)
	}
}

func TestPrepareRequestAppliesSupportedAuth(t *testing.T) {
	t.Parallel()

	executor := NewExecutor(nil)
	operation := &model.Operation{Method: "GET", Path: "/me"}
	requirement := &model.SecurityRequirement{
		Alternatives: []model.SecurityAlternative{
			{Schemes: []model.SecurityRequirementRef{{Name: "bearer_auth"}}},
		},
	}
	request, err := executor.PrepareRequest(
		operation,
		&model.RequestDraft{},
		"https://api.example.com",
		requirement,
		map[string]model.SecurityScheme{
			"bearer_auth": {
				Name:   "bearer_auth",
				Type:   model.SecuritySchemeTypeHTTP,
				Scheme: model.HTTPAuthSchemeBearer,
			},
		},
		map[string]model.AuthValue{
			"bearer_auth": {Type: model.AuthSchemeValueTypeBearer, BearerToken: "token-123"},
		},
	)
	if err != nil {
		t.Fatalf("PrepareRequest returned error: %v", err)
	}
	if got := request.Header.Get("Authorization"); got != "Bearer token-123" {
		t.Fatalf("expected bearer auth header, got %q", got)
	}
}

func TestPrepareRequestAppliesBasicAuth(t *testing.T) {
	t.Parallel()

	executor := NewExecutor(nil)
	request, err := executor.PrepareRequest(
		&model.Operation{Method: "GET", Path: "/me"},
		&model.RequestDraft{},
		"https://api.example.com",
		&model.SecurityRequirement{
			Alternatives: []model.SecurityAlternative{
				{Schemes: []model.SecurityRequirementRef{{Name: "basic_auth"}}},
			},
		},
		map[string]model.SecurityScheme{
			"basic_auth": {
				Name:   "basic_auth",
				Type:   model.SecuritySchemeTypeHTTP,
				Scheme: model.HTTPAuthSchemeBasic,
			},
		},
		map[string]model.AuthValue{
			"basic_auth": {Type: model.AuthSchemeValueTypeBasic, Username: "alice", Password: "secret"},
		},
	)
	if err != nil {
		t.Fatalf("PrepareRequest returned error: %v", err)
	}
	if got := request.Header.Get("Authorization"); !strings.HasPrefix(got, "Basic ") {
		t.Fatalf("expected basic auth header, got %q", got)
	}
}

func TestPrepareRequestAppliesQueryAPIKey(t *testing.T) {
	t.Parallel()

	executor := NewExecutor(nil)
	request, err := executor.PrepareRequest(
		&model.Operation{Method: "GET", Path: "/me"},
		&model.RequestDraft{},
		"https://api.example.com",
		&model.SecurityRequirement{
			Alternatives: []model.SecurityAlternative{
				{Schemes: []model.SecurityRequirementRef{{Name: "query_key"}}},
			},
		},
		map[string]model.SecurityScheme{
			"query_key": {
				Name:          "query_key",
				Type:          model.SecuritySchemeTypeAPIKey,
				In:            model.ParameterLocationQuery,
				ParameterName: "api_key",
			},
		},
		map[string]model.AuthValue{
			"query_key": {Type: model.AuthSchemeValueTypeAPIKey, APIKey: "secret"},
		},
	)
	if err != nil {
		t.Fatalf("PrepareRequest returned error: %v", err)
	}
	if got := request.URL.Query().Get("api_key"); got != "secret" {
		t.Fatalf("expected query api key, got %q", got)
	}
}

func TestExecuteCapturesHTTPResponseAndPrettyPrintsJSON(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	executor := NewExecutor(server.Client())
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext returned error: %v", err)
	}

	response := executor.Execute(context.Background(), model.NewOperationKey("GET", "/ping"), request)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", response.StatusCode)
	}
	if response.ContentType != "application/json" {
		t.Fatalf("expected normalised content type, got %q", response.ContentType)
	}
	if !containsAll(response.PrettyBody, []string{"{", "\"ok\": true"}) {
		t.Fatalf("expected pretty body to contain formatted JSON, got %q", response.PrettyBody)
	}
}

func TestExecuteReturnsTransportErrorOnNetworkFailure(t *testing.T) {
	t.Parallel()

	executor := NewExecutor(&http.Client{})
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://127.0.0.1:1", nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext returned error: %v", err)
	}

	response := executor.Execute(context.Background(), model.NewOperationKey("GET", "/fail"), request)
	if response.TransportError == "" {
		t.Fatal("expected transport error to be captured")
	}
}

func containsAll(value string, snippets []string) bool {
	for _, snippet := range snippets {
		if !strings.Contains(value, snippet) {
			return false
		}
	}

	return true
}
