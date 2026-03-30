package spec

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/phergul/apiscope/internal/model"
	"github.com/phergul/apiscope/internal/spec/internal/pipeline"

	"github.com/getkin/kin-openapi/openapi3"
	"gopkg.in/yaml.v3"
)

// parseDocument decodes a loaded document and identifies its source family.
func (l *Loader) parseDocument(document *pipeline.LoadedDocument) (*pipeline.ParsedDocument, error) {
	decoded, err := decodeDocument(document)
	if err != nil {
		return nil, err
	}

	family, version, err := detectSpecFamilyVersion(document, decoded)
	if err != nil {
		return nil, err
	}

	parsed := &pipeline.ParsedDocument{
		BaseDocument: pipeline.BaseDocument{
			Document:      document,
			SourceFamily:  family,
			SourceVersion: version,
		},
	}

	switch family {
	case model.SourceFamilyOpenAPI3:
		openapiDoc, err := parseOpenAPI3Document(document, decoded)
		if err != nil {
			return nil, err
		}
		parsed.OpenAPI3Doc = openapiDoc
	case model.SourceFamilySwagger2:
		parsed.SwaggerDoc = decoded
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

// decodeDocument decodes the raw JSON or YAML document into a generic object map.
func decodeDocument(document *pipeline.LoadedDocument) (map[string]any, error) {
	var decoded map[string]any

	switch document.Format {
	case pipeline.DocumentFormatJSON:
		if err := json.Unmarshal(document.Raw, &decoded); err != nil {
			return nil, &Error{
				Kind:   ErrorKindDecodeFailure,
				Op:     "decode json",
				Source: document.CanonicalLocation,
				Err:    err,
			}
		}
	case pipeline.DocumentFormatYAML:
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

// detectSpecFamilyVersion identifies the supported spec family and version markers.
func detectSpecFamilyVersion(document *pipeline.LoadedDocument, decoded map[string]any) (model.SourceFamily, string, error) {
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

// parseOpenAPI3Document converts a decoded OpenAPI 3 document into the kin-openapi model.
func parseOpenAPI3Document(document *pipeline.LoadedDocument, decoded map[string]any) (*openapi3.T, error) {
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
