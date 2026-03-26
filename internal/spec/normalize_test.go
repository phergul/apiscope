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

func TestLoadUsesMostSpecificServerOverride(t *testing.T) {
	t.Parallel()

	path := writeTempSpecFile(t, "servers.yaml", `openapi: 3.0.3
info:
  title: Demo
  version: 1.0.0
servers:
  - url: https://root.example.com
paths:
  /root:
    get:
      responses:
        "200":
          description: ok
  /path:
    servers:
      - url: https://path.example.com
    get:
      responses:
        "200":
          description: ok
  /op:
    servers:
      - url: https://path-ignored.example.com
    get:
      servers:
        - url: https://op.example.com
      responses:
        "200":
          description: ok
`)

	spec, err := NewLoader(nil).Load(context.Background(), Source{Value: path})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	ops := map[string]model.Operation{}
	for _, op := range spec.Operations {
		ops[op.Path] = op
	}

	if got := ops["/root"].DefaultServerURLs[0]; got != "https://root.example.com" {
		t.Fatalf("expected root server, got %q", got)
	}
	if got := ops["/path"].DefaultServerURLs[0]; got != "https://path.example.com" {
		t.Fatalf("expected path override server, got %q", got)
	}
	if got := ops["/op"].DefaultServerURLs[0]; got != "https://op.example.com" {
		t.Fatalf("expected operation override server, got %q", got)
	}
}

func TestLoadAppliesParameterOverrideSemantics(t *testing.T) {
	t.Parallel()

	path := writeTempSpecFile(t, "params.yaml", `openapi: 3.0.3
info:
  title: Demo
  version: 1.0.0
paths:
  /pets/{id}:
    parameters:
      - name: id
        in: path
        required: true
        description: path-level
        schema:
          type: string
      - name: trace
        in: header
        schema:
          type: string
    get:
      parameters:
        - name: id
          in: path
          required: true
          description: operation-level
          schema:
            type: string
        - name: limit
          in: query
          schema:
            type: integer
      responses:
        "200":
          description: ok
`)

	spec, err := NewLoader(nil).Load(context.Background(), Source{Value: path})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	params := spec.Operations[0].Parameters
	if len(params) != 3 {
		t.Fatalf("expected 3 merged parameters, got %d", len(params))
	}
	if params[0].Name != "id" || params[0].Description != "operation-level" {
		t.Fatalf("expected path parameter to be overridden in place, got %#v", params[0])
	}
	if params[1].Name != "trace" {
		t.Fatalf("expected second path-only parameter to remain, got %#v", params[1])
	}
	if params[2].Name != "limit" || params[2].In != model.ParameterLocationQuery {
		t.Fatalf("expected operation-only parameter appended, got %#v", params[2])
	}
}

func TestLoadDerivesSourceAwareCapabilities(t *testing.T) {
	t.Parallel()

	oasPath := writeTempSpecFile(t, "oas3.yaml", `openapi: 3.0.3
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
	swaggerPath := writeTempSpecFile(t, "swagger.yaml", `swagger: "2.0"
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

	if !oasSpec.Capabilities.SupportsOpenAPI3 {
		t.Fatal("expected OAS3 spec to report SupportsOpenAPI3")
	}
	if oasSpec.Capabilities.SupportsSwagger2Conversion {
		t.Fatal("expected OAS3 spec to not report Swagger conversion support")
	}
	if swaggerSpec.Capabilities.SupportsOpenAPI3 {
		t.Fatal("expected Swagger spec to not report SupportsOpenAPI3")
	}
	if !swaggerSpec.Capabilities.SupportsSwagger2Conversion {
		t.Fatal("expected Swagger spec to report Swagger conversion support")
	}
	if swaggerSpec.Capabilities.SupportsCookieParameters {
		t.Fatal("expected Swagger spec to not report cookie parameter support")
	}
	if swaggerSpec.Capabilities.SupportsServerVariables {
		t.Fatal("expected Swagger spec to not report server variable support")
	}
	if !swaggerSpec.Capabilities.SupportsRequestBodies || !swaggerSpec.Capabilities.SupportsSecuritySchemes {
		t.Fatal("expected Swagger common execution capabilities to remain true")
	}
}

func TestLoadUsesCanonicalFingerprintAcrossFormatsAndFormatting(t *testing.T) {
	t.Parallel()

	yamlPath := writeTempSpecFile(t, "spec.yaml", `# comment
openapi: 3.0.3
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
	jsonPath := writeTempSpecFile(t, "spec.json", `{"openapi":"3.0.3","info":{"title":"Demo","version":"1.0.0"},"paths":{"/pets":{"get":{"responses":{"200":{"description":"ok"}}}}}}`)
	yamlVariantPath := writeTempSpecFile(t, "spec-variant.yaml", `
openapi: 3.0.3
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

	yamlSpec, err := NewLoader(nil).Load(context.Background(), Source{Value: yamlPath})
	if err != nil {
		t.Fatalf("load yaml: %v", err)
	}
	jsonSpec, err := NewLoader(nil).Load(context.Background(), Source{Value: jsonPath})
	if err != nil {
		t.Fatalf("load json: %v", err)
	}
	yamlVariantSpec, err := NewLoader(nil).Load(context.Background(), Source{Value: yamlVariantPath})
	if err != nil {
		t.Fatalf("load yaml variant: %v", err)
	}

	if yamlSpec.Fingerprint != jsonSpec.Fingerprint {
		t.Fatalf("expected YAML and JSON fingerprints to match, got %q and %q", yamlSpec.Fingerprint, jsonSpec.Fingerprint)
	}
	if yamlSpec.Fingerprint != yamlVariantSpec.Fingerprint {
		t.Fatalf("expected formatting-only variant fingerprint to match, got %q and %q", yamlSpec.Fingerprint, yamlVariantSpec.Fingerprint)
	}
}

func TestLoadUsesSourceAwareFingerprintAcrossEquivalentSwaggerAndOAS3(t *testing.T) {
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

	if oasSpec.Fingerprint == swaggerSpec.Fingerprint {
		t.Fatalf("expected source-aware fingerprints to differ, both were %q", oasSpec.Fingerprint)
	}
}
