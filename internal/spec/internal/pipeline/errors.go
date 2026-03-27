package pipeline

import "fmt"

type ErrorKind string

const (
	ErrorKindInvalidSource                  ErrorKind = "invalid_source"
	ErrorKindFileReadFailure                ErrorKind = "file_read_failure"
	ErrorKindURLFetchFailure                ErrorKind = "url_fetch_failure"
	ErrorKindUnsupportedScheme              ErrorKind = "unsupported_source_scheme"
	ErrorKindUnknownFormat                  ErrorKind = "unknown_document_format"
	ErrorKindEmptyDocument                  ErrorKind = "empty_document"
	ErrorKindDecodeFailure                  ErrorKind = "decode_failure"
	ErrorKindUnsupportedFamily              ErrorKind = "unsupported_spec_family"
	ErrorKindUnsupportedVersion             ErrorKind = "unsupported_spec_version"
	ErrorKindOpenAPIParseFailure            ErrorKind = "openapi_parse_failure"
	ErrorKindSwaggerConversionFailure       ErrorKind = "swagger_conversion_failure"
	ErrorKindUnsupportedSwaggerConstruct    ErrorKind = "unsupported_swagger_construct"
	ErrorKindRefResolutionFailure           ErrorKind = "ref_resolution_failure"
	ErrorKindUnsupportedExternalRef         ErrorKind = "unsupported_external_ref"
	ErrorKindNormalisationFailure           ErrorKind = "normalisation_failure"
	ErrorKindUnsupportedNormalisedConstruct ErrorKind = "unsupported_normalised_construct"
	ErrorKindNotImplemented                 ErrorKind = "not_implemented"
)

type Error struct {
	Kind       ErrorKind
	Op         string
	Source     string
	StatusCode int
	Err        error
}

func (e *Error) Error() string {
	base := fmt.Sprintf("%s: %s", e.Op, e.Source)
	if e.StatusCode > 0 {
		base = fmt.Sprintf("%s (status=%d)", base, e.StatusCode)
	}
	if e.Err == nil {
		return fmt.Sprintf("%s: %s", e.Kind, base)
	}

	return fmt.Sprintf("%s: %s: %v", e.Kind, base, e.Err)
}

func (e *Error) Unwrap() error {
	return e.Err
}
