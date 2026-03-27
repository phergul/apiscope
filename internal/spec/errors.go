package spec

import (
	"errors"

	"api-tui/internal/spec/internal/pipeline"
)

const (
	ErrorKindInvalidSource                  = pipeline.ErrorKindInvalidSource
	ErrorKindFileReadFailure                = pipeline.ErrorKindFileReadFailure
	ErrorKindURLFetchFailure                = pipeline.ErrorKindURLFetchFailure
	ErrorKindUnsupportedScheme              = pipeline.ErrorKindUnsupportedScheme
	ErrorKindUnknownFormat                  = pipeline.ErrorKindUnknownFormat
	ErrorKindEmptyDocument                  = pipeline.ErrorKindEmptyDocument
	ErrorKindDecodeFailure                  = pipeline.ErrorKindDecodeFailure
	ErrorKindUnsupportedFamily              = pipeline.ErrorKindUnsupportedFamily
	ErrorKindUnsupportedVersion             = pipeline.ErrorKindUnsupportedVersion
	ErrorKindOpenAPIParseFailure            = pipeline.ErrorKindOpenAPIParseFailure
	ErrorKindSwaggerConversionFailure       = pipeline.ErrorKindSwaggerConversionFailure
	ErrorKindUnsupportedSwaggerConstruct    = pipeline.ErrorKindUnsupportedSwaggerConstruct
	ErrorKindRefResolutionFailure           = pipeline.ErrorKindRefResolutionFailure
	ErrorKindUnsupportedExternalRef         = pipeline.ErrorKindUnsupportedExternalRef
	ErrorKindNormalisationFailure           = pipeline.ErrorKindNormalisationFailure
	ErrorKindUnsupportedNormalisedConstruct = pipeline.ErrorKindUnsupportedNormalisedConstruct
	ErrorKindNotImplemented                 = pipeline.ErrorKindNotImplemented
)

type ErrorKind = pipeline.ErrorKind
type Error = pipeline.Error

func IsErrorKind(err error, kind ErrorKind) bool {
	var specErr *Error
	if !errors.As(err, &specErr) {
		return false
	}

	return specErr.Kind == kind
}
