package spec

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadDocumentFromFileJSON(t *testing.T) {
	t.Parallel()

	path := writeTempSpecFile(t, "spec.json", `{"openapi":"3.0.3"}`)

	doc, err := newLoader(nil).loadDocument(context.Background(), Source{Value: path})
	if err != nil {
		t.Fatalf("loadDocument returned error: %v", err)
	}

	if doc.Format != DocumentFormatJSON {
		t.Fatalf("expected json format, got %q", doc.Format)
	}
	if doc.CanonicalLocation != path {
		t.Fatalf("expected canonical location %q, got %q", path, doc.CanonicalLocation)
	}
}

func TestLoadDocumentFromFileYAML(t *testing.T) {
	t.Parallel()

	path := writeTempSpecFile(t, "spec.yaml", "openapi: 3.0.3\ninfo:\n  title: Demo\n")

	doc, err := newLoader(nil).loadDocument(context.Background(), Source{Value: path})
	if err != nil {
		t.Fatalf("loadDocument returned error: %v", err)
	}

	if doc.Format != DocumentFormatYAML {
		t.Fatalf("expected yaml format, got %q", doc.Format)
	}
}

func TestLoadDocumentCanonicalizesRelativeFilePath(t *testing.T) {
	t.Parallel()

	path := writeTempSpecFile(t, "spec.yaml", "openapi: 3.0.3\ninfo:\n  title: Demo\n")
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}
	relativePath, err := filepath.Rel(cwd, path)
	if err != nil {
		t.Fatalf("filepath.Rel: %v", err)
	}

	doc, err := newLoader(nil).loadDocument(context.Background(), Source{Value: relativePath})
	if err != nil {
		t.Fatalf("loadDocument returned error: %v", err)
	}

	if !filepath.IsAbs(doc.CanonicalLocation) {
		t.Fatalf("expected canonical location to be absolute, got %q", doc.CanonicalLocation)
	}
}

func TestLoadDocumentRejectsMissingFile(t *testing.T) {
	t.Parallel()

	_, err := newLoader(nil).loadDocument(context.Background(), Source{Value: filepath.Join(t.TempDir(), "missing.yaml")})
	if !IsErrorKind(err, ErrorKindFileReadFailure) {
		t.Fatalf("expected file read failure, got %v", err)
	}
}

func TestLoadDocumentRejectsEmptyFile(t *testing.T) {
	t.Parallel()

	path := writeTempSpecFile(t, "empty.yaml", "")

	_, err := newLoader(nil).loadDocument(context.Background(), Source{Value: path})
	if !IsErrorKind(err, ErrorKindEmptyDocument) {
		t.Fatalf("expected empty document error, got %v", err)
	}
}

func TestLoadDocumentRejectsUnsupportedScheme(t *testing.T) {
	t.Parallel()

	_, err := newLoader(nil).loadDocument(context.Background(), Source{Value: "ftp://example.com/openapi.yaml"})
	if !IsErrorKind(err, ErrorKindUnsupportedScheme) {
		t.Fatalf("expected unsupported scheme error, got %v", err)
	}
}

func TestLoadDocumentFromURLJSON(t *testing.T) {
	t.Parallel()

	client := newTestClient(func(req *http.Request) (*http.Response, error) {
		return stringResponse(req, http.StatusOK, "application/json", `{"openapi":"3.0.3"}`), nil
	})

	doc, err := newLoader(client).loadDocument(context.Background(), Source{Value: "https://example.com/spec"})
	if err != nil {
		t.Fatalf("loadDocument returned error: %v", err)
	}

	if doc.Format != DocumentFormatJSON {
		t.Fatalf("expected json format, got %q", doc.Format)
	}
	if doc.FinalURL != "https://example.com/spec" {
		t.Fatalf("expected final url %q, got %q", "https://example.com/spec", doc.FinalURL)
	}
	if doc.MediaType != "application/json" {
		t.Fatalf("expected media type to be preserved, got %q", doc.MediaType)
	}
}

func TestLoadDocumentFromURLYAMLByContentType(t *testing.T) {
	t.Parallel()

	client := newTestClient(func(req *http.Request) (*http.Response, error) {
		return stringResponse(req, http.StatusOK, "application/yaml", "openapi: 3.0.3\n"), nil
	})

	doc, err := newLoader(client).loadDocument(context.Background(), Source{Value: "https://example.com/spec"})
	if err != nil {
		t.Fatalf("loadDocument returned error: %v", err)
	}

	if doc.Format != DocumentFormatYAML {
		t.Fatalf("expected yaml format, got %q", doc.Format)
	}
}

func TestLoadDocumentPreservesFinalURLAfterRedirect(t *testing.T) {
	t.Parallel()

	client := newTestClient(func(req *http.Request) (*http.Response, error) {
		resp := stringResponse(req, http.StatusOK, "text/plain", "openapi: 3.0.3\n")
		resp.Request = clonedRequest(req, "https://example.com/final.yaml")
		return resp, nil
	})

	doc, err := newLoader(client).loadDocument(context.Background(), Source{Value: "https://example.com/start"})
	if err != nil {
		t.Fatalf("loadDocument returned error: %v", err)
	}

	if doc.FinalURL != "https://example.com/final.yaml" {
		t.Fatalf("expected redirected final url, got %q", doc.FinalURL)
	}
	if doc.Format != DocumentFormatYAML {
		t.Fatalf("expected yaml format, got %q", doc.Format)
	}
}

func TestLoadDocumentRejectsNon2xxResponses(t *testing.T) {
	t.Parallel()

	client := newTestClient(func(req *http.Request) (*http.Response, error) {
		return stringResponse(req, http.StatusBadGateway, "text/plain", "boom"), nil
	})

	_, err := newLoader(client).loadDocument(context.Background(), Source{Value: "https://example.com/spec"})
	if !IsErrorKind(err, ErrorKindURLFetchFailure) {
		t.Fatalf("expected url fetch failure, got %v", err)
	}

	var specErr *Error
	if !errors.As(err, &specErr) {
		t.Fatalf("expected spec error, got %T", err)
	}
	if specErr.StatusCode != http.StatusBadGateway {
		t.Fatalf("expected status code %d, got %d", http.StatusBadGateway, specErr.StatusCode)
	}
}

func TestLoadDocumentHandlesNetworkFailure(t *testing.T) {
	t.Parallel()

	client := newTestClient(func(req *http.Request) (*http.Response, error) {
		return nil, errors.New("dial tcp: connection refused")
	})

	_, err := newLoader(client).loadDocument(context.Background(), Source{Value: "https://example.com/spec"})
	if !IsErrorKind(err, ErrorKindURLFetchFailure) {
		t.Fatalf("expected url fetch failure, got %v", err)
	}
}

func TestDetectDocumentFormatByExtension(t *testing.T) {
	t.Parallel()

	format, err := detectDocumentFormat("spec.yaml", "", []byte(`{"openapi":"3.0.3"}`))
	if err != nil {
		t.Fatalf("detectDocumentFormat returned error: %v", err)
	}
	if format != DocumentFormatYAML {
		t.Fatalf("expected yaml from extension, got %q", format)
	}
}

func TestDetectDocumentFormatByContentType(t *testing.T) {
	t.Parallel()

	format, err := detectDocumentFormat("spec", "application/json; charset=utf-8", []byte("openapi: 3.0.3\n"))
	if err != nil {
		t.Fatalf("detectDocumentFormat returned error: %v", err)
	}
	if format != DocumentFormatJSON {
		t.Fatalf("expected json from content type, got %q", format)
	}
}

func TestDetectDocumentFormatByContentSniffing(t *testing.T) {
	t.Parallel()

	format, err := detectDocumentFormat("spec", "", []byte("openapi: 3.0.3\ninfo:\n  title: Demo\n"))
	if err != nil {
		t.Fatalf("detectDocumentFormat returned error: %v", err)
	}
	if format != DocumentFormatYAML {
		t.Fatalf("expected yaml from content sniffing, got %q", format)
	}
}

func TestDetectDocumentFormatRejectsUnknownContent(t *testing.T) {
	t.Parallel()

	_, err := detectDocumentFormat("spec", "", []byte("this definitely is not an api description"))
	if !IsErrorKind(err, ErrorKindUnknownFormat) {
		t.Fatalf("expected unknown format error, got %v", err)
	}
}

func writeTempSpecFile(t *testing.T, name, contents string) string {
	t.Helper()

	dir := t.TempDir()
	return writeTempSpecFileInDir(t, dir, name, contents)
}

func writeTempSpecFileInDir(t *testing.T, dir, name, contents string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	return path
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newTestClient(fn roundTripFunc) *http.Client {
	return &http.Client{Transport: fn}
}

func stringResponse(req *http.Request, statusCode int, contentType, body string) *http.Response {
	resp := &http.Response{
		StatusCode: statusCode,
		Status:     fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode)),
		Header:     make(http.Header),
		Body:       http.NoBody,
		Request:    req,
	}
	if contentType != "" {
		resp.Header.Set("Content-Type", contentType)
	}
	if body != "" {
		resp.Body = ioNopCloser(body)
	}

	return resp
}

func clonedRequest(req *http.Request, rawURL string) *http.Request {
	clone := req.Clone(req.Context())
	parsed, err := url.Parse(rawURL)
	if err == nil {
		clone.URL = parsed
	}

	return clone
}

func ioNopCloser(body string) io.ReadCloser {
	return io.NopCloser(strings.NewReader(body))
}
