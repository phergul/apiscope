package spec

import (
	"context"
	"net/http"
	"testing"

	"github.com/phergul/apiscope/internal/model"
	subconverter "github.com/phergul/apiscope/internal/spec/internal/converter"
	"github.com/phergul/apiscope/internal/spec/internal/pipeline"
)

func TestResolveDocumentResolvesInternalOpenAPIRefs(t *testing.T) {
	t.Parallel()

	converted := mustConvertDocument(t, `openapi: 3.0.3
info:
  title: Demo
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Pet"
components:
  schemas:
    Pet:
      type: object
      properties:
        id:
          type: string
`)

	resolved, err := NewLoader(nil, nil).resolveDocument(context.Background(), converted)
	if err != nil {
		t.Fatalf("resolveDocument returned error: %v", err)
	}

	if resolved.SourceFamily != model.SourceFamilyOpenAPI3 {
		t.Fatalf("expected source family openapi3, got %q", resolved.SourceFamily)
	}
	response := resolved.OpenAPI3Doc.Paths.Value("/pets").Get.Responses.Value("200")
	schema := response.Value.Content["application/json"].Schema
	if schema == nil || schema.Value == nil {
		t.Fatal("expected schema ref to be resolved")
	}
}

func TestResolveDocumentResolvesConvertedSwaggerRefs(t *testing.T) {
	t.Parallel()

	converted := mustConvertDocument(t, `swagger: "2.0"
info:
  title: Demo
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
          schema:
            $ref: "#/definitions/Pet"
definitions:
  Pet:
    type: object
    properties:
      id:
        type: string
`)

	resolved, err := NewLoader(nil, nil).resolveDocument(context.Background(), converted)
	if err != nil {
		t.Fatalf("resolveDocument returned error: %v", err)
	}

	if resolved.SourceFamily != model.SourceFamilySwagger2 {
		t.Fatalf("expected source family swagger2, got %q", resolved.SourceFamily)
	}
	response := resolved.OpenAPI3Doc.Paths.Value("/pets").Get.Responses.Value("200")
	schema := response.Value.Content["application/json"].Schema
	if schema == nil || schema.Value == nil {
		t.Fatal("expected converted swagger schema ref to be resolved")
	}
}

func TestResolveDocumentRejectsMissingInternalTargets(t *testing.T) {
	t.Parallel()

	converted := mustConvertDocument(t, `openapi: 3.0.3
info:
  title: Demo
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Missing"
components:
  schemas:
    Pet:
      type: object
`)

	_, err := NewLoader(nil, nil).resolveDocument(context.Background(), converted)
	if !IsErrorKind(err, ErrorKindRefResolutionFailure) {
		t.Fatalf("expected ref resolution failure, got %v", err)
	}
}

func TestResolveDocumentResolvesExternalFileRefs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	rootPath := writeTempSpecFileInDir(t, dir, "root.yaml", `openapi: 3.0.3
info:
  title: Demo
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "other.yaml#/components/schemas/Pet"
`)
	writeTempSpecFileInDir(t, dir, "other.yaml", `openapi: 3.0.3
info:
  title: Child
  version: 1.0.0
components:
  schemas:
    Pet:
      type: object
      properties:
        id:
          type: string
`)

	document, err := NewLoader(nil, nil).loadDocument(context.Background(), Source{Value: rootPath})
	if err != nil {
		t.Fatalf("loadDocument returned error: %v", err)
	}
	converted := mustConvertLoadedDocument(t, document)

	resolved, err := NewLoader(nil, nil).resolveDocument(context.Background(), converted)
	if err != nil {
		t.Fatalf("resolveDocument returned error: %v", err)
	}

	schema := resolved.OpenAPI3Doc.Paths.Value("/pets").Get.Responses.Value("200").Value.Content["application/json"].Schema
	if schema == nil || schema.Value == nil {
		t.Fatal("expected external file ref to be resolved")
	}
}

func TestResolveDocumentResolvesRemoteRefs(t *testing.T) {
	t.Parallel()

	converted := mustConvertLoadedDocument(t, &pipeline.LoadedDocument{
		Source:            pipeline.Source{Kind: pipeline.SourceKindURL, Value: "https://example.com/spec/root.yaml"},
		CanonicalLocation: "https://example.com/spec/root.yaml",
		FinalURL:          "https://example.com/spec/root.yaml",
		Format:            pipeline.DocumentFormatYAML,
		Raw: []byte(`openapi: 3.0.3
info:
  title: Demo
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "https://example.com/pet.yaml#/components/schemas/Pet"
`),
	})

	client := newTestClient(func(req *http.Request) (*http.Response, error) {
		switch req.URL.String() {
		case "https://example.com/pet.yaml":
			return stringResponse(req, http.StatusOK, "application/yaml", `openapi: 3.0.3
info:
  title: Child
  version: 1.0.0
components:
  schemas:
    Pet:
      type: object
      properties:
        id:
          type: string
`), nil
		default:
			return stringResponse(req, http.StatusNotFound, "text/plain", "not found"), nil
		}
	})

	resolved, err := NewLoader(client, nil).resolveDocument(context.Background(), converted)
	if err != nil {
		t.Fatalf("resolveDocument returned error: %v", err)
	}

	schema := resolved.OpenAPI3Doc.Paths.Value("/pets").Get.Responses.Value("200").Value.Content["application/json"].Schema
	if schema == nil || schema.Value == nil {
		t.Fatal("expected remote ref to be resolved")
	}
}

func TestResolveDocumentUsesRedirectedFinalURLForRelativeRefs(t *testing.T) {
	t.Parallel()

	converted := mustConvertLoadedDocument(t, &pipeline.LoadedDocument{
		Source:            pipeline.Source{Kind: pipeline.SourceKindURL, Value: "https://example.com/start/root.yaml"},
		CanonicalLocation: "https://example.com/final/root.yaml",
		FinalURL:          "https://example.com/final/root.yaml",
		Format:            pipeline.DocumentFormatYAML,
		Raw: []byte(`openapi: 3.0.3
info:
  title: Demo
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "common.yaml#/components/schemas/Pet"
`),
	})

	client := newTestClient(func(req *http.Request) (*http.Response, error) {
		switch req.URL.String() {
		case "https://example.com/final/common.yaml":
			return stringResponse(req, http.StatusOK, "application/yaml", `openapi: 3.0.3
info:
  title: Child
  version: 1.0.0
components:
  schemas:
    Pet:
      type: object
`), nil
		default:
			return stringResponse(req, http.StatusNotFound, "text/plain", "not found"), nil
		}
	})

	resolved, err := NewLoader(client, nil).resolveDocument(context.Background(), converted)
	if err != nil {
		t.Fatalf("resolveDocument returned error: %v", err)
	}

	schema := resolved.OpenAPI3Doc.Paths.Value("/pets").Get.Responses.Value("200").Value.Content["application/json"].Schema
	if schema == nil || schema.Value == nil {
		t.Fatal("expected redirected final URL to be used as the ref base")
	}
}

func TestResolveDocumentRejectsUnsupportedExternalSchemes(t *testing.T) {
	t.Parallel()

	converted := mustConvertLoadedDocument(t, &pipeline.LoadedDocument{
		CanonicalLocation: "spec.yaml",
		Format:            pipeline.DocumentFormatYAML,
		Raw: []byte(`openapi: 3.0.3
info:
  title: Demo
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "ftp://example.com/pet.yaml#/components/schemas/Pet"
`),
	})

	_, err := NewLoader(nil, nil).resolveDocument(context.Background(), converted)
	if !IsErrorKind(err, ErrorKindUnsupportedExternalRef) {
		t.Fatalf("expected unsupported external ref, got %v", err)
	}
}

func TestResolveDocumentReturnsRefResolutionFailureForMissingExternalTargets(t *testing.T) {
	t.Parallel()

	converted := mustConvertLoadedDocument(t, &pipeline.LoadedDocument{
		Source:            pipeline.Source{Kind: pipeline.SourceKindURL, Value: "https://example.com/spec/root.yaml"},
		CanonicalLocation: "https://example.com/spec/root.yaml",
		FinalURL:          "https://example.com/spec/root.yaml",
		Format:            pipeline.DocumentFormatYAML,
		Raw: []byte(`openapi: 3.0.3
info:
  title: Demo
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "https://example.com/missing.yaml#/components/schemas/Pet"
`),
	})

	client := newTestClient(func(req *http.Request) (*http.Response, error) {
		return stringResponse(req, http.StatusNotFound, "text/plain", "missing"), nil
	})

	_, err := NewLoader(client, nil).resolveDocument(context.Background(), converted)
	if !IsErrorKind(err, ErrorKindRefResolutionFailure) {
		t.Fatalf("expected ref resolution failure, got %v", err)
	}
}

func TestLoadReturnsResolverErrorForBadRefs(t *testing.T) {
	t.Parallel()

	path := writeTempSpecFile(t, "badref.yaml", `openapi: 3.0.3
info:
  title: Demo
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Missing"
components:
  schemas:
    Pet:
      type: object
`)

	_, err := NewLoader(nil, nil).Load(context.Background(), Source{Value: path})
	if !IsErrorKind(err, ErrorKindRefResolutionFailure) {
		t.Fatalf("expected ref resolution failure, got %v", err)
	}
}

func TestLoadReturnsNormalisedSpecAfterSuccessfulResolution(t *testing.T) {
	t.Parallel()

	path := writeTempSpecFile(t, "resolved.yaml", `openapi: 3.0.3
info:
  title: Demo
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Pet"
components:
  schemas:
    Pet:
      type: object
      properties:
        id:
          type: string
`)

	spec, err := NewLoader(nil, nil).Load(context.Background(), Source{Value: path})
	if err != nil {
		t.Fatalf("expected successful normalised load, got %v", err)
	}
	if spec == nil {
		t.Fatal("expected normalised spec after successful resolution")
	}
}

func mustConvertDocument(t *testing.T, raw string) *pipeline.ConvertedDocument {
	t.Helper()

	return mustConvertLoadedDocument(t, &pipeline.LoadedDocument{
		CanonicalLocation: "spec.yaml",
		Format:            pipeline.DocumentFormatYAML,
		Raw:               []byte(raw),
	})
}

func mustConvertLoadedDocument(t *testing.T, document *pipeline.LoadedDocument) *pipeline.ConvertedDocument {
	t.Helper()

	parsed, err := NewLoader(nil, nil).parseDocument(document)
	if err != nil {
		t.Fatalf("parseDocument returned error: %v", err)
	}

	converted, err := subconverter.Convert(parsed)
	if err != nil {
		t.Fatalf("convertDocument returned error: %v", err)
	}

	return converted
}
