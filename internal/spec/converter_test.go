package spec

import (
	"context"
	"testing"

	"api-tui/internal/model"
)

func TestConvertDocumentPassesThroughOpenAPI3(t *testing.T) {
	t.Parallel()

	parsed, err := newLoader(nil).parseDocument(&loadedDocument{
		CanonicalLocation: "spec.json",
		Format:            DocumentFormatJSON,
		Raw:               []byte(`{"openapi":"3.0.3","info":{"title":"Demo","version":"1.0.0"},"paths":{}}`),
	})
	if err != nil {
		t.Fatalf("parseDocument returned error: %v", err)
	}

	converted, err := newLoader(nil).convertDocument(parsed)
	if err != nil {
		t.Fatalf("convertDocument returned error: %v", err)
	}

	if converted.sourceFamily != model.SourceFamilyOpenAPI3 {
		t.Fatalf("expected source family openapi3, got %q", converted.sourceFamily)
	}
	if converted.openAPI3Doc != parsed.openAPI3Doc {
		t.Fatal("expected openapi3 document to pass through unchanged")
	}
}

func TestConvertDocumentConvertsSwagger2CommonCase(t *testing.T) {
	t.Parallel()

	raw := `swagger: "2.0"
host: api.example.com
basePath: /v1
schemes: [https]
consumes: [application/json]
produces: [application/json]
info:
  title: Demo
  version: 1.0.0
paths:
  /pets:
    get:
      operationId: listPets
      summary: List pets
      parameters:
        - name: limit
          in: query
          type: integer
      responses:
        "200":
          description: ok
          schema:
            type: array
            items:
              $ref: "#/definitions/Pet"
    post:
      consumes: [application/json]
      parameters:
        - name: body
          in: body
          required: true
          schema:
            $ref: "#/definitions/Pet"
      responses:
        "201":
          description: created
securityDefinitions:
  api_key:
    type: apiKey
    name: X-API-Key
    in: header
security:
  - api_key: []
definitions:
  Pet:
    type: object
    required: [id]
    properties:
      id:
        type: string
`

	parsed, err := newLoader(nil).parseDocument(&loadedDocument{
		CanonicalLocation: "swagger.yaml",
		Format:            DocumentFormatYAML,
		Raw:               []byte(raw),
	})
	if err != nil {
		t.Fatalf("parseDocument returned error: %v", err)
	}

	converted, err := newLoader(nil).convertDocument(parsed)
	if err != nil {
		t.Fatalf("convertDocument returned error: %v", err)
	}

	if converted.sourceFamily != model.SourceFamilySwagger2 {
		t.Fatalf("expected swagger2 source family, got %q", converted.sourceFamily)
	}
	if got := converted.openAPI3Doc.Servers[0].URL; got != "https://api.example.com/v1" {
		t.Fatalf("expected converted server, got %q", got)
	}
	if converted.openAPI3Doc.Paths == nil {
		t.Fatal("expected converted paths")
	}
	pathItem := converted.openAPI3Doc.Paths.Value("/pets")
	if pathItem == nil || pathItem.Get == nil || pathItem.Post == nil {
		t.Fatal("expected get and post operations to be converted")
	}
	if len(pathItem.Get.Parameters) != 1 {
		t.Fatalf("expected one query parameter, got %d", len(pathItem.Get.Parameters))
	}
	if pathItem.Post.RequestBody == nil || pathItem.Post.RequestBody.Value == nil {
		t.Fatal("expected request body to be converted")
	}
	if pathItem.Post.RequestBody.Value.Content["application/json"] == nil {
		t.Fatal("expected request body media type from consumes")
	}
	if converted.openAPI3Doc.Components == nil || converted.openAPI3Doc.Components.SecuritySchemes["api_key"] == nil {
		t.Fatal("expected security definition to be converted")
	}
	if len(converted.openAPI3Doc.Security) != 1 {
		t.Fatalf("expected top-level security to be preserved, got %d entries", len(converted.openAPI3Doc.Security))
	}
	response := pathItem.Get.Responses.Value("200")
	if response == nil || response.Value == nil {
		t.Fatal("expected response 200 to be converted")
	}
	if response.Value.Content["application/json"] == nil {
		t.Fatal("expected response media type from produces")
	}
}

func TestConvertDocumentRejectsUnsupportedSwaggerConstruct(t *testing.T) {
	t.Parallel()

	raw := `swagger: "2.0"
info:
  title: Demo
  version: 1.0.0
paths:
  /upload:
    post:
      parameters:
        - name: file
          in: formData
          type: string
      responses:
        "200":
          description: ok
`

	parsed, err := newLoader(nil).parseDocument(&loadedDocument{
		CanonicalLocation: "swagger.yaml",
		Format:            DocumentFormatYAML,
		Raw:               []byte(raw),
	})
	if err != nil {
		t.Fatalf("parseDocument returned error: %v", err)
	}

	_, err = newLoader(nil).convertDocument(parsed)
	if !IsErrorKind(err, ErrorKindUnsupportedSwaggerConstruct) {
		t.Fatalf("expected unsupported swagger construct error, got %v", err)
	}
}

func TestLoadReturnsConversionErrorForUnsupportedSwaggerInput(t *testing.T) {
	t.Parallel()

	path := writeTempSpecFile(t, "swagger.yaml", `swagger: "2.0"
info:
  title: Demo
  version: 1.0.0
paths:
  /upload:
    post:
      parameters:
        - name: file
          in: formData
          type: string
      responses:
        "200":
          description: ok
`)

	_, err := NewLoader(nil).Load(context.Background(), Source{Value: path})
	if !IsErrorKind(err, ErrorKindUnsupportedSwaggerConstruct) {
		t.Fatalf("expected unsupported swagger construct error, got %v", err)
	}
}

func TestLoadReturnsNotImplementedAfterSuccessfulConversion(t *testing.T) {
	t.Parallel()

	path := writeTempSpecFile(t, "swagger.yaml", `swagger: "2.0"
info:
  title: Demo
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
`)

	_, err := NewLoader(nil).Load(context.Background(), Source{Value: path})
	if !IsErrorKind(err, ErrorKindNotImplemented) {
		t.Fatalf("expected not implemented error, got %v", err)
	}
}
