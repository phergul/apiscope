package logging

import (
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
)

const DefaultFilename = "apiscope.log"

// NewDefaultLogger creates a JSON diagnostics logger backed by the default temp-file path.
func NewDefaultLogger() (*slog.Logger, io.Closer, error) {
	return NewJSONFileLogger(filepath.Join(os.TempDir(), DefaultFilename))
}

// NewJSONFileLogger creates a JSON diagnostics logger that truncates the target file on startup.
func NewJSONFileLogger(path string) (*slog.Logger, io.Closer, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return nil, nil, err
	}

	return slog.New(slog.NewJSONHandler(file, nil)), file, nil
}

// NopLogger returns a logger that discards all output.
func NopLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}

// OrNop returns logger when non-nil, otherwise a no-op logger.
func OrNop(logger *slog.Logger) *slog.Logger {
	if logger != nil {
		return logger
	}

	return NopLogger()
}

// SafeSource preserves local file paths and redacts URL-like source strings.
func SafeSource(raw string) string {
	return SafeURL(raw)
}

// SafeURL removes userinfo, query, and fragment information from URL-like strings.
func SafeURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}

	if parsed.Scheme == "" || parsed.Host == "" {
		return raw
	}

	parsed.User = nil
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

// QueryKeys returns sorted query parameter names for a URL string.
func QueryKeys(raw string) []string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil
	}

	values := parsed.Query()
	if len(values) == 0 {
		return nil
	}

	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// HeaderNames returns sorted header names without any values.
func HeaderNames(header http.Header) []string {
	if len(header) == 0 {
		return nil
	}

	names := make([]string, 0, len(header))
	for name := range header {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
