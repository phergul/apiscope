package spec

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"

	"github.com/phergul/apiscope/internal/spec/internal/pipeline"

	"github.com/getkin/kin-openapi/openapi3"
)

func (l *Loader) resolveDocument(ctx context.Context, converted *pipeline.ConvertedDocument) (*pipeline.ResolvedDocument, error) {
	baseURI, err := resolveBaseURI(converted.Document)
	if err != nil {
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
		return nil, &Error{
			Kind:   kind,
			Op:     "resolve refs",
			Source: converted.Document.CanonicalLocation,
			Err:    err,
		}
	}

	return &pipeline.ResolvedDocument{
		BaseDocument: pipeline.BaseDocument{
			Document:      converted.Document,
			SourceFamily:  converted.SourceFamily,
			SourceVersion: converted.SourceVersion,
			OpenAPI3Doc:   converted.OpenAPI3Doc,
		},
	}, nil
}

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
