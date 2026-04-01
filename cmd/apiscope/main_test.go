package main

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/phergul/apiscope/internal/app"
	"github.com/phergul/apiscope/internal/logging"
)

type stubRunner struct {
	err   error
	calls int
}

func (r *stubRunner) Run() error {
	r.calls++
	return r.err
}

func TestRunMissingArgumentPrintsUsage(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer

	exitCode := run(nil, strings.NewReader(""), io.Discard, &stderr)

	if exitCode == 0 {
		t.Fatal("expected non-zero exit code when spec source is missing")
	}
	if !strings.Contains(stderr.String(), "usage: apiscope <spec-source>") {
		t.Fatalf("expected usage text, got %q", stderr.String())
	}
}

func TestRunValidArgumentStartsProgram(t *testing.T) {
	previousFactory := newProgram
	previousLoggerFactory := newDiagnosticsLogger
	t.Cleanup(func() {
		newProgram = previousFactory
		newDiagnosticsLogger = previousLoggerFactory
	})
	newDiagnosticsLogger = func() (*slog.Logger, io.Closer, error) {
		return logging.NopLogger(), io.NopCloser(strings.NewReader("")), nil
	}

	var (
		gotService *app.Service
		gotSource  string
		program    = &stubRunner{}
	)
	newProgram = func(service *app.Service, source string, input io.Reader, output io.Writer) runner {
		gotService = service
		gotSource = source
		return program
	}

	exitCode := run([]string{"spec.yaml"}, strings.NewReader(""), io.Discard, io.Discard)

	if exitCode != 0 {
		t.Fatalf("expected zero exit code, got %d", exitCode)
	}
	if gotService == nil {
		t.Fatal("expected service to be created")
	}
	if gotSource != "spec.yaml" {
		t.Fatalf("expected source spec.yaml, got %q", gotSource)
	}
	if program.calls != 1 {
		t.Fatalf("expected runner to be called once, got %d", program.calls)
	}
}

func TestRunLoggerSetupFailureWarnsButContinues(t *testing.T) {
	previousFactory := newProgram
	previousLoggerFactory := newDiagnosticsLogger
	t.Cleanup(func() {
		newProgram = previousFactory
		newDiagnosticsLogger = previousLoggerFactory
	})

	program := &stubRunner{}
	newProgram = func(service *app.Service, source string, input io.Reader, output io.Writer) runner {
		return program
	}
	newDiagnosticsLogger = func() (*slog.Logger, io.Closer, error) {
		return nil, nil, errors.New("permission denied")
	}

	var stderr bytes.Buffer
	exitCode := run([]string{"spec.yaml"}, strings.NewReader(""), io.Discard, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected zero exit code, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "diagnostics logging disabled") {
		t.Fatalf("expected diagnostics logging warning, got %q", stderr.String())
	}
	if program.calls != 1 {
		t.Fatalf("expected runner to be called once, got %d", program.calls)
	}
}
