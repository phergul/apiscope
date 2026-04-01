package spec

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/phergul/apiscope/internal/logging"
	"github.com/phergul/apiscope/internal/model"
	"github.com/phergul/apiscope/internal/spec/internal/converter"
	"github.com/phergul/apiscope/internal/spec/internal/normalise"
	"github.com/phergul/apiscope/internal/spec/internal/pipeline"
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
	logger *slog.Logger
}

// NewLoader builds a spec loader with the provided HTTP client.
func NewLoader(client *http.Client, logger *slog.Logger) *Loader {
	if client == nil {
		client = http.DefaultClient
	}

	return &Loader{
		client: client,
		logger: logging.OrNop(logger).With("component", "spec"),
	}
}

// Load reads, parses, resolves, and normalises one spec source.
func (l *Loader) Load(ctx context.Context, source Source) (*model.APISpec, error) {
	safeSource := logging.SafeSource(source.Value)
	l.logger.Info("spec load started", "event", "load_start", "source", safeSource)

	document, err := l.loadDocument(ctx, source)
	if err != nil {
		l.logError("load_failed", "spec load failed", err, slog.String("source", safeSource))
		return nil, err
	}

	parsed, err := l.parseDocument(document)
	if err != nil {
		l.logError("parse_failed", "spec parse failed", err, slog.String("source", safeSource))
		return nil, err
	}

	l.logger.Info(
		"swagger conversion started",
		"event", "swagger_convert_start",
		"source", logging.SafeSource(document.CanonicalLocation),
		"source_family", parsed.SourceFamily,
		"source_version", parsed.SourceVersion,
	)
	converted, err := converter.Convert(parsed)
	if err != nil {
		l.logError("swagger_convert_failed", "swagger conversion failed", err,
			slog.String("source", logging.SafeSource(document.CanonicalLocation)),
			slog.String("source_family", string(parsed.SourceFamily)),
			slog.String("source_version", parsed.SourceVersion),
		)
		return nil, err
	}
	l.logger.Info(
		"swagger conversion completed",
		"event", "swagger_convert_complete",
		"source", logging.SafeSource(document.CanonicalLocation),
		"source_family", converted.SourceFamily,
		"source_version", converted.SourceVersion,
	)

	resolved, err := l.resolveDocument(ctx, converted)
	if err != nil {
		l.logError("resolve_failed", "reference resolution failed", err,
			slog.String("source", logging.SafeSource(document.CanonicalLocation)),
			slog.String("source_family", string(converted.SourceFamily)),
			slog.String("source_version", converted.SourceVersion),
		)
		return nil, err
	}

	l.logger.Info(
		"normalisation started",
		"event", "normalise_start",
		"source", logging.SafeSource(document.CanonicalLocation),
		"source_family", resolved.SourceFamily,
		"source_version", resolved.SourceVersion,
	)
	apiSpec, err := normalise.Document(resolved)
	if err != nil {
		l.logError("normalise_failed", "normalisation failed", err,
			slog.String("source", logging.SafeSource(document.CanonicalLocation)),
			slog.String("source_family", string(resolved.SourceFamily)),
			slog.String("source_version", resolved.SourceVersion),
		)
		return nil, err
	}

	l.logger.Info(
		"spec load completed",
		"event", "load_complete",
		"source", logging.SafeSource(document.CanonicalLocation),
		"source_kind", document.Source.Kind,
		"source_family", apiSpec.SourceFamily,
		"source_version", apiSpec.SourceVersion,
		"operation_count", len(apiSpec.Operations),
		"warning_count", len(apiSpec.Warnings),
		"fingerprint", string(apiSpec.Fingerprint),
	)
	return apiSpec, nil
}

// loadDocument loads a raw source document from a file path or URL.
func (l *Loader) loadDocument(ctx context.Context, source Source) (*pipeline.LoadedDocument, error) {
	l.logger.Info("classifying spec source", "event", "source_classify_start", "source", logging.SafeSource(source.Value))
	classified, err := classifySource(source)
	if err != nil {
		l.logError("source_classify_failed", "spec source classification failed", err, slog.String("source", logging.SafeSource(source.Value)))
		return nil, err
	}
	l.logger.Info(
		"spec source classified",
		"event", "source_classify_complete",
		"source", logging.SafeSource(classified.Value),
		"source_kind", classified.Kind,
	)

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

// loadFile loads and classifies a spec document from a local file path.
func (l *Loader) loadFile(source Source) (*pipeline.LoadedDocument, error) {
	absolutePath, err := filepath.Abs(source.Value)
	if err != nil {
		absolutePath = source.Value
	}

	canonicalSource := source
	canonicalSource.Value = absolutePath
	l.logger.Info(
		"loading spec file",
		"event", "file_load_start",
		"source", canonicalSource.Value,
		"source_kind", canonicalSource.Kind,
	)

	raw, err := os.ReadFile(canonicalSource.Value)
	if err != nil {
		l.logError("file_load_failed", "spec file read failed", err, slog.String("source", canonicalSource.Value))
		return nil, &Error{
			Kind:   ErrorKindFileReadFailure,
			Op:     "read file",
			Source: canonicalSource.Value,
			Err:    err,
		}
	}

	l.logger.Info("detecting document format", "event", "format_detect_start", "source", canonicalSource.Value)
	format, err := detectDocumentFormat(canonicalSource.Value, "", raw)
	if err != nil {
		l.logError("format_detect_failed", "document format detection failed", err, slog.String("source", canonicalSource.Value))
		return nil, wrapDocumentError(err, canonicalSource.Value, "detect format")
	}
	l.logger.Info(
		"spec file loaded",
		"event", "file_load_complete",
		"source", canonicalSource.Value,
		"source_kind", canonicalSource.Kind,
		"format", format,
		"bytes", len(raw),
	)

	return &pipeline.LoadedDocument{
		Source:            Source{Kind: canonicalSource.Kind, Value: canonicalSource.Value},
		CanonicalLocation: canonicalSource.Value,
		Raw:               raw,
		Format:            format,
	}, nil
}

// loadURL loads and classifies a spec document from a remote URL.
func (l *Loader) loadURL(ctx context.Context, source Source) (*pipeline.LoadedDocument, error) {
	l.logger.Info(
		"fetching spec url",
		"event", "url_fetch_start",
		"source", logging.SafeSource(source.Value),
		"source_kind", source.Kind,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source.Value, nil)
	if err != nil {
		l.logError("url_request_build_failed", "spec url request build failed", err, slog.String("source", logging.SafeSource(source.Value)))
		return nil, &Error{
			Kind:   ErrorKindInvalidSource,
			Op:     "build request",
			Source: source.Value,
			Err:    err,
		}
	}

	resp, err := l.client.Do(req)
	if err != nil {
		l.logError("url_fetch_failed", "spec url fetch failed", err, slog.String("source", logging.SafeSource(source.Value)))
		return nil, &Error{
			Kind:   ErrorKindURLFetchFailure,
			Op:     "fetch url",
			Source: source.Value,
			Err:    err,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		l.logger.Error(
			"spec url returned non-success status",
			"event", "url_fetch_failed",
			"source", logging.SafeSource(source.Value),
			"status_code", resp.StatusCode,
			"error", fmt.Sprintf("unexpected response status %s", resp.Status),
		)
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
		l.logError("url_read_failed", "spec url response read failed", err, slog.String("source", logging.SafeSource(source.Value)))
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
	l.logger.Info(
		"detecting document format",
		"event", "format_detect_start",
		"source", logging.SafeSource(finalURL),
		"content_type", mediaType,
	)
	format, err := detectDocumentFormat(finalURL, mediaType, raw)
	if err != nil {
		l.logError("format_detect_failed", "document format detection failed", err, slog.String("source", logging.SafeSource(source.Value)))
		return nil, wrapDocumentError(err, source.Value, "detect format")
	}
	l.logger.Info(
		"spec url loaded",
		"event", "url_fetch_complete",
		"source", logging.SafeSource(source.Value),
		"final_url", logging.SafeSource(finalURL),
		"source_kind", source.Kind,
		"status_code", resp.StatusCode,
		"format", format,
		"content_type", mediaType,
		"bytes", len(raw),
	)

	return &pipeline.LoadedDocument{
		Source:            Source{Kind: source.Kind, Value: source.Value},
		CanonicalLocation: finalURL,
		Raw:               raw,
		Format:            format,
		MediaType:         mediaType,
		FinalURL:          finalURL,
	}, nil
}

func (l *Loader) logError(event, msg string, err error, attrs ...slog.Attr) {
	args := make([]any, 0, len(attrs)*2+10)
	args = append(args, "event", event)
	for _, attr := range attrs {
		args = append(args, attr.Key, attr.Value.Any())
	}

	var specErr *Error
	if errors.As(err, &specErr) {
		args = append(args, "error_kind", specErr.Kind)
		if specErr.Op != "" {
			args = append(args, "error_op", specErr.Op)
		}
		if specErr.Source != "" {
			args = append(args, "error_source", logging.SafeSource(specErr.Source))
		}
		if specErr.StatusCode > 0 {
			args = append(args, "status_code", specErr.StatusCode)
		}
	}
	args = append(args, "error", err.Error())
	l.logger.Error(msg, args...)
}

// detectDocumentFormat determines whether a loaded document is JSON or YAML.
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

// wrapDocumentError fills in missing source and operation metadata on spec errors.
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
