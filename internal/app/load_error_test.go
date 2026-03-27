package app

import (
	"errors"
	"testing"

	"github.com/phergul/apiscope/internal/spec"
)

func TestDescribeLoadErrorInvalidSource(t *testing.T) {
	t.Parallel()

	view := DescribeLoadError(&spec.Error{
		Kind:   spec.ErrorKindInvalidSource,
		Source: "ftp://example.com/spec.yaml",
		Err:    errors.New("unsupported source scheme"),
	}, "")

	if view.Category != "invalid source" {
		t.Fatalf("expected invalid source category, got %q", view.Category)
	}
	if view.Title != "Invalid spec source" {
		t.Fatalf("unexpected title: %q", view.Title)
	}
	if view.Source != "ftp://example.com/spec.yaml" {
		t.Fatalf("unexpected source: %q", view.Source)
	}
}

func TestDescribeLoadErrorFileReadFailure(t *testing.T) {
	t.Parallel()

	view := DescribeLoadError(&spec.Error{
		Kind:   spec.ErrorKindFileReadFailure,
		Source: "/tmp/missing.yaml",
		Err:    errors.New("no such file"),
	}, "")

	if view.Category != "file read failure" {
		t.Fatalf("expected file read failure category, got %q", view.Category)
	}
	if view.Hint == "" {
		t.Fatal("expected recovery hint")
	}
}

func TestDescribeLoadErrorURLFetchFailureIncludesStatus(t *testing.T) {
	t.Parallel()

	view := DescribeLoadError(&spec.Error{
		Kind:       spec.ErrorKindURLFetchFailure,
		Source:     "https://example.com/spec.yaml",
		StatusCode: 502,
		Err:        errors.New("bad gateway"),
	}, "")

	if view.Category != "url fetch failure" {
		t.Fatalf("expected url fetch failure category, got %q", view.Category)
	}
	if view.Summary != "The remote spec request failed with HTTP status 502." {
		t.Fatalf("unexpected summary: %q", view.Summary)
	}
}

func TestDescribeLoadErrorParseFailure(t *testing.T) {
	t.Parallel()

	view := DescribeLoadError(&spec.Error{
		Kind:   spec.ErrorKindUnsupportedVersion,
		Source: "spec.yaml",
		Err:    errors.New("unsupported version"),
	}, "")

	if view.Category != "parse failure" {
		t.Fatalf("expected parse failure category, got %q", view.Category)
	}
	if view.Title != "Spec could not be parsed" {
		t.Fatalf("unexpected title: %q", view.Title)
	}
}

func TestDescribeLoadErrorUnsupportedContent(t *testing.T) {
	t.Parallel()

	view := DescribeLoadError(&spec.Error{
		Kind:   spec.ErrorKindRefResolutionFailure,
		Source: "spec.yaml",
		Err:    errors.New("bad ref"),
	}, "")

	if view.Category != "unsupported spec content" {
		t.Fatalf("expected unsupported content category, got %q", view.Category)
	}
	if view.Title != "Spec contains unsupported content" {
		t.Fatalf("unexpected title: %q", view.Title)
	}
}
