package normalise

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/phergul/apiscope/internal/model"
	"github.com/phergul/apiscope/internal/spec/internal/converter"
	"github.com/phergul/apiscope/internal/spec/internal/pipeline"

	"github.com/getkin/kin-openapi/openapi3"
	"gopkg.in/yaml.v3"
)

func TestDocumentReturnsNormalisedOpenAPI3Spec(t *testing.T) {
	t.Parallel()

	spec, err := Document(mustResolvedOpenAPI3(t, `openapi: 3.0.3
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
                type: object
                required: [id]
                properties:
                  id:
                    type: string
`))
	if err != nil {
		t.Fatalf("Document returned error: %v", err)
	}
	if spec == nil {
		t.Fatal("expected normalised spec")
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
		t.Fatalf("expected normalised path parameter, got %#v", op.Parameters)
	}
	if len(op.Responses) != 1 || len(op.Responses[0].Content) != 1 {
		t.Fatalf("expected normalised response content, got %#v", op.Responses)
	}
	if spec.Fingerprint == "" {
		t.Fatal("expected fingerprint to be populated")
	}
}

func TestDocumentReturnsNormalisedSwagger2Spec(t *testing.T) {
	t.Parallel()

	spec, err := Document(mustResolvedSwagger(t, `swagger: "2.0"
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
`))
	if err != nil {
		t.Fatalf("Document returned error: %v", err)
	}
	if spec.SourceFamily != model.SourceFamilySwagger2 {
		t.Fatalf("expected swagger2 source family, got %q", spec.SourceFamily)
	}
	if len(spec.Servers) != 1 || spec.Servers[0].URL != "https://api.example.com/v1" {
		t.Fatalf("expected normalised server from swagger, got %#v", spec.Servers)
	}
	if len(spec.Operations) != 1 {
		t.Fatalf("expected 1 operation, got %d", len(spec.Operations))
	}
	op := spec.Operations[0]
	if op.RequestBody == nil || len(op.RequestBody.Content) != 1 {
		t.Fatalf("expected normalised request body, got %#v", op.RequestBody)
	}
	if spec.Security == nil || len(spec.Security.Alternatives) != 1 {
		t.Fatalf("expected top-level security, got %#v", spec.Security)
	}
	if spec.SecuritySchemes["api_key"].Type != model.SecuritySchemeTypeAPIKey {
		t.Fatalf("expected normalised api key security scheme, got %#v", spec.SecuritySchemes["api_key"])
	}
}

func TestDocumentNormalisesSwaggerFormDataAsFormParameters(t *testing.T) {
	t.Parallel()

	spec, err := Document(mustResolvedSwagger(t, `swagger: "2.0"
info:
  title: Swagger Demo
  version: 1.0.0
paths:
  /pets:
    post:
      consumes: [application/x-www-form-urlencoded]
      parameters:
        - name: name
          in: formData
          required: true
          type: string
      responses:
        "200":
          description: ok
`))
	if err != nil {
		t.Fatalf("Document returned error: %v", err)
	}

	if len(spec.Operations) != 1 {
		t.Fatalf("expected 1 operation, got %d", len(spec.Operations))
	}
	op := spec.Operations[0]
	if op.FormBodyMediaType != "application/x-www-form-urlencoded" {
		t.Fatalf("expected form body media type marker, got %q", op.FormBodyMediaType)
	}
	if op.RequestBody != nil {
		t.Fatalf("expected form-only operation to omit request body, got %#v", op.RequestBody)
	}
	if len(op.Parameters) != 1 || op.Parameters[0].In != model.ParameterLocationForm {
		t.Fatalf("expected normalised form parameter, got %#v", op.Parameters)
	}
}

func TestDocumentNormalisesReusableSwaggerFormDataParameterRef(t *testing.T) {
	t.Parallel()

	spec, err := Document(mustResolvedSwagger(t, `swagger: "2.0"
info:
  title: Swagger Demo
  version: 1.0.0
parameters:
  PetName:
    name: name
    in: formData
    required: true
    type: string
paths:
  /pets:
    post:
      consumes: [application/x-www-form-urlencoded]
      parameters:
        - $ref: "#/parameters/PetName"
      responses:
        "200":
          description: ok
`))
	if err != nil {
		t.Fatalf("Document returned error: %v", err)
	}

	op := spec.Operations[0]
	if op.FormBodyMediaType != "application/x-www-form-urlencoded" {
		t.Fatalf("expected form body media type marker, got %q", op.FormBodyMediaType)
	}
	if len(op.Parameters) != 1 || op.Parameters[0].In != model.ParameterLocationForm {
		t.Fatalf("expected reusable form parameter to normalize, got %#v", op.Parameters)
	}
}

func TestDocumentWarnsWhenSwaggerFormDataAssumesUrlencodedConsumes(t *testing.T) {
	t.Parallel()

	spec, err := Document(mustResolvedSwagger(t, `swagger: "2.0"
info:
  title: Swagger Demo
  version: 1.0.0
paths:
  /pets:
    post:
      parameters:
        - name: name
          in: formData
          type: string
      responses:
        "200":
          description: ok
`))
	if err != nil {
		t.Fatalf("Document returned error: %v", err)
	}

	if len(spec.Operations) != 1 || spec.Operations[0].FormBodyMediaType != "application/x-www-form-urlencoded" {
		t.Fatalf("expected assumed form body media type, got %#v", spec.Operations)
	}
	if !hasWarningContaining(spec.Warnings, "assumed application/x-www-form-urlencoded") {
		t.Fatalf("expected consumes assumption warning, got %#v", spec.Warnings)
	}
}

func TestDocumentNormalisesMultipartSwaggerFormData(t *testing.T) {
	t.Parallel()

	spec, err := Document(mustResolvedSwagger(t, `swagger: "2.0"
info:
  title: Swagger Demo
  version: 1.0.0
paths:
  /pets:
    post:
      consumes: [multipart/form-data]
      parameters:
        - name: name
          in: formData
          type: string
      responses:
        "200":
          description: ok
`))
	if err != nil {
		t.Fatalf("Document returned error: %v", err)
	}

	op := spec.Operations[0]
	if op.FormBodyMediaType != "multipart/form-data" {
		t.Fatalf("expected multipart form body media type, got %q", op.FormBodyMediaType)
	}
	if len(op.Parameters) != 1 || op.Parameters[0].In != model.ParameterLocationForm {
		t.Fatalf("expected multipart form parameter, got %#v", op.Parameters)
	}
}

func TestDocumentNormalisesSwaggerFileUploadParameter(t *testing.T) {
	t.Parallel()

	spec, err := Document(mustResolvedSwagger(t, `swagger: "2.0"
info:
  title: Swagger Demo
  version: 1.0.0
paths:
  /upload:
    post:
      consumes: [multipart/form-data]
      parameters:
        - name: file
          in: formData
          required: true
          type: file
      responses:
        "200":
          description: ok
`))
	if err != nil {
		t.Fatalf("Document returned error: %v", err)
	}

	op := spec.Operations[0]
	if op.FormBodyMediaType != "multipart/form-data" {
		t.Fatalf("expected multipart form body media type, got %q", op.FormBodyMediaType)
	}
	if len(op.Parameters) != 1 || op.Parameters[0].FormInputKind != model.FormInputKindFile {
		t.Fatalf("expected file upload parameter to normalize, got %#v", op.Parameters)
	}
}

func TestDocumentNormalisesEquivalentSwaggerAndOAS3Shapes(t *testing.T) {
	t.Parallel()

	oasSpec, err := Document(mustResolvedOpenAPI3(t, `openapi: 3.0.3
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
`))
	if err != nil {
		t.Fatalf("normalise oas3: %v", err)
	}
	swaggerSpec, err := Document(mustResolvedSwagger(t, `swagger: "2.0"
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
`))
	if err != nil {
		t.Fatalf("normalise swagger: %v", err)
	}

	if len(oasSpec.Operations) != len(swaggerSpec.Operations) {
		t.Fatalf("expected matching operation counts, got %d and %d", len(oasSpec.Operations), len(swaggerSpec.Operations))
	}
	if oasSpec.Operations[0].Key != swaggerSpec.Operations[0].Key {
		t.Fatalf("expected matching normalised operation keys, got %q and %q", oasSpec.Operations[0].Key, swaggerSpec.Operations[0].Key)
	}
	if oasSpec.Servers[0].URL != swaggerSpec.Servers[0].URL {
		t.Fatalf("expected matching normalised server urls, got %q and %q", oasSpec.Servers[0].URL, swaggerSpec.Servers[0].URL)
	}
}

func TestDocumentMergesPathLevelReusableParameterRefsWithOperationOverrides(t *testing.T) {
	t.Parallel()

	spec, err := Document(mustResolvedSwagger(t, `swagger: "2.0"
info:
  title: Swagger Demo
  version: 1.0.0
parameters:
  Limit:
    name: limit
    in: query
    type: string
paths:
  /pets:
    parameters:
      - $ref: "#/parameters/Limit"
    get:
      parameters:
        - name: limit
          in: query
          type: integer
      responses:
        "200":
          description: ok
`))
	if err != nil {
		t.Fatalf("Document returned error: %v", err)
	}

	op := spec.Operations[0]
	if len(op.Parameters) != 1 {
		t.Fatalf("expected merged parameter list, got %#v", op.Parameters)
	}
	if got := op.Parameters[0].Schema.Type; got != "integer" {
		t.Fatalf("expected operation-level parameter override to win, got %#v", op.Parameters[0])
	}
}

func TestDocumentDerivesCapabilitiesAndWarnings(t *testing.T) {
	t.Parallel()

	spec, err := Document(mustResolvedOpenAPI3(t, `openapi: 3.0.3
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
`))
	if err != nil {
		t.Fatalf("Document returned error: %v", err)
	}
	if !spec.Capabilities.SupportsCookieParameters {
		t.Fatal("expected cookie parameter capability")
	}
	if !spec.Capabilities.SupportsSecuritySchemes {
		t.Fatal("expected security scheme capability")
	}
	if len(spec.Warnings) == 0 {
		t.Fatal("expected normalisation warnings")
	}
}

func TestDocumentUsesMostSpecificServerOverride(t *testing.T) {
	t.Parallel()

	spec, err := Document(mustResolvedOpenAPI3(t, `openapi: 3.0.3
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
`))
	if err != nil {
		t.Fatalf("Document returned error: %v", err)
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

func TestDocumentAppliesParameterOverrideSemantics(t *testing.T) {
	t.Parallel()

	spec, err := Document(mustResolvedOpenAPI3(t, `openapi: 3.0.3
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
`))
	if err != nil {
		t.Fatalf("Document returned error: %v", err)
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

func TestDocumentDerivesSourceAwareCapabilities(t *testing.T) {
	t.Parallel()

	oasSpec, err := Document(mustResolvedOpenAPI3(t, `openapi: 3.0.3
info:
  title: Demo
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
`))
	if err != nil {
		t.Fatalf("normalise oas3: %v", err)
	}
	swaggerSpec, err := Document(mustResolvedSwagger(t, `swagger: "2.0"
info:
  title: Demo
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
`))
	if err != nil {
		t.Fatalf("normalise swagger: %v", err)
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

func TestDocumentUsesCanonicalFingerprintAcrossFormatsAndFormatting(t *testing.T) {
	t.Parallel()

	yamlSpec, err := Document(mustResolvedOpenAPI3(t, `# comment
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
`))
	if err != nil {
		t.Fatalf("normalise yaml: %v", err)
	}
	jsonSpec, err := Document(mustResolvedOpenAPI3(t, `{"openapi":"3.0.3","info":{"title":"Demo","version":"1.0.0"},"paths":{"/pets":{"get":{"responses":{"200":{"description":"ok"}}}}}}`))
	if err != nil {
		t.Fatalf("normalise json: %v", err)
	}
	yamlVariantSpec, err := Document(mustResolvedOpenAPI3(t, `
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
`))
	if err != nil {
		t.Fatalf("normalise yaml variant: %v", err)
	}

	if yamlSpec.Fingerprint != jsonSpec.Fingerprint {
		t.Fatalf("expected YAML and JSON fingerprints to match, got %q and %q", yamlSpec.Fingerprint, jsonSpec.Fingerprint)
	}
	if yamlSpec.Fingerprint != yamlVariantSpec.Fingerprint {
		t.Fatalf("expected formatting-only variant fingerprint to match, got %q and %q", yamlSpec.Fingerprint, yamlVariantSpec.Fingerprint)
	}
}

func TestDocumentUsesSourceAwareFingerprintAcrossEquivalentSwaggerAndOAS3(t *testing.T) {
	t.Parallel()

	oasSpec, err := Document(mustResolvedOpenAPI3(t, `openapi: 3.0.3
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
`))
	if err != nil {
		t.Fatalf("normalise oas3: %v", err)
	}
	swaggerSpec, err := Document(mustResolvedSwagger(t, `swagger: "2.0"
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
`))
	if err != nil {
		t.Fatalf("normalise swagger: %v", err)
	}

	if oasSpec.Fingerprint == swaggerSpec.Fingerprint {
		t.Fatalf("expected source-aware fingerprints to differ, both were %q", oasSpec.Fingerprint)
	}
}

func TestDocumentPreservesParameterContent(t *testing.T) {
	t.Parallel()

	spec, err := Document(mustResolvedOpenAPI3(t, `openapi: 3.0.3
info:
  title: Demo
  version: 1.0.0
paths:
  /pets:
    get:
      parameters:
        - name: filter
          in: query
          content:
            application/json:
              schema:
                type: object
                properties:
                  status:
                    type: string
              example:
                status: active
      responses:
        "200":
          description: ok
`))
	if err != nil {
		t.Fatalf("Document returned error: %v", err)
	}

	param := spec.Operations[0].Parameters[0]
	if len(param.Content) != 1 {
		t.Fatalf("expected parameter content to be preserved, got %#v", param)
	}
	if got := param.SelectedContentType; got != "application/json" {
		t.Fatalf("expected selected parameter content type application/json, got %q", got)
	}
	if param.Example != nil || param.Default != nil {
		t.Fatalf("expected content-based parameter example/default to stay in content, got %#v", param)
	}
}

func TestDocumentPreservesResponseHeaderContent(t *testing.T) {
	t.Parallel()

	spec, err := Document(mustResolvedOpenAPI3(t, `openapi: 3.0.3
info:
  title: Demo
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
          headers:
            X-Filter:
              content:
                application/json:
                  schema:
                    type: array
                    items:
                      type: string
`))
	if err != nil {
		t.Fatalf("Document returned error: %v", err)
	}

	header := spec.Operations[0].Responses[0].Headers[0]
	if len(header.Content) != 1 {
		t.Fatalf("expected header content to be preserved, got %#v", header)
	}
	if got := header.SelectedContentType; got != "application/json" {
		t.Fatalf("expected selected header content type application/json, got %q", got)
	}
}

func TestDocumentPreservesSwaggerDowngradeWarnings(t *testing.T) {
	t.Parallel()

	spec, err := Document(mustResolvedSwagger(t, `swagger: "2.0"
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
      parameters:
        - name: ids
          in: query
          type: array
          items:
            type: string
          collectionFormat: tsv
      responses:
        "200":
          description: ok
`))
	if err != nil {
		t.Fatalf("Document returned error: %v", err)
	}

	param := spec.Operations[0].Parameters[0]
	if got := param.CollectionFormat; got != "tsv" {
		t.Fatalf("expected collectionFormat tsv to be preserved, got %q", got)
	}
	if !hasWarningContaining(spec.Warnings, "collectionFormat") {
		t.Fatalf("expected collectionFormat warning, got %#v", spec.Warnings)
	}
	if !hasWarningContaining(spec.Warnings, "security scheme") {
		t.Fatalf("expected oauth security warning, got %#v", spec.Warnings)
	}
}

func TestDocumentNormalisesEquivalentSwaggerAndOAS3SerialisationShapes(t *testing.T) {
	t.Parallel()

	oasSpec, err := Document(mustResolvedOpenAPI3(t, `openapi: 3.0.3
info:
  title: Demo
  version: 1.0.0
paths:
  /pets:
    get:
      parameters:
        - name: ids
          in: query
          style: spaceDelimited
          explode: false
          schema:
            type: array
            items:
              type: string
      responses:
        "200":
          description: ok
`))
	if err != nil {
		t.Fatalf("normalise oas3: %v", err)
	}
	swaggerSpec, err := Document(mustResolvedSwagger(t, `swagger: "2.0"
info:
  title: Demo
  version: 1.0.0
paths:
  /pets:
    get:
      parameters:
        - name: ids
          in: query
          type: array
          items:
            type: string
          collectionFormat: ssv
      responses:
        "200":
          description: ok
`))
	if err != nil {
		t.Fatalf("normalise swagger: %v", err)
	}

	oasParam := oasSpec.Operations[0].Parameters[0]
	swaggerParam := swaggerSpec.Operations[0].Parameters[0]
	if oasParam.Style != swaggerParam.Style {
		t.Fatalf("expected matching style, got %q and %q", oasParam.Style, swaggerParam.Style)
	}
	if (oasParam.Explode == nil) != (swaggerParam.Explode == nil) {
		t.Fatalf("expected matching explode pointers, got %v and %v", oasParam.Explode, swaggerParam.Explode)
	}
	if oasParam.Explode != nil && swaggerParam.Explode != nil && *oasParam.Explode != *swaggerParam.Explode {
		t.Fatalf("expected matching explode values, got %v and %v", *oasParam.Explode, *swaggerParam.Explode)
	}
	if swaggerParam.CollectionFormat != "" {
		t.Fatalf("expected exact Swagger mapping to avoid preserved collectionFormat, got %q", swaggerParam.CollectionFormat)
	}
}

func TestDocumentFingerprintChangesWhenParameterContentChanges(t *testing.T) {
	t.Parallel()

	jsonSpec, err := Document(mustResolvedOpenAPI3(t, `openapi: 3.0.3
info:
  title: Demo
  version: 1.0.0
paths:
  /pets:
    get:
      parameters:
        - name: filter
          in: query
          content:
            application/json:
              schema:
                type: string
      responses:
        "200":
          description: ok
`))
	if err != nil {
		t.Fatalf("normalise json content spec: %v", err)
	}
	xmlSpec, err := Document(mustResolvedOpenAPI3(t, `openapi: 3.0.3
info:
  title: Demo
  version: 1.0.0
paths:
  /pets:
    get:
      parameters:
        - name: filter
          in: query
          content:
            application/xml:
              schema:
                type: string
      responses:
        "200":
          description: ok
`))
	if err != nil {
		t.Fatalf("normalise xml content spec: %v", err)
	}

	if jsonSpec.Fingerprint == xmlSpec.Fingerprint {
		t.Fatalf("expected fingerprints to differ when parameter content changes, both were %q", jsonSpec.Fingerprint)
	}
}

func TestDocumentFingerprintChangesWhenSwaggerCollectionFormatChanges(t *testing.T) {
	t.Parallel()

	csvSpec, err := Document(mustResolvedSwagger(t, `swagger: "2.0"
info:
  title: Demo
  version: 1.0.0
paths:
  /pets:
    get:
      parameters:
        - name: ids
          in: query
          type: array
          items:
            type: string
          collectionFormat: csv
      responses:
        "200":
          description: ok
`))
	if err != nil {
		t.Fatalf("normalise csv spec: %v", err)
	}
	tsvSpec, err := Document(mustResolvedSwagger(t, `swagger: "2.0"
info:
  title: Demo
  version: 1.0.0
paths:
  /pets:
    get:
      parameters:
        - name: ids
          in: query
          type: array
          items:
            type: string
          collectionFormat: tsv
      responses:
        "200":
          description: ok
`))
	if err != nil {
		t.Fatalf("normalise tsv spec: %v", err)
	}

	if csvSpec.Fingerprint == tsvSpec.Fingerprint {
		t.Fatalf("expected fingerprints to differ when collectionFormat changes, both were %q", csvSpec.Fingerprint)
	}
}

func mustResolvedOpenAPI3(t *testing.T, raw string) *pipeline.ResolvedDocument {
	t.Helper()

	doc := mustOpenAPI3Doc(t, raw)
	return &pipeline.ResolvedDocument{
		BaseDocument: pipeline.BaseDocument{
			Document:      &pipeline.LoadedDocument{CanonicalLocation: "spec.yaml", Format: pipeline.DocumentFormatYAML},
			SourceFamily:  model.SourceFamilyOpenAPI3,
			SourceVersion: strings.TrimSpace(doc.OpenAPI),
			OpenAPI3Doc:   doc,
		},
	}
}

func mustResolvedSwagger(t *testing.T, raw string) *pipeline.ResolvedDocument {
	t.Helper()

	parsed := mustParsedSwagger(t, raw)
	converted, err := converter.Convert(parsed)
	if err != nil {
		t.Fatalf("converter.Convert: %v", err)
	}
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	if err := loader.ResolveRefsIn(converted.OpenAPI3Doc, nil); err != nil {
		t.Fatalf("loader.ResolveRefsIn: %v", err)
	}

	return &pipeline.ResolvedDocument{BaseDocument: converted.BaseDocument}
}

func mustParsedSwagger(t *testing.T, raw string) *pipeline.ParsedDocument {
	t.Helper()

	var decoded map[string]any
	if err := yaml.Unmarshal([]byte(raw), &decoded); err != nil {
		t.Fatalf("yaml.Unmarshal: %v", err)
	}

	return &pipeline.ParsedDocument{
		BaseDocument: pipeline.BaseDocument{
			Document:      &pipeline.LoadedDocument{CanonicalLocation: "swagger.yaml", Format: pipeline.DocumentFormatYAML},
			SourceFamily:  model.SourceFamilySwagger2,
			SourceVersion: strings.TrimSpace(decoded["swagger"].(string)),
		},
		SwaggerDoc: decoded,
	}
}

func mustOpenAPI3Doc(t *testing.T, raw string) *openapi3.T {
	t.Helper()

	var decoded map[string]any
	var err error
	if strings.HasPrefix(strings.TrimSpace(raw), "{") {
		err = json.Unmarshal([]byte(raw), &decoded)
	} else {
		err = yaml.Unmarshal([]byte(raw), &decoded)
	}
	if err != nil {
		t.Fatalf("decode spec: %v", err)
	}

	payload, err := json.Marshal(decoded)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var doc openapi3.T
	if err := json.Unmarshal(payload, &doc); err != nil {
		t.Fatalf("json.Unmarshal openapi3: %v", err)
	}

	return &doc
}

func hasWarningContaining(warnings []model.SpecWarning, substring string) bool {
	for _, warning := range warnings {
		if strings.Contains(warning.Message, substring) {
			return true
		}
	}
	return false
}
