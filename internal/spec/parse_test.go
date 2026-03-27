package spec

import (
	"context"
	"testing"

	"github.com/phergul/apiscope/internal/model"
	"github.com/phergul/apiscope/internal/spec/internal/pipeline"
)

func TestParseDocumentDecodesJSONOpenAPI3(t *testing.T) {
	t.Parallel()

	document := &pipeline.LoadedDocument{
		CanonicalLocation: "spec.json",
		Format:            pipeline.DocumentFormatJSON,
		Raw:               []byte(`{"openapi":"3.0.3","info":{"title":"Demo","version":"1.0.0"},"paths":{}}`),
	}

	parsed, err := NewLoader(nil).parseDocument(document)
	if err != nil {
		t.Fatalf("parseDocument returned error: %v", err)
	}

	if parsed.SourceFamily != model.SourceFamilyOpenAPI3 {
		t.Fatalf("expected openapi3 family, got %q", parsed.SourceFamily)
	}
	if parsed.SourceVersion != "3.0.3" {
		t.Fatalf("expected version 3.0.3, got %q", parsed.SourceVersion)
	}
	if parsed.OpenAPI3Doc == nil {
		t.Fatal("expected openapi3 document to be populated")
	}
}

func TestParseDocumentDecodesYAMLSwagger2(t *testing.T) {
	t.Parallel()

	document := &pipeline.LoadedDocument{
		CanonicalLocation: "spec.yaml",
		Format:            pipeline.DocumentFormatYAML,
		Raw:               []byte("swagger: '2.0'\ninfo:\n  title: Demo\n  version: 1.0.0\npaths: {}\n"),
	}

	parsed, err := NewLoader(nil).parseDocument(document)
	if err != nil {
		t.Fatalf("parseDocument returned error: %v", err)
	}

	if parsed.SourceFamily != model.SourceFamilySwagger2 {
		t.Fatalf("expected swagger2 family, got %q", parsed.SourceFamily)
	}
	if parsed.SourceVersion != "2.0" {
		t.Fatalf("expected version 2.0, got %q", parsed.SourceVersion)
	}
	if parsed.SwaggerDoc == nil {
		t.Fatal("expected swagger document to be preserved")
	}
}

func TestParseDocumentRejectsUnknownSpecFamily(t *testing.T) {
	t.Parallel()

	document := &pipeline.LoadedDocument{
		CanonicalLocation: "spec.yaml",
		Format:            pipeline.DocumentFormatYAML,
		Raw:               []byte("info:\n  title: Demo\npaths: {}\n"),
	}

	_, err := NewLoader(nil).parseDocument(document)
	if !IsErrorKind(err, ErrorKindUnsupportedFamily) {
		t.Fatalf("expected unsupported family error, got %v", err)
	}
}

func TestParseDocumentRejectsUnsupportedOpenAPIVersion(t *testing.T) {
	t.Parallel()

	document := &pipeline.LoadedDocument{
		CanonicalLocation: "spec.yaml",
		Format:            pipeline.DocumentFormatYAML,
		Raw:               []byte("openapi: 2.1.0\ninfo:\n  title: Demo\npaths: {}\n"),
	}

	_, err := NewLoader(nil).parseDocument(document)
	if !IsErrorKind(err, ErrorKindUnsupportedVersion) {
		t.Fatalf("expected unsupported version error, got %v", err)
	}
}

func TestParseDocumentRejectsMalformedOpenAPI3(t *testing.T) {
	t.Parallel()

	document := &pipeline.LoadedDocument{
		CanonicalLocation: "spec.yaml",
		Format:            pipeline.DocumentFormatYAML,
		Raw:               []byte("openapi: 3.0.3\ninfo: broken\npaths: {}\n"),
	}

	_, err := NewLoader(nil).parseDocument(document)
	if !IsErrorKind(err, ErrorKindOpenAPIParseFailure) {
		t.Fatalf("expected openapi parse failure, got %v", err)
	}
}

func TestLoadReturnsDecodeErrorForInvalidSyntax(t *testing.T) {
	t.Parallel()

	path := writeTempSpecFile(t, "invalid.yaml", "openapi: 3.0.3\ninfo: [broken\n")

	_, err := NewLoader(nil).Load(context.Background(), Source{Value: path})
	if !IsErrorKind(err, ErrorKindDecodeFailure) {
		t.Fatalf("expected decode failure, got %v", err)
	}
}
