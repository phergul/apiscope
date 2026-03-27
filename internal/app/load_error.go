package app

import (
	"errors"
	"fmt"

	"api-tui/internal/spec"
)

type LoadErrorView struct {
	Category string
	Title    string
	Summary  string
	Source   string
	Hint     string
}

func DescribeLoadError(err error, fallbackSource string) LoadErrorView {
	view := LoadErrorView{
		Category: "load error",
		Title:    "Failed to load spec",
		Summary:  "The spec could not be loaded.",
		Source:   fallbackSource,
		Hint:     "Check the source and try again.",
	}

	var specErr *spec.Error
	if !errors.As(err, &specErr) {
		if err != nil {
			view.Summary = err.Error()
		}
		return view
	}

	if specErr.Source != "" {
		view.Source = specErr.Source
	}

	switch specErr.Kind {
	case spec.ErrorKindInvalidSource, spec.ErrorKindUnsupportedScheme:
		view.Category = "invalid source"
		view.Title = "Invalid spec source"
		view.Summary = "The source could not be interpreted as a readable file path or supported URL."
		view.Hint = "Use a local file path or an http/https URL."
	case spec.ErrorKindFileReadFailure:
		view.Category = "file read failure"
		view.Title = "Couldn't read spec file"
		view.Summary = "The spec file could not be opened or read."
		view.Hint = "Check that the file exists and that you have permission to read it."
	case spec.ErrorKindURLFetchFailure:
		view.Category = "url fetch failure"
		view.Title = "Couldn't fetch spec URL"
		view.Summary = "The remote spec could not be downloaded."
		if specErr.StatusCode > 0 {
			view.Summary = fmt.Sprintf("The remote spec request failed with HTTP status %d.", specErr.StatusCode)
		}
		view.Hint = "Check the URL, network connection, and any required access."
	case spec.ErrorKindUnknownFormat, spec.ErrorKindEmptyDocument, spec.ErrorKindDecodeFailure,
		spec.ErrorKindUnsupportedFamily, spec.ErrorKindUnsupportedVersion, spec.ErrorKindOpenAPIParseFailure:
		view.Category = "parse failure"
		view.Title = "Spec could not be parsed"
		view.Summary = "The document could not be understood as a supported Swagger or OpenAPI spec."
		view.Hint = "Validate the file contents and make sure the document is valid JSON or YAML."
	case spec.ErrorKindRefResolutionFailure, spec.ErrorKindUnsupportedExternalRef,
		spec.ErrorKindNormalizationFailure, spec.ErrorKindUnsupportedNormalizedConstruct,
		spec.ErrorKindUnsupportedSwaggerConstruct, spec.ErrorKindSwaggerConversionFailure,
		spec.ErrorKindNotImplemented:
		view.Category = "unsupported spec content"
		view.Title = "Spec contains unsupported content"
		view.Summary = "The spec uses references or features that this explorer cannot normalize yet."
		view.Hint = "Simplify the spec or remove unsupported features, then reload it."
	default:
		if specErr.Err != nil {
			view.Summary = specErr.Err.Error()
		}
	}

	return view
}
