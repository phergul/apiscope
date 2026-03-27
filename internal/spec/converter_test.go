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

func TestConvertDocumentConvertsSwaggerReusableRefsAndResponseHeaders(t *testing.T) {
	t.Parallel()

	raw := `swagger: "2.0"
info:
  title: Demo
  version: 1.0.0
parameters:
  Limit:
    name: limit
    in: query
    type: array
    items:
      type: string
    collectionFormat: multi
responses:
  Error:
    description: failed
    headers:
      X-Trace:
        type: array
        items:
          type: string
        collectionFormat: csv
paths:
  /pets:
    get:
      parameters:
        - $ref: "#/parameters/Limit"
      responses:
        default:
          $ref: "#/responses/Error"
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

	limit := converted.openAPI3Doc.Components.Parameters["Limit"]
	if limit == nil || limit.Value == nil {
		t.Fatal("expected converted reusable parameter")
	}
	if limit.Value.Style != "form" || limit.Value.Explode == nil || !*limit.Value.Explode {
		t.Fatalf("expected multi query collectionFormat to map to form+explode, got style=%q explode=%v", limit.Value.Style, limit.Value.Explode)
	}

	pathItem := converted.openAPI3Doc.Paths.Value("/pets")
	if pathItem == nil || pathItem.Get == nil {
		t.Fatal("expected converted get operation")
	}
	if got := pathItem.Get.Parameters[0].Ref; got != "#/components/parameters/Limit" {
		t.Fatalf("expected parameter ref rewrite, got %q", got)
	}
	if got := pathItem.Get.Responses.Default().Ref; got != "#/components/responses/Error" {
		t.Fatalf("expected response ref rewrite, got %q", got)
	}

	errorResponse := converted.openAPI3Doc.Components.Responses["Error"]
	if errorResponse == nil || errorResponse.Value == nil {
		t.Fatal("expected converted reusable response")
	}
	header := errorResponse.Value.Headers["X-Trace"]
	if header == nil || header.Value == nil {
		t.Fatal("expected converted response header")
	}
	if header.Value.Style != "simple" || header.Value.Explode == nil || *header.Value.Explode {
		t.Fatalf("expected csv header collectionFormat to map to simple+explode=false, got style=%q explode=%v", header.Value.Style, header.Value.Explode)
	}
}

func TestConvertDocumentRewritesExternalSwaggerRefs(t *testing.T) {
	t.Parallel()

	raw := `swagger: "2.0"
info:
  title: Demo
  version: 1.0.0
paths:
  /pets:
    get:
      parameters:
        - $ref: "common.yaml#/parameters/Limit"
      responses:
        default:
          $ref: "https://example.com/common.yaml#/responses/Error"
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

	pathItem := converted.openAPI3Doc.Paths.Value("/pets")
	if got := pathItem.Get.Parameters[0].Ref; got != "common.yaml#/components/parameters/Limit" {
		t.Fatalf("expected external parameter ref rewrite, got %q", got)
	}
	if got := pathItem.Get.Responses.Default().Ref; got != "https://example.com/common.yaml#/components/responses/Error" {
		t.Fatalf("expected external response ref rewrite, got %q", got)
	}
}

func TestConvertDocumentPreservesSwaggerPathItemRefs(t *testing.T) {
	t.Parallel()

	raw := `swagger: "2.0"
info:
  title: Demo
  version: 1.0.0
paths:
  /pets:
    $ref: "common.yaml#/paths/~1pets"
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

	pathItem := converted.openAPI3Doc.Paths.Value("/pets")
	if pathItem == nil {
		t.Fatal("expected converted path item")
	}
	if got := pathItem.Ref; got != "common.yaml#/paths/~1pets" {
		t.Fatalf("expected path item ref to be preserved, got %q", got)
	}
}

func TestConvertDocumentConvertsSwaggerOAuth2Definitions(t *testing.T) {
	t.Parallel()

	raw := `swagger: "2.0"
info:
  title: Demo
  version: 1.0.0
securityDefinitions:
  petstore_auth:
    type: oauth2
    flow: accessCode
    authorizationUrl: https://example.com/oauth/authorize
    tokenUrl: https://example.com/oauth/token
    scopes:
      read:pets: read your pets
paths:
  /pets:
    get:
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

	converted, err := newLoader(nil).convertDocument(parsed)
	if err != nil {
		t.Fatalf("convertDocument returned error: %v", err)
	}

	scheme := converted.openAPI3Doc.Components.SecuritySchemes["petstore_auth"]
	if scheme == nil || scheme.Value == nil {
		t.Fatal("expected converted oauth2 scheme")
	}
	if scheme.Value.Type != "oauth2" {
		t.Fatalf("expected oauth2 scheme type, got %q", scheme.Value.Type)
	}
	if scheme.Value.Flows == nil || scheme.Value.Flows.AuthorizationCode == nil {
		t.Fatalf("expected accessCode flow to map to authorizationCode, got %#v", scheme.Value.Flows)
	}
}

func TestConvertDocumentMapsSwaggerCollectionFormats(t *testing.T) {
	t.Parallel()

	raw := `swagger: "2.0"
info:
  title: Demo
  version: 1.0.0
paths:
  /pets:
    get:
      parameters:
        - name: multi
          in: query
          type: array
          items:
            type: string
          collectionFormat: multi
        - name: ssv
          in: query
          type: array
          items:
            type: string
          collectionFormat: ssv
        - name: pipes
          in: query
          type: array
          items:
            type: string
          collectionFormat: pipes
        - name: tsv
          in: query
          type: array
          items:
            type: string
          collectionFormat: tsv
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

	converted, err := newLoader(nil).convertDocument(parsed)
	if err != nil {
		t.Fatalf("convertDocument returned error: %v", err)
	}

	params := converted.openAPI3Doc.Paths.Value("/pets").Get.Parameters
	if got := params[0].Value.Style; got != "form" {
		t.Fatalf("expected multi to map to form, got %q", got)
	}
	if params[0].Value.Explode == nil || !*params[0].Value.Explode {
		t.Fatalf("expected multi to map to explode=true, got %v", params[0].Value.Explode)
	}
	if got := params[1].Value.Style; got != "spaceDelimited" {
		t.Fatalf("expected ssv to map to spaceDelimited, got %q", got)
	}
	if params[1].Value.Explode == nil || *params[1].Value.Explode {
		t.Fatalf("expected ssv to map to explode=false, got %v", params[1].Value.Explode)
	}
	if got := params[2].Value.Style; got != "pipeDelimited" {
		t.Fatalf("expected pipes to map to pipeDelimited, got %q", got)
	}
	if params[2].Value.Explode == nil || *params[2].Value.Explode {
		t.Fatalf("expected pipes to map to explode=false, got %v", params[2].Value.Explode)
	}
	if got := params[3].Value.Extensions[swaggerCollectionFormatExtension]; got != "tsv" {
		t.Fatalf("expected tsv to be preserved in extensions, got %#v", got)
	}
}
