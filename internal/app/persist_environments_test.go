package app

import (
	"testing"
	"time"

	"github.com/phergul/apiscope/internal/model"
	"github.com/phergul/apiscope/internal/persist"
)

func TestServiceApplyEnvironmentResolvesEnvVarBindings(t *testing.T) {
	service := NewService(nil, nil, nil, nil)
	service.lookupEnv = func(name string) (string, bool) {
		switch name {
		case "API_KEY_ENV":
			return "secret", true
		case "BASIC_USER_ENV":
			return "alice", true
		default:
			return "", false
		}
	}

	session := model.SessionState{
		SelectedServerURL: "https://api.example.com",
		Spec: &model.APISpec{
			SecuritySchemes: map[string]model.SecurityScheme{
				"api_key": {
					Name:          "api_key",
					Type:          model.SecuritySchemeTypeAPIKey,
					In:            model.ParameterLocationHeader,
					ParameterName: "X-API-Key",
				},
				"basic_auth": {
					Name:   "basic_auth",
					Type:   model.SecuritySchemeTypeHTTP,
					Scheme: model.HTTPAuthSchemeBasic,
				},
			},
		},
		AuthState: map[string]model.AuthValue{
			"legacy": {Type: model.AuthSchemeValueTypeBearer, BearerToken: "session-only"},
		},
	}

	result := service.ApplyEnvironment(&session, model.SavedEnvironment{
		Name:              "staging",
		SelectedServerURL: "https://staging.example.com",
		AuthBindings: map[string]model.SavedAuthBinding{
			"api_key": {
				FieldEnvVars: map[model.AuthField]string{
					model.AuthFieldAPIKey: "API_KEY_ENV",
				},
			},
			"basic_auth": {
				FieldEnvVars: map[model.AuthField]string{
					model.AuthFieldUsername: "BASIC_USER_ENV",
					model.AuthFieldPassword: "BASIC_PASSWORD_ENV",
				},
			},
		},
	})

	if !result.Changed {
		t.Fatal("expected environment apply to report a change")
	}
	if got := session.SelectedServerURL; got != "https://staging.example.com" {
		t.Fatalf("expected selected server to update, got %q", got)
	}
	if len(result.MissingEnvVars) != 1 || result.MissingEnvVars[0] != "BASIC_PASSWORD_ENV" {
		t.Fatalf("expected missing password env warning, got %#v", result.MissingEnvVars)
	}
	if got := session.AuthState["api_key"].APIKey; got != "secret" {
		t.Fatalf("expected resolved API key auth, got %#v", session.AuthState)
	}
	if got := session.AuthState["basic_auth"].Username; got != "alice" {
		t.Fatalf("expected resolved basic username, got %#v", session.AuthState)
	}
	if _, ok := session.AuthState["legacy"]; ok {
		t.Fatalf("expected apply to rebuild auth state from resolved bindings only, got %#v", session.AuthState)
	}
}

func TestServiceSaveEnvironmentPreservesExistingBindings(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	store := persist.NewStore(rootDir)
	scopeKey := model.NewPersistenceScopeKey("spec.yaml", model.SourceFamilyOpenAPI3)
	createdAt := time.Date(2026, time.April, 2, 12, 0, 0, 0, time.UTC)
	if err := store.SaveEnvironments([]model.SavedEnvironment{{
		Name:              "staging",
		ScopeKey:          scopeKey,
		SelectedServerURL: "https://old.example.com",
		AuthBindings: map[string]model.SavedAuthBinding{
			"api_key": {
				FieldEnvVars: map[model.AuthField]string{
					model.AuthFieldAPIKey: "API_KEY_ENV",
				},
			},
		},
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}}); err != nil {
		t.Fatalf("SaveEnvironments returned error: %v", err)
	}

	service := NewService(nil, nil, store, nil)
	now := time.Date(2026, time.April, 3, 9, 30, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	environments, err := service.SaveEnvironment(model.SessionState{
		PersistenceScopeKey: scopeKey,
		SelectedServerURL:   "https://new.example.com",
	}, "staging")
	if err != nil {
		t.Fatalf("SaveEnvironment returned error: %v", err)
	}

	if len(environments) != 1 {
		t.Fatalf("expected one saved environment, got %#v", environments)
	}
	if environments[0].AuthBindings["api_key"].FieldEnvVars[model.AuthFieldAPIKey] != "API_KEY_ENV" {
		t.Fatalf("expected existing binding to be preserved, got %#v", environments[0].AuthBindings)
	}
	if !environments[0].CreatedAt.Equal(createdAt) {
		t.Fatalf("expected created timestamp to stay unchanged, got %v", environments[0].CreatedAt)
	}
	if !environments[0].UpdatedAt.Equal(now) {
		t.Fatalf("expected updated timestamp to refresh, got %v", environments[0].UpdatedAt)
	}
}
