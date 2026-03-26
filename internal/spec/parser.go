package spec

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"api-tui/internal/model"

	"github.com/getkin/kin-openapi/openapi3"
	"gopkg.in/yaml.v3"
)

type parsedDocument struct {
	document      *loadedDocument
	sourceFamily  model.SourceFamily
	sourceVersion string
	openAPI3Doc   *openapi3.T
	swaggerDoc    map[string]any
}

func (l *loader) parseDocument(document *loadedDocument) (*parsedDocument, error) {
	decoded, err := decodeDocument(document)
	if err != nil {
		return nil, err
	}

	family, version, err := detectSpecFamilyVersion(document, decoded)
	if err != nil {
		return nil, err
	}

	parsed := &parsedDocument{
		document:      document,
		sourceFamily:  family,
		sourceVersion: version,
	}

	switch family {
	case model.SourceFamilyOpenAPI3:
		openapiDoc, err := parseOpenAPI3Document(document, decoded)
		if err != nil {
			return nil, err
		}
		parsed.openAPI3Doc = openapiDoc
	case model.SourceFamilySwagger2:
		parsed.swaggerDoc = decoded
	default:
		return nil, &Error{
			Kind:   ErrorKindUnsupportedFamily,
			Op:     "parse document",
			Source: document.CanonicalLocation,
			Err:    fmt.Errorf("unexpected source family %q", family),
		}
	}

	return parsed, nil
}

func decodeDocument(document *loadedDocument) (map[string]any, error) {
	var decoded map[string]any

	switch document.Format {
	case DocumentFormatJSON:
		if err := json.Unmarshal(document.Raw, &decoded); err != nil {
			return nil, &Error{
				Kind:   ErrorKindDecodeFailure,
				Op:     "decode json",
				Source: document.CanonicalLocation,
				Err:    err,
			}
		}
	case DocumentFormatYAML:
		if err := yaml.Unmarshal(document.Raw, &decoded); err != nil {
			return nil, &Error{
				Kind:   ErrorKindDecodeFailure,
				Op:     "decode yaml",
				Source: document.CanonicalLocation,
				Err:    err,
			}
		}
	default:
		return nil, &Error{
			Kind:   ErrorKindUnknownFormat,
			Op:     "decode document",
			Source: document.CanonicalLocation,
			Err:    fmt.Errorf("unsupported document format %q", document.Format),
		}
	}

	if len(decoded) == 0 {
		return nil, &Error{
			Kind:   ErrorKindDecodeFailure,
			Op:     "decode document",
			Source: document.CanonicalLocation,
			Err:    errors.New("document must decode to a non-empty object"),
		}
	}

	return decoded, nil
}

func detectSpecFamilyVersion(document *loadedDocument, decoded map[string]any) (model.SourceFamily, string, error) {
	if rawVersion, ok := decoded["openapi"]; ok {
		version := strings.TrimSpace(fmt.Sprint(rawVersion))
		if strings.HasPrefix(version, "3.") {
			return model.SourceFamilyOpenAPI3, version, nil
		}

		return model.SourceFamilyUnknown, version, &Error{
			Kind:   ErrorKindUnsupportedVersion,
			Op:     "detect spec family",
			Source: document.CanonicalLocation,
			Err:    fmt.Errorf("unsupported OpenAPI version %q", version),
		}
	}

	if rawVersion, ok := decoded["swagger"]; ok {
		version := strings.TrimSpace(fmt.Sprint(rawVersion))
		if version == "2.0" {
			return model.SourceFamilySwagger2, version, nil
		}

		return model.SourceFamilyUnknown, version, &Error{
			Kind:   ErrorKindUnsupportedVersion,
			Op:     "detect spec family",
			Source: document.CanonicalLocation,
			Err:    fmt.Errorf("unsupported Swagger version %q", version),
		}
	}

	return model.SourceFamilyUnknown, "", &Error{
		Kind:   ErrorKindUnsupportedFamily,
		Op:     "detect spec family",
		Source: document.CanonicalLocation,
		Err:    errors.New("document is missing openapi or swagger version markers"),
	}
}

func parseOpenAPI3Document(document *loadedDocument, decoded map[string]any) (*openapi3.T, error) {
	jsonBytes, err := json.Marshal(decoded)
	if err != nil {
		return nil, &Error{
			Kind:   ErrorKindOpenAPIParseFailure,
			Op:     "marshal openapi document",
			Source: document.CanonicalLocation,
			Err:    err,
		}
	}

	var doc openapi3.T
	if err := json.Unmarshal(jsonBytes, &doc); err != nil {
		return nil, &Error{
			Kind:   ErrorKindOpenAPIParseFailure,
			Op:     "parse openapi document",
			Source: document.CanonicalLocation,
			Err:    err,
		}
	}

	return &doc, nil
}
