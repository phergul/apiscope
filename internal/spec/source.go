package spec

import (
	"net/url"
	"strings"
)

type SourceKind string

const (
	SourceKindFile SourceKind = "file"
	SourceKindURL  SourceKind = "url"
)

type Source struct {
	Kind  SourceKind
	Value string
}

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
