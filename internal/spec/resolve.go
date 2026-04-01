package spec

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"path/filepath"

	"github.com/phergul/apiscope/internal/logging"
	"github.com/phergul/apiscope/internal/spec/internal/pipeline"

	"github.com/getkin/kin-openapi/openapi3"
)

// resolveDocument resolves internal and external references in the converted OpenAPI document.
func (l *Loader) resolveDocument(ctx context.Context, converted *pipeline.ConvertedDocument) (*pipeline.ResolvedDocument, error) {
	source := logging.SafeSource(converted.Document.CanonicalLocation)
	l.logger.Info(
		"resolving references",
		"event", "resolve_start",
		"source", source,
		"source_family", converted.SourceFamily,
		"source_version", converted.SourceVersion,
	)
	baseURI, err := resolveBaseURI(converted.Document)
	if err != nil {
		l.logError("resolve_failed", "reference resolution failed", err, slog.String("source", source))
		return nil, &Error{
			Kind:   ErrorKindRefResolutionFailure,
			Op:     "resolve refs",
			Source: converted.Document.CanonicalLocation,
			Err:    err,
		}
	}

	refLoader := openapi3.NewLoader()
	refLoader.IsExternalRefsAllowed = true
	refLoader.Context = ctx
	refLoader.ReadFromURIFunc = newRefReadFromURIFunc(l.client)

	if err := refLoader.ResolveRefsIn(converted.OpenAPI3Doc, baseURI); err != nil {
		kind := ErrorKindRefResolutionFailure
		if errors.Is(err, openapi3.ErrURINotSupported) {
			kind = ErrorKindUnsupportedExternalRef
		}
		l.logger.Error(
			"reference resolution failed",
			"event", "resolve_failed",
			"source", source,
			"source_family", converted.SourceFamily,
			"source_version", converted.SourceVersion,
			"error_kind", kind,
			"error", err.Error(),
		)
		return nil, &Error{
			Kind:   kind,
			Op:     "resolve refs",
			Source: converted.Document.CanonicalLocation,
			Err:    err,
		}
	}
	l.logger.Info(
		"references resolved",
		"event", "resolve_complete",
		"source", source,
		"source_family", converted.SourceFamily,
		"source_version", converted.SourceVersion,
	)

	return &pipeline.ResolvedDocument{
		BaseDocument: pipeline.BaseDocument{
			Document:      converted.Document,
			SourceFamily:  converted.SourceFamily,
			SourceVersion: converted.SourceVersion,
			OpenAPI3Doc:   converted.OpenAPI3Doc,
		},
	}, nil
}

// resolveBaseURI returns the base URI used for relative reference resolution.
func resolveBaseURI(document *pipeline.LoadedDocument) (*url.URL, error) {
	if document == nil {
		return nil, errors.New("document is required")
	}

	if document.Source.Kind == pipeline.SourceKindURL || document.FinalURL != "" {
		location := document.FinalURL
		if location == "" {
			location = document.CanonicalLocation
		}
		parsed, err := url.Parse(location)
		if err != nil {
			return nil, fmt.Errorf("parse ref base uri: %w", err)
		}
		return parsed, nil
	}

	location := document.CanonicalLocation
	if !filepath.IsAbs(location) {
		absolutePath, err := filepath.Abs(location)
		if err != nil {
			return nil, fmt.Errorf("resolve absolute ref base path: %w", err)
		}
		location = absolutePath
	}

	return &url.URL{
		Scheme: "file",
		Path:   filepath.ToSlash(location),
	}, nil
}

// newRefReadFromURIFunc builds the external reference reader used by kin-openapi.
func newRefReadFromURIFunc(client *http.Client) openapi3.ReadFromURIFunc {
	reader := openapi3.ReadFromURIs(
		openapi3.ReadFromHTTP(client),
		openapi3.ReadFromFile,
	)

	return openapi3.URIMapCache(func(loader *openapi3.Loader, location *url.URL) ([]byte, error) {
		switch location.Scheme {
		case "", "file", "http", "https":
			return reader(loader, location)
		default:
			return nil, openapi3.ErrURINotSupported
		}
	})
}
