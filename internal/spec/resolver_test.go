package spec

import (
	"context"
	"testing"

	"api-tui/internal/model"
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

	resolved, err := newLoader(nil).resolveDocument(context.Background(), converted)
	if err != nil {
		t.Fatalf("resolveDocument returned error: %v", err)
	}

	if resolved.sourceFamily != model.SourceFamilyOpenAPI3 {
		t.Fatalf("expected source family openapi3, got %q", resolved.sourceFamily)
	}
	response := resolved.openAPI3Doc.Paths.Value("/pets").Get.Responses.Value("200")
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

	resolved, err := newLoader(nil).resolveDocument(context.Background(), converted)
	if err != nil {
		t.Fatalf("resolveDocument returned error: %v", err)
	}

	if resolved.sourceFamily != model.SourceFamilySwagger2 {
		t.Fatalf("expected source family swagger2, got %q", resolved.sourceFamily)
	}
	response := resolved.openAPI3Doc.Paths.Value("/pets").Get.Responses.Value("200")
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

	_, err := newLoader(nil).resolveDocument(context.Background(), converted)
	if !IsErrorKind(err, ErrorKindRefResolutionFailure) {
		t.Fatalf("expected ref resolution failure, got %v", err)
	}
}

func TestResolveDocumentRejectsExternalFileRefs(t *testing.T) {
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
                $ref: "other.yaml#/components/schemas/Pet"
`)

	_, err := newLoader(nil).resolveDocument(context.Background(), converted)
	if !IsErrorKind(err, ErrorKindUnsupportedExternalRef) {
		t.Fatalf("expected unsupported external ref, got %v", err)
	}
}

func TestResolveDocumentRejectsRemoteRefs(t *testing.T) {
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
                $ref: "https://example.com/pet.yaml#/components/schemas/Pet"
`)

	_, err := newLoader(nil).resolveDocument(context.Background(), converted)
	if !IsErrorKind(err, ErrorKindUnsupportedExternalRef) {
		t.Fatalf("expected unsupported external ref, got %v", err)
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

	_, err := NewLoader(nil).Load(context.Background(), Source{Value: path})
	if !IsErrorKind(err, ErrorKindRefResolutionFailure) {
		t.Fatalf("expected ref resolution failure, got %v", err)
	}
}

func TestLoadReturnsNormalizedSpecAfterSuccessfulResolution(t *testing.T) {
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

	spec, err := NewLoader(nil).Load(context.Background(), Source{Value: path})
	if err != nil {
		t.Fatalf("expected successful normalized load, got %v", err)
	}
	if spec == nil {
		t.Fatal("expected normalized spec after successful resolution")
	}
}

func mustConvertDocument(t *testing.T, raw string) *convertedDocument {
	t.Helper()

	document := &loadedDocument{
		CanonicalLocation: "spec.yaml",
		Format:            DocumentFormatYAML,
		Raw:               []byte(raw),
	}

	parsed, err := newLoader(nil).parseDocument(document)
	if err != nil {
		t.Fatalf("parseDocument returned error: %v", err)
	}

	converted, err := newLoader(nil).convertDocument(parsed)
	if err != nil {
		t.Fatalf("convertDocument returned error: %v", err)
	}

	return converted
}
