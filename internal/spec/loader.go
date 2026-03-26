package spec

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"api-tui/internal/model"
)

var (
	errEmptySource             = errors.New("source value is empty")
	errUnknownSourceKind       = errors.New("unknown source kind")
	errUnsupportedSourceScheme = errors.New("unsupported source scheme")
	errMissingURLHost          = errors.New("url host is required")
	errEmptyDocument           = errors.New("document is empty")
	errNormalizationPending    = errors.New("normalization is not implemented yet")
)

type DocumentFormat string

const (
	DocumentFormatJSON DocumentFormat = "json"
	DocumentFormatYAML DocumentFormat = "yaml"
)

type loadedDocument struct {
	Source            Source
	CanonicalLocation string
	Raw               []byte
	Format            DocumentFormat
	MediaType         string
	FinalURL          string
}

type Loader interface {
	Load(ctx context.Context, source Source) (*model.APISpec, error)
}

type loader struct {
	client *http.Client
}

func NewLoader(client *http.Client) Loader {
	return newLoader(client)
}

func newLoader(client *http.Client) *loader {
	if client == nil {
		client = http.DefaultClient
	}

	return &loader{client: client}
}

func (l *loader) Load(ctx context.Context, source Source) (*model.APISpec, error) {
	document, err := l.loadDocument(ctx, source)
	if err != nil {
		return nil, err
	}

	parsed, err := l.parseDocument(document)
	if err != nil {
		return nil, err
	}

	converted, err := l.convertDocument(parsed)
	if err != nil {
		return nil, err
	}

	_, err = l.resolveDocument(ctx, converted)
	resolved, err := l.resolveDocument(ctx, converted)
	if err != nil {
		return nil, err
	}

	return l.normalizeDocument(resolved)
}

func fingerprintForDocument(document *loadedDocument) model.SpecFingerprint {
	sum := sha256.Sum256(document.Raw)
	return model.SpecFingerprint(hex.EncodeToString(sum[:]))
}

func (l *loader) loadDocument(ctx context.Context, source Source) (*loadedDocument, error) {
	classified, err := classifySource(source)
	if err != nil {
		return nil, err
	}

	switch classified.Kind {
	case SourceKindFile:
		return l.loadFile(classified)
	case SourceKindURL:
		return l.loadURL(ctx, classified)
	default:
		return nil, &Error{
			Kind:   ErrorKindInvalidSource,
			Op:     "load document",
			Source: classified.Value,
			Err:    fmt.Errorf("%w: %q", errUnknownSourceKind, classified.Kind),
		}
	}
}

func (l *loader) loadFile(source Source) (*loadedDocument, error) {
	raw, err := os.ReadFile(source.Value)
	if err != nil {
		return nil, &Error{
			Kind:   ErrorKindFileReadFailure,
			Op:     "read file",
			Source: source.Value,
			Err:    err,
		}
	}

	format, err := detectDocumentFormat(source.Value, "", raw)
	if err != nil {
		return nil, wrapDocumentError(err, source.Value, "detect format")
	}

	return &loadedDocument{
		Source:            source,
		CanonicalLocation: source.Value,
		Raw:               raw,
		Format:            format,
	}, nil
}

func (l *loader) loadURL(ctx context.Context, source Source) (*loadedDocument, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source.Value, nil)
	if err != nil {
		return nil, &Error{
			Kind:   ErrorKindInvalidSource,
			Op:     "build request",
			Source: source.Value,
			Err:    err,
		}
	}

	resp, err := l.client.Do(req)
	if err != nil {
		return nil, &Error{
			Kind:   ErrorKindURLFetchFailure,
			Op:     "fetch url",
			Source: source.Value,
			Err:    err,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &Error{
			Kind:       ErrorKindURLFetchFailure,
			Op:         "fetch url",
			Source:     source.Value,
			StatusCode: resp.StatusCode,
			Err:        fmt.Errorf("unexpected response status %s", resp.Status),
		}
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &Error{
			Kind:   ErrorKindURLFetchFailure,
			Op:     "read response body",
			Source: source.Value,
			Err:    err,
		}
	}

	finalURL := source.Value
	if resp.Request != nil && resp.Request.URL != nil {
		finalURL = resp.Request.URL.String()
	}

	mediaType := resp.Header.Get("Content-Type")
	format, err := detectDocumentFormat(finalURL, mediaType, raw)
	if err != nil {
		return nil, wrapDocumentError(err, source.Value, "detect format")
	}

	return &loadedDocument{
		Source:            source,
		CanonicalLocation: finalURL,
		Raw:               raw,
		Format:            format,
		MediaType:         mediaType,
		FinalURL:          finalURL,
	}, nil
}

func detectDocumentFormat(location, contentType string, raw []byte) (DocumentFormat, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return "", &Error{
			Kind:   ErrorKindEmptyDocument,
			Op:     "detect format",
			Source: location,
			Err:    errEmptyDocument,
		}
	}

	if format, ok := formatFromLocation(location); ok {
		return format, nil
	}
	if format, ok := formatFromContentType(contentType); ok {
		return format, nil
	}
	if format, ok := formatFromContent([]byte(trimmed)); ok {
		return format, nil
	}

	return "", &Error{
		Kind:   ErrorKindUnknownFormat,
		Op:     "detect format",
		Source: location,
		Err:    errors.New("could not determine whether document is JSON or YAML"),
	}
}

func wrapDocumentError(err error, source, op string) error {
	var specErr *Error
	if !errors.As(err, &specErr) {
		return err
	}

	if specErr.Source == "" {
		specErr.Source = source
	}
	if specErr.Op == "" {
		specErr.Op = op
	}

	return specErr
}
