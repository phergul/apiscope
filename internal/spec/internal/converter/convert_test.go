package converter

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/phergul/apiscope/internal/model"
	"github.com/phergul/apiscope/internal/spec/internal/pipeline"

	"github.com/getkin/kin-openapi/openapi3"
	"gopkg.in/yaml.v3"
)

func TestConvertPassesThroughOpenAPI3(t *testing.T) {
	t.Parallel()

	doc := mustOpenAPI3Doc(t, `{"openapi":"3.0.3","info":{"title":"Demo","version":"1.0.0"},"paths":{}}`)
	parsed := &pipeline.ParsedDocument{
		BaseDocument: pipeline.BaseDocument{
			Document:      &pipeline.LoadedDocument{CanonicalLocation: "spec.json", Format: pipeline.DocumentFormatJSON},
			SourceFamily:  model.SourceFamilyOpenAPI3,
			SourceVersion: "3.0.3",
			OpenAPI3Doc:   doc,
		},
	}

	converted, err := Convert(parsed)
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}

	if converted.SourceFamily != model.SourceFamilyOpenAPI3 {
		t.Fatalf("expected source family openapi3, got %q", converted.SourceFamily)
	}
	if converted.OpenAPI3Doc != parsed.OpenAPI3Doc {
		t.Fatal("expected openapi3 document to pass through unchanged")
	}
}

func TestConvertConvertsSwagger2CommonCase(t *testing.T) {
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

	converted, err := Convert(mustParsedSwagger(t, raw))
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}

	if converted.SourceFamily != model.SourceFamilySwagger2 {
		t.Fatalf("expected swagger2 source family, got %q", converted.SourceFamily)
	}
	if got := converted.OpenAPI3Doc.Servers[0].URL; got != "https://api.example.com/v1" {
		t.Fatalf("expected converted server, got %q", got)
	}
	if converted.OpenAPI3Doc.Paths == nil {
		t.Fatal("expected converted paths")
	}
	pathItem := converted.OpenAPI3Doc.Paths.Value("/pets")
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
	if converted.OpenAPI3Doc.Components == nil || converted.OpenAPI3Doc.Components.SecuritySchemes["api_key"] == nil {
		t.Fatal("expected security definition to be converted")
	}
	if len(converted.OpenAPI3Doc.Security) != 1 {
		t.Fatalf("expected top-level security to be preserved, got %d entries", len(converted.OpenAPI3Doc.Security))
	}
	response := pathItem.Get.Responses.Value("200")
	if response == nil || response.Value == nil {
		t.Fatal("expected response 200 to be converted")
	}
	if response.Value.Content["application/json"] == nil {
		t.Fatal("expected response media type from produces")
	}
}

func TestConvertRejectsUnsupportedSwaggerConstruct(t *testing.T) {
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
          type: file
      responses:
        "200":
          description: ok
`

	_, err := Convert(mustParsedSwagger(t, raw))
	if !isPipelineErrorKind(err, pipeline.ErrorKindUnsupportedSwaggerConstruct) {
		t.Fatalf("expected unsupported swagger construct error, got %v", err)
	}
}

func TestConvertSupportsUrlencodedSwaggerFormData(t *testing.T) {
	t.Parallel()

	raw := `swagger: "2.0"
info:
  title: Demo
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
`

	converted, err := Convert(mustParsedSwagger(t, raw))
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}

	pathItem := converted.OpenAPI3Doc.Paths.Value("/pets")
	if pathItem == nil || pathItem.Post == nil {
		t.Fatal("expected converted post operation")
	}
	if pathItem.Post.RequestBody != nil {
		t.Fatalf("expected formData to stay in parameters for pane 3 projection, got %#v", pathItem.Post.RequestBody)
	}
	if len(pathItem.Post.Parameters) != 1 {
		t.Fatalf("expected one converted form parameter, got %d", len(pathItem.Post.Parameters))
	}
	parameter := pathItem.Post.Parameters[0].Value
	if parameter == nil {
		t.Fatal("expected converted form parameter value")
	}
	if got := parameter.Extensions[pipeline.SwaggerParameterLocationExtension]; got != "formData" {
		t.Fatalf("expected swagger formData extension, got %#v", got)
	}
	if got := pathItem.Post.Extensions[pipeline.SwaggerFormBodyMediaTypeExtension]; got != "application/x-www-form-urlencoded" {
		t.Fatalf("expected form body media type extension, got %#v", got)
	}
}

func TestConvertSupportsReusableSwaggerFormDataParameterRef(t *testing.T) {
	t.Parallel()

	raw := `swagger: "2.0"
info:
  title: Demo
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
`

	converted, err := Convert(mustParsedSwagger(t, raw))
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}

	component := converted.OpenAPI3Doc.Components.Parameters["PetName"]
	if component == nil || component.Value == nil {
		t.Fatal("expected converted reusable form parameter")
	}
	if got := component.Value.Extensions[pipeline.SwaggerParameterLocationExtension]; got != "formData" {
		t.Fatalf("expected reusable form parameter extension, got %#v", got)
	}
	pathItem := converted.OpenAPI3Doc.Paths.Value("/pets")
	if pathItem == nil || pathItem.Post == nil {
		t.Fatal("expected converted post operation")
	}
	if got := pathItem.Post.Parameters[0].Ref; got != "#/components/parameters/PetName" {
		t.Fatalf("expected reusable parameter ref rewrite, got %q", got)
	}
	if got := pathItem.Post.Extensions[pipeline.SwaggerFormBodyMediaTypeExtension]; got != "application/x-www-form-urlencoded" {
		t.Fatalf("expected form body media type extension for reusable form ref, got %#v", got)
	}
}

func TestConvertRejectsUnsupportedReusableSwaggerParameterDefinition(t *testing.T) {
	t.Parallel()

	raw := `swagger: "2.0"
info:
  title: Demo
  version: 1.0.0
parameters:
  CookieToken:
    name: token
    in: cookie
    type: string
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
`

	_, err := Convert(mustParsedSwagger(t, raw))
	if !isPipelineErrorKind(err, pipeline.ErrorKindUnsupportedSwaggerConstruct) {
		t.Fatalf("expected unsupported swagger construct error, got %v", err)
	}
}

func TestConvertSupportsMultipartSwaggerFormData(t *testing.T) {
	t.Parallel()

	raw := `swagger: "2.0"
info:
  title: Demo
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
`

	converted, err := Convert(mustParsedSwagger(t, raw))
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}

	pathItem := converted.OpenAPI3Doc.Paths.Value("/pets")
	if got := pathItem.Post.Extensions[pipeline.SwaggerFormBodyMediaTypeExtension]; got != "multipart/form-data" {
		t.Fatalf("expected multipart form body media type extension, got %#v", got)
	}
}

func TestConvertRejectsMixedSwaggerBodyAndFormData(t *testing.T) {
	t.Parallel()

	raw := `swagger: "2.0"
info:
  title: Demo
  version: 1.0.0
paths:
  /pets:
    post:
      consumes: [application/x-www-form-urlencoded]
      parameters:
        - name: body
          in: body
          schema:
            type: object
        - name: name
          in: formData
          type: string
      responses:
        "200":
          description: ok
`

	_, err := Convert(mustParsedSwagger(t, raw))
	if !isPipelineErrorKind(err, pipeline.ErrorKindUnsupportedSwaggerConstruct) {
		t.Fatalf("expected unsupported swagger construct error, got %v", err)
	}
}

func TestConvertSupportsMultipartSwaggerFileUpload(t *testing.T) {
	t.Parallel()

	raw := `swagger: "2.0"
info:
  title: Demo
  version: 1.0.0
paths:
  /upload:
    post:
      consumes: [multipart/form-data]
      parameters:
        - name: description
          in: formData
          type: string
        - name: file
          in: formData
          type: file
      responses:
        "200":
          description: ok
`

	converted, err := Convert(mustParsedSwagger(t, raw))
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}

	pathItem := converted.OpenAPI3Doc.Paths.Value("/upload")
	if pathItem == nil || pathItem.Post == nil {
		t.Fatal("expected converted upload operation")
	}
	if got := pathItem.Post.Extensions[pipeline.SwaggerFormBodyMediaTypeExtension]; got != "multipart/form-data" {
		t.Fatalf("expected multipart form body media type extension, got %#v", got)
	}
	fileParam := pathItem.Post.Parameters[1].Value
	if fileParam == nil {
		t.Fatal("expected file form parameter")
	}
	if got := fileParam.Extensions[pipeline.SwaggerFormFileParameterExtension]; got != true {
		t.Fatalf("expected file form extension, got %#v", got)
	}
}

func TestConvertRejectsUrlencodedSwaggerFileUpload(t *testing.T) {
	t.Parallel()

	raw := `swagger: "2.0"
info:
  title: Demo
  version: 1.0.0
paths:
  /upload:
    post:
      consumes: [application/x-www-form-urlencoded]
      parameters:
        - name: file
          in: formData
          type: file
      responses:
        "200":
          description: ok
`

	_, err := Convert(mustParsedSwagger(t, raw))
	if !isPipelineErrorKind(err, pipeline.ErrorKindUnsupportedSwaggerConstruct) {
		t.Fatalf("expected unsupported swagger construct error, got %v", err)
	}
}

func TestConvertConvertsSwaggerReusableRefsAndResponseHeaders(t *testing.T) {
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

	converted, err := Convert(mustParsedSwagger(t, raw))
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}

	limit := converted.OpenAPI3Doc.Components.Parameters["Limit"]
	if limit == nil || limit.Value == nil {
		t.Fatal("expected converted reusable parameter")
	}
	if limit.Value.Style != "form" || limit.Value.Explode == nil || !*limit.Value.Explode {
		t.Fatalf("expected multi query collectionFormat to map to form+explode, got style=%q explode=%v", limit.Value.Style, limit.Value.Explode)
	}

	pathItem := converted.OpenAPI3Doc.Paths.Value("/pets")
	if pathItem == nil || pathItem.Get == nil {
		t.Fatal("expected converted get operation")
	}
	if got := pathItem.Get.Parameters[0].Ref; got != "#/components/parameters/Limit" {
		t.Fatalf("expected parameter ref rewrite, got %q", got)
	}
	if got := pathItem.Get.Responses.Default().Ref; got != "#/components/responses/Error" {
		t.Fatalf("expected response ref rewrite, got %q", got)
	}

	errorResponse := converted.OpenAPI3Doc.Components.Responses["Error"]
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

func TestConvertRewritesExternalSwaggerRefs(t *testing.T) {
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

	converted, err := Convert(mustParsedSwagger(t, raw))
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}

	pathItem := converted.OpenAPI3Doc.Paths.Value("/pets")
	if got := pathItem.Get.Parameters[0].Ref; got != "common.yaml#/components/parameters/Limit" {
		t.Fatalf("expected external parameter ref rewrite, got %q", got)
	}
	if got := pathItem.Get.Responses.Default().Ref; got != "https://example.com/common.yaml#/components/responses/Error" {
		t.Fatalf("expected external response ref rewrite, got %q", got)
	}
}

func TestConvertPreservesSwaggerPathItemRefs(t *testing.T) {
	t.Parallel()

	raw := `swagger: "2.0"
info:
  title: Demo
  version: 1.0.0
paths:
  /pets:
    $ref: "common.yaml#/paths/~1pets"
`

	converted, err := Convert(mustParsedSwagger(t, raw))
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}

	pathItem := converted.OpenAPI3Doc.Paths.Value("/pets")
	if pathItem == nil {
		t.Fatal("expected converted path item")
	}
	if got := pathItem.Ref; got != "common.yaml#/paths/~1pets" {
		t.Fatalf("expected path item ref to be preserved, got %q", got)
	}
}

func TestConvertConvertsSwaggerOAuth2Definitions(t *testing.T) {
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

	converted, err := Convert(mustParsedSwagger(t, raw))
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}

	scheme := converted.OpenAPI3Doc.Components.SecuritySchemes["petstore_auth"]
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

func TestConvertMapsSwaggerCollectionFormats(t *testing.T) {
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

	converted, err := Convert(mustParsedSwagger(t, raw))
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}

	params := converted.OpenAPI3Doc.Paths.Value("/pets").Get.Parameters
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
	if got := params[3].Value.Extensions[pipeline.SwaggerCollectionFormatExtension]; got != "tsv" {
		t.Fatalf("expected tsv to be preserved in extensions, got %#v", got)
	}
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

func isPipelineErrorKind(err error, kind pipeline.ErrorKind) bool {
	var pipelineErr *pipeline.Error
	return err != nil && kind != "" && errors.As(err, &pipelineErr) && pipelineErr.Kind == kind
}
