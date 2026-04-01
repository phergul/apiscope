package transport

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/phergul/apiscope/internal/model"
)

func TestExecuteLogsStartAndCompletion(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	executor := NewExecutor(server.Client(), logger)
	request, err := http.NewRequest(http.MethodGet, server.URL+"/pets?token=secret", nil)
	if err != nil {
		t.Fatalf("NewRequest returned error: %v", err)
	}
	request.Header.Set("Authorization", "Bearer secret")

	response := executor.Execute(context.Background(), model.NewOperationKey("GET", "/pets"), request)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}
	if !strings.Contains(buf.String(), `"event":"execute_start"`) || !strings.Contains(buf.String(), `"event":"execute_complete"`) {
		t.Fatalf("expected execute start and complete logs, got %s", buf.String())
	}
	if strings.Contains(buf.String(), "token=secret") || strings.Contains(buf.String(), "Bearer secret") {
		t.Fatalf("expected logs to redact URL query and header values, got %s", buf.String())
	}
}

func TestExecuteLogsNetworkFailure(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	executor := NewExecutor(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("dial tcp: connection refused")
		}),
	}, logger)
	request, err := http.NewRequest(http.MethodGet, "https://api.example.com/pets", nil)
	if err != nil {
		t.Fatalf("NewRequest returned error: %v", err)
	}

	response := executor.Execute(context.Background(), model.NewOperationKey("GET", "/pets"), request)
	if response.TransportError == "" {
		t.Fatal("expected transport error")
	}
	if !strings.Contains(buf.String(), `"event":"execute_failed"`) {
		t.Fatalf("expected execute failure log, got %s", buf.String())
	}
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
