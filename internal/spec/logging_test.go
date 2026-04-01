package spec

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"testing"
)

func TestLoadLogsSuccessfulPipeline(t *testing.T) {
	t.Parallel()

	path := writeTempSpecFile(t, "spec.yaml", "openapi: 3.0.3\ninfo:\n  title: Demo\n  version: 1.0.0\npaths:\n  /pets:\n    get:\n      responses:\n        \"200\":\n          description: ok\n")
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	apiSpec, err := NewLoader(nil, logger).Load(context.Background(), Source{Value: path})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if apiSpec == nil {
		t.Fatal("expected spec result")
	}

	events := parseLogEvents(t, buf.String())
	if !containsEvent(events, "load_start") || !containsEvent(events, "load_complete") {
		t.Fatalf("expected load start and complete events, got %#v", events)
	}
	if !containsEvent(events, "resolve_complete") || !containsEvent(events, "normalise_start") {
		t.Fatalf("expected pipeline events, got %#v", events)
	}
}

func TestLoadLogsUnsupportedSwaggerFailure(t *testing.T) {
	t.Parallel()

	path := writeTempSpecFile(t, "swagger.yaml", `swagger: "2.0"
info:
  title: Demo
  version: 1.0.0
paths:
  /upload:
    post:
      parameters:
        - name: file
          in: formData
          type: file
      responses:
        "200":
          description: ok
`)
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	_, err := NewLoader(nil, logger).Load(context.Background(), Source{Value: path})
	if !IsErrorKind(err, ErrorKindUnsupportedSwaggerConstruct) {
		t.Fatalf("expected unsupported swagger construct error, got %v", err)
	}
	if !strings.Contains(buf.String(), `"event":"swagger_convert_failed"`) {
		t.Fatalf("expected swagger convert failure event, got %s", buf.String())
	}
	if !strings.Contains(buf.String(), `"error_kind":"unsupported_swagger_construct"`) {
		t.Fatalf("expected error kind in logs, got %s", buf.String())
	}
}

func TestLoadDocumentFromURLLogsRedactedSource(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	client := newTestClient(func(req *http.Request) (*http.Response, error) {
		return stringResponse(req, http.StatusOK, "application/json", `{"openapi":"3.0.3"}`), nil
	})

	_, err := NewLoader(client, logger).loadDocument(context.Background(), Source{Value: "https://user:secret@example.com/spec?token=abc"})
	if err != nil {
		t.Fatalf("loadDocument returned error: %v", err)
	}

	if strings.Contains(buf.String(), "secret@example.com/spec?token=abc") {
		t.Fatalf("expected redacted source in logs, got %s", buf.String())
	}
	if !strings.Contains(buf.String(), `"source":"https://example.com/spec"`) {
		t.Fatalf("expected safe source in logs, got %s", buf.String())
	}
}

func parseLogEvents(t *testing.T, logs string) []string {
	t.Helper()

	lines := strings.Split(strings.TrimSpace(logs), "\n")
	events := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("json.Unmarshal returned error: %v", err)
		}
		if event, ok := record["event"].(string); ok {
			events = append(events, event)
		}
	}
	return events
}

func containsEvent(events []string, want string) bool {
	for _, event := range events {
		if event == want {
			return true
		}
	}
	return false
}
