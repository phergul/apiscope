package logging

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestNewJSONFileLoggerTruncatesExistingFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "apiscope.log")
	if err := os.WriteFile(path, []byte("old-data"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	logger, closer, err := NewJSONFileLogger(path)
	if err != nil {
		t.Fatalf("NewJSONFileLogger returned error: %v", err)
	}
	logger.Info("hello", "component", "test", "event", "write")
	if err := closer.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if bytes.Contains(data, []byte("old-data")) {
		t.Fatalf("expected existing contents to be truncated, got %q", string(data))
	}
}

func TestJSONLoggerWritesValidRecords(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "apiscope.log")
	logger, closer, err := NewJSONFileLogger(path)
	if err != nil {
		t.Fatalf("NewJSONFileLogger returned error: %v", err)
	}
	logger.Info("hello", "component", "test", "event", "write")
	if err := closer.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}

	var record map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(data), &record); err != nil {
		t.Fatalf("expected valid json log record, got %v", err)
	}
	if record["component"] != "test" {
		t.Fatalf("expected component field, got %#v", record)
	}
}

func TestSafeURLRedactsSensitiveParts(t *testing.T) {
	t.Parallel()

	got := SafeURL("https://user:secret@example.com/pets?token=abc#frag")
	if got != "https://example.com/pets" {
		t.Fatalf("expected redacted url, got %q", got)
	}
}

func TestHeaderNamesReturnsSortedNamesOnly(t *testing.T) {
	t.Parallel()

	names := HeaderNames(http.Header{
		"Authorization": []string{"Bearer secret"},
		"X-Trace-ID":    []string{"trace-1"},
	})
	if len(names) != 2 || names[0] != "Authorization" || names[1] != "X-Trace-ID" {
		t.Fatalf("unexpected header names %#v", names)
	}
}

func TestNopLoggerAcceptsWrites(t *testing.T) {
	t.Parallel()

	NopLogger().Info("discarded", "component", "test")
}
