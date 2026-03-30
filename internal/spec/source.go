package spec

import (
	"net/url"
	"strings"

	"github.com/phergul/apiscope/internal/spec/internal/pipeline"
)

type SourceKind = pipeline.SourceKind
type Source = pipeline.Source

const (
	SourceKindFile = pipeline.SourceKindFile
	SourceKindURL  = pipeline.SourceKindURL
)

// classifySource validates and canonicalizes a raw source input.
func classifySource(source Source) (Source, error) {
	value := strings.TrimSpace(source.Value)
	if value == "" {
		return Source{}, &Error{
			Kind:   ErrorKindInvalidSource,
			Op:     "classify source",
			Source: source.Value,
			Err:    errEmptySource,
		}
	}

	switch source.Kind {
	case SourceKindFile:
		return Source{Kind: SourceKindFile, Value: value}, nil
	case SourceKindURL:
		return validateURLSource(value)
	case "":
		return inferSourceKind(value)
	default:
		return Source{}, &Error{
			Kind:   ErrorKindInvalidSource,
			Op:     "classify source",
			Source: source.Value,
			Err:    errUnknownSourceKind,
		}
	}
}

// inferSourceKind infers whether a raw source value is a file path or URL.
func inferSourceKind(value string) (Source, error) {
	parsed, err := url.Parse(value)
	if err != nil {
		return Source{Kind: SourceKindFile, Value: value}, nil
	}

	if parsed.Scheme == "" {
		return Source{Kind: SourceKindFile, Value: value}, nil
	}

	switch parsed.Scheme {
	case "http", "https":
		return Source{Kind: SourceKindURL, Value: value}, nil
	default:
		return Source{}, &Error{
			Kind:   ErrorKindUnsupportedScheme,
			Op:     "classify source",
			Source: value,
			Err:    errUnsupportedSourceScheme,
		}
	}
}

// validateURLSource validates that a raw source value is a supported URL source.
func validateURLSource(value string) (Source, error) {
	parsed, err := url.Parse(value)
	if err != nil {
		return Source{}, &Error{
			Kind:   ErrorKindInvalidSource,
			Op:     "parse url",
			Source: value,
			Err:    err,
		}
	}

	switch parsed.Scheme {
	case "http", "https":
		if parsed.Host == "" {
			return Source{}, &Error{
				Kind:   ErrorKindInvalidSource,
				Op:     "parse url",
				Source: value,
				Err:    errMissingURLHost,
			}
		}
		return Source{Kind: SourceKindURL, Value: value}, nil
	default:
		return Source{}, &Error{
			Kind:   ErrorKindUnsupportedScheme,
			Op:     "parse url",
			Source: value,
			Err:    errUnsupportedSourceScheme,
		}
	}
}
