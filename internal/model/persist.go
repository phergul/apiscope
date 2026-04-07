package model

import (
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// PersistenceScopeKey identifies one durable source scope for persisted data.
//
// Unlike SpecFingerprint, it is intentionally coarse so saved environments and
// history survive routine spec edits for the same source.
type PersistenceScopeKey string

// NewPersistenceScopeKey derives a coarse persisted identity for one spec source.
func NewPersistenceScopeKey(rawSource string, family SourceFamily) PersistenceScopeKey {
	if family == "" {
		family = SourceFamilyUnknown
	}

	sum := sha256.Sum256([]byte(string(family) + "\n" + normalisePersistenceSource(rawSource)))
	return PersistenceScopeKey(hex.EncodeToString(sum[:]))
}

// UserConfig stores durable user preferences and recently opened specs.
type UserConfig struct {
	ThemeName   string       `json:"theme_name,omitempty"`
	RecentSpecs []RecentSpec `json:"recent_specs,omitempty"`
}

// RecentSpec stores one recently opened spec source.
type RecentSpec struct {
	Source       string       `json:"source"`
	Title        string       `json:"title,omitempty"`
	LastOpenedAt time.Time    `json:"last_opened_at"`
	SourceFamily SourceFamily `json:"source_family,omitempty"`
}

// SavedAuthBinding stores durable env-var references for one auth scheme.
type SavedAuthBinding struct {
	FieldEnvVars map[AuthField]string `json:"field_env_vars,omitempty"`
}

// SavedEnvironment stores one named server and env-var auth binding set for a source scope.
type SavedEnvironment struct {
	Name              string                      `json:"name"`
	ScopeKey          PersistenceScopeKey         `json:"scope_key"`
	SelectedServerURL string                      `json:"selected_server_url,omitempty"`
	AuthBindings      map[string]SavedAuthBinding `json:"auth_bindings,omitempty"`
	CreatedAt         time.Time                   `json:"created_at"`
	UpdatedAt         time.Time                   `json:"updated_at"`
}

// PersistedHistoryBucket stores durable history for one source scope and operation.
type PersistedHistoryBucket struct {
	ScopeKey     PersistenceScopeKey `json:"scope_key"`
	OperationKey OperationKey        `json:"operation_key"`
	Entries      []HistoryEntry      `json:"entries,omitempty"`
}

func normalisePersistenceSource(rawSource string) string {
	rawSource = strings.TrimSpace(rawSource)
	if rawSource == "" {
		return ""
	}

	if parsed, err := url.Parse(rawSource); err == nil && parsed.Scheme != "" && strings.TrimSpace(parsed.Host) != "" {
		parsed.Scheme = strings.ToLower(parsed.Scheme)
		parsed.Host = strings.ToLower(parsed.Host)
		if strings.TrimSpace(parsed.Path) == "" {
			parsed.Path = "/"
		} else {
			parsed.Path = path.Clean(parsed.Path)
			if !strings.HasPrefix(parsed.Path, "/") {
				parsed.Path = "/" + parsed.Path
			}
		}
		return parsed.String()
	}

	return filepath.Clean(rawSource)
}
