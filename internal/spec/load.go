package spec

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"api-tui/internal/model"
	"api-tui/internal/spec/internal/converter"
	"api-tui/internal/spec/internal/normalise"
	"api-tui/internal/spec/internal/pipeline"
)

var (
	errEmptySource             = errors.New("source value is empty")
	errUnknownSourceKind       = errors.New("unknown source kind")
	errUnsupportedSourceScheme = errors.New("unsupported source scheme")
	errMissingURLHost          = errors.New("url host is required")
	errEmptyDocument           = errors.New("document is empty")
)

type Loader struct {
	client *http.Client
}

func NewLoader(client *http.Client) *Loader {
	if client == nil {
		client = http.DefaultClient
	}

	return &Loader{client: client}
}

func (l *Loader) Load(ctx context.Context, source Source) (*model.APISpec, error) {
	document, err := l.loadDocument(ctx, source)
	if err != nil {
		return nil, err
	}

	parsed, err := l.parseDocument(document)
	if err != nil {
		return nil, err
	}

	converted, err := converter.Convert(parsed)
	if err != nil {
		return nil, err
	}

	resolved, err := l.resolveDocument(ctx, converted)
	if err != nil {
		return nil, err
	}

	return normalise.Document(resolved)
}

func (l *Loader) loadDocument(ctx context.Context, source Source) (*pipeline.LoadedDocument, error) {
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

func (l *Loader) loadFile(source Source) (*pipeline.LoadedDocument, error) {
	absolutePath, err := filepath.Abs(source.Value)
	if err != nil {
		absolutePath = source.Value
	}

	canonicalSource := source
	canonicalSource.Value = absolutePath

	raw, err := os.ReadFile(canonicalSource.Value)
	if err != nil {
		return nil, &Error{
			Kind:   ErrorKindFileReadFailure,
			Op:     "read file",
			Source: canonicalSource.Value,
			Err:    err,
		}
	}

	format, err := detectDocumentFormat(canonicalSource.Value, "", raw)
	if err != nil {
		return nil, wrapDocumentError(err, canonicalSource.Value, "detect format")
	}

	return &pipeline.LoadedDocument{
		Source:            Source{Kind: canonicalSource.Kind, Value: canonicalSource.Value},
		CanonicalLocation: canonicalSource.Value,
		Raw:               raw,
		Format:            format,
	}, nil
}

func (l *Loader) loadURL(ctx context.Context, source Source) (*pipeline.LoadedDocument, error) {
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

	return &pipeline.LoadedDocument{
		Source:            Source{Kind: source.Kind, Value: source.Value},
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
