package spec

import (
	"context"
	"testing"

	"api-tui/internal/model"
)

func TestLoadReturnsNormalizedOpenAPI3Spec(t *testing.T) {
	t.Parallel()

	path := writeTempSpecFile(t, "oas3.yaml", `openapi: 3.0.3
info:
  title: Demo API
  summary: Demo summary
  description: Demo description
  version: 1.0.0
servers:
  - url: https://api.example.com
paths:
  /pets/{id}:
    get:
      operationId: getPet
      summary: Get pet
      tags: [pets]
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
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
      required: [id]
      properties:
        id:
          type: string
`)

	spec, err := NewLoader(nil).Load(context.Background(), Source{Value: path})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if spec == nil {
		t.Fatal("expected normalized spec")
	}
	if spec.Title != "Demo API" {
		t.Fatalf("expected title Demo API, got %q", spec.Title)
	}
	if spec.SourceFamily != model.SourceFamilyOpenAPI3 {
		t.Fatalf("expected openapi3 source family, got %q", spec.SourceFamily)
	}
	if len(spec.Operations) != 1 {
		t.Fatalf("expected 1 operation, got %d", len(spec.Operations))
	}
	op := spec.Operations[0]
	if op.Key != model.NewOperationKey("GET", "/pets/{id}") {
		t.Fatalf("unexpected operation key: %q", op.Key)
	}
	if op.ID != "getPet" {
		t.Fatalf("expected operationId metadata to be preserved, got %q", op.ID)
	}
	if len(op.Parameters) != 1 || op.Parameters[0].In != model.ParameterLocationPath {
		t.Fatalf("expected normalized path parameter, got %#v", op.Parameters)
	}
	if len(op.Responses) != 1 || len(op.Responses[0].Content) != 1 {
		t.Fatalf("expected normalized response content, got %#v", op.Responses)
	}
	if spec.Fingerprint == "" {
		t.Fatal("expected fingerprint to be populated")
	}
}

func TestLoadReturnsNormalizedSwagger2Spec(t *testing.T) {
	t.Parallel()

	path := writeTempSpecFile(t, "swagger.yaml", `swagger: "2.0"
host: api.example.com
basePath: /v1
schemes: [https]
info:
  title: Swagger Demo
  version: 1.0.0
paths:
  /pets:
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
    properties:
      id:
        type: string
`)

	spec, err := NewLoader(nil).Load(context.Background(), Source{Value: path})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if spec.SourceFamily != model.SourceFamilySwagger2 {
		t.Fatalf("expected swagger2 source family, got %q", spec.SourceFamily)
	}
	if len(spec.Servers) != 1 || spec.Servers[0].URL != "https://api.example.com/v1" {
		t.Fatalf("expected normalized server from swagger, got %#v", spec.Servers)
	}
	if len(spec.Operations) != 1 {
		t.Fatalf("expected 1 operation, got %d", len(spec.Operations))
	}
	op := spec.Operations[0]
	if op.RequestBody == nil || len(op.RequestBody.Content) != 1 {
		t.Fatalf("expected normalized request body, got %#v", op.RequestBody)
	}
	if spec.Security == nil || len(spec.Security.Alternatives) != 1 {
		t.Fatalf("expected top-level security, got %#v", spec.Security)
	}
	if spec.SecuritySchemes["api_key"].Type != model.SecuritySchemeTypeAPIKey {
		t.Fatalf("expected normalized api key security scheme, got %#v", spec.SecuritySchemes["api_key"])
	}
}

func TestLoadNormalizesEquivalentSwaggerAndOAS3Shapes(t *testing.T) {
	t.Parallel()

	oasPath := writeTempSpecFile(t, "oas3.yaml", `openapi: 3.0.3
info:
  title: Demo
  version: 1.0.0
servers:
  - url: https://api.example.com/v1
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
`)
	swaggerPath := writeTempSpecFile(t, "swagger.yaml", `swagger: "2.0"
host: api.example.com
basePath: /v1
schemes: [https]
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

	oasSpec, err := NewLoader(nil).Load(context.Background(), Source{Value: oasPath})
	if err != nil {
		t.Fatalf("load oas3: %v", err)
	}
	swaggerSpec, err := NewLoader(nil).Load(context.Background(), Source{Value: swaggerPath})
	if err != nil {
		t.Fatalf("load swagger: %v", err)
	}

	if len(oasSpec.Operations) != len(swaggerSpec.Operations) {
		t.Fatalf("expected matching operation counts, got %d and %d", len(oasSpec.Operations), len(swaggerSpec.Operations))
	}
	if oasSpec.Operations[0].Key != swaggerSpec.Operations[0].Key {
		t.Fatalf("expected matching normalized operation keys, got %q and %q", oasSpec.Operations[0].Key, swaggerSpec.Operations[0].Key)
	}
	if oasSpec.Servers[0].URL != swaggerSpec.Servers[0].URL {
		t.Fatalf("expected matching normalized server urls, got %q and %q", oasSpec.Servers[0].URL, swaggerSpec.Servers[0].URL)
	}
}

func TestLoadDerivesCapabilitiesAndWarnings(t *testing.T) {
	t.Parallel()

	path := writeTempSpecFile(t, "warnings.yaml", `openapi: 3.0.3
info:
  title: Demo
  version: 1.0.0
components:
  securitySchemes:
    digest:
      type: http
      scheme: digest
paths:
  /pets:
    get:
      parameters:
        - name: sid
          in: cookie
          schema:
            type: string
      responses:
        "200":
          description: ok
`)

	spec, err := NewLoader(nil).Load(context.Background(), Source{Value: path})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if !spec.Capabilities.SupportsCookieParameters {
		t.Fatal("expected cookie parameter capability")
	}
	if !spec.Capabilities.SupportsSecuritySchemes {
		t.Fatal("expected security scheme capability")
	}
	if len(spec.Warnings) == 0 {
		t.Fatal("expected normalization warnings")
	}
}
