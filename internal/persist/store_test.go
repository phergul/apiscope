package persist

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/phergul/apiscope/internal/model"
)

func TestStoreLoadMissingFilesReturnsEmptyState(t *testing.T) {
	t.Parallel()

	store := NewStore(t.TempDir())

	config, err := store.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}
	if config.ThemeName != "" || len(config.RecentSpecs) != 0 {
		t.Fatalf("expected empty config, got %#v", config)
	}

	environments, err := store.LoadEnvironments()
	if err != nil {
		t.Fatalf("LoadEnvironments returned error: %v", err)
	}
	if len(environments) != 0 {
		t.Fatalf("expected no environments, got %#v", environments)
	}

	history, err := store.LoadHistory()
	if err != nil {
		t.Fatalf("LoadHistory returned error: %v", err)
	}
	if len(history) != 0 {
		t.Fatalf("expected no history buckets, got %#v", history)
	}
}

func TestStoreLoadMalformedJSONReturnsTypedError(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(rootDir, configFileName), []byte("{"), filePermissions); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	_, err := NewStore(rootDir).LoadConfig()
	if err == nil {
		t.Fatal("expected malformed config error")
	}

	var persistErr *Error
	if !errors.As(err, &persistErr) {
		t.Fatalf("expected typed persistence error, got %T", err)
	}
	if persistErr.Op != "decode" {
		t.Fatalf("expected decode failure, got %#v", persistErr)
	}
}

func TestStoreWritesSeparateFilesInOneDirectory(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	store := NewStore(rootDir)
	scopeKey := model.NewPersistenceScopeKey("spec.yaml", model.SourceFamilyOpenAPI3)

	if err := store.SaveConfig(model.UserConfig{
		ThemeName: "harbor",
		RecentSpecs: []model.RecentSpec{{
			Source: "spec.yaml",
			Title:  "Demo API",
		}},
	}); err != nil {
		t.Fatalf("SaveConfig returned error: %v", err)
	}
	if err := store.SaveEnvironments([]model.SavedEnvironment{{
		Name:              "staging",
		ScopeKey:          scopeKey,
		SelectedServerURL: "https://staging.example.com",
		AuthBindings: map[string]model.SavedAuthBinding{
			"api_key": {
				FieldEnvVars: map[model.AuthField]string{
					model.AuthFieldAPIKey: "API_KEY_ENV",
				},
			},
		},
	}}); err != nil {
		t.Fatalf("SaveEnvironments returned error: %v", err)
	}
	if err := store.SaveHistory([]model.PersistedHistoryBucket{{
		ScopeKey:     scopeKey,
		OperationKey: model.NewOperationKey("GET", "/pets"),
		Entries: []model.HistoryEntry{{
			RequestID:    7,
			OperationKey: model.NewOperationKey("GET", "/pets"),
			Request: model.ExecutedRequestSnapshot{
				AuthState: map[string]model.AuthValue{
					"api_key": {Type: model.AuthSchemeValueTypeAPIKey, APIKey: "secret"},
				},
			},
		}},
	}}); err != nil {
		t.Fatalf("SaveHistory returned error: %v", err)
	}

	for _, fileName := range []string{configFileName, environmentsFileName, historyFileName} {
		if _, err := os.Stat(filepath.Join(rootDir, fileName)); err != nil {
			t.Fatalf("expected %s to exist: %v", fileName, err)
		}
	}

	config, err := store.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}
	if config.ThemeName != "harbor" {
		t.Fatalf("expected theme harbor, got %#v", config)
	}

	environments, err := store.LoadEnvironments()
	if err != nil {
		t.Fatalf("LoadEnvironments returned error: %v", err)
	}
	if len(environments) != 1 || environments[0].Name != "staging" {
		t.Fatalf("expected one saved environment, got %#v", environments)
	}
	if got := environments[0].AuthBindings["api_key"].FieldEnvVars[model.AuthFieldAPIKey]; got != "API_KEY_ENV" {
		t.Fatalf("expected auth binding round-trip, got %#v", environments[0].AuthBindings)
	}

	history, err := store.LoadHistory()
	if err != nil {
		t.Fatalf("LoadHistory returned error: %v", err)
	}
	if len(history) != 1 || len(history[0].Entries) != 1 || history[0].Entries[0].RequestID != 7 {
		t.Fatalf("expected one history entry, got %#v", history)
	}
	if history[0].Entries[0].Request.AuthState != nil {
		t.Fatalf("expected history auth snapshot to stay out of durable storage, got %#v", history[0].Entries[0].Request.AuthState)
	}
}
