package panes

import (
	"strings"
	"testing"
)

func TestRenderRequestShowsGroupedInputsAndAuthSummary(t *testing.T) {
	t.Parallel()

	requestContent := RenderRequest(RequestData{
		Sections: []Section{
			{
				Label: "Path",
				Body: strings.Join([]string{
					"- petId (required, string)",
					"  Description: Unique user identifier",
				}, "\n"),
			},
			{
				Label: "Query",
				Body: strings.Join([]string{
					"- limit (optional, integer/int32)",
				}, "\n"),
			},
			{
				Label: "Body",
				Body: strings.Join([]string{
					"Required: required",
					"Media types: application/json, application/xml",
					"Description: Pet filter payload",
				}, "\n"),
			},
			{
				Label: "Auth",
				Body:  "- api_key\nOR\n- oauth (pets:read)",
			},
		},
		ActiveSection: "Path",
	})

	wantRequestSnippets := []string{
		"[Path]  Query  Body  Auth",
		"- petId (required, string)",
		"Description: Unique user identifier",
	}
	for _, snippet := range wantRequestSnippets {
		if !strings.Contains(requestContent, snippet) {
			t.Fatalf("expected request pane to include %q, got %q", snippet, requestContent)
		}
	}
}

func TestRenderRequestShowsExplicitEmptyStates(t *testing.T) {
	t.Parallel()

	requestContent := RenderRequest(RequestData{
		EmptyState: "This operation does not declare request parameters, request body, or auth requirements.",
	})

	if !strings.Contains(requestContent, "This operation does not declare request parameters, request body, or auth requirements.") {
		t.Fatalf("expected request pane empty state, got %q", requestContent)
	}
}

func TestRenderResponseShowsDeclaredResponses(t *testing.T) {
	t.Parallel()

	responseContent := RenderResponse(ResponseData{
		Sections: []Section{
			{
				Label: "200",
				Body: strings.Join([]string{
					"Description: OK",
					"Headers:",
					"- X-Rate-Limit (integer)",
					"- X-Trace-ID (string)",
					"Media types: application/json",
				}, "\n"),
			},
			{
				Label: "default",
				Body: strings.Join([]string{
					"Description: Unexpected error",
					"Headers:",
					"- none",
					"Media types: application/problem+json",
				}, "\n"),
			},
		},
		ActiveSection: "200",
	})

	wantResponseSnippets := []string{
		"[200]  default",
		"Description: OK",
		"Headers:",
		"- X-Rate-Limit (integer)",
		"- X-Trace-ID (string)",
		"Media types: application/json",
	}
	for _, snippet := range wantResponseSnippets {
		if !strings.Contains(responseContent, snippet) {
			t.Fatalf("expected response pane to include %q, got %q", snippet, responseContent)
		}
	}
}

func TestRenderResponseShowsExplicitEmptyState(t *testing.T) {
	t.Parallel()

	responseContent := RenderResponse(ResponseData{
		EmptyState: "This operation does not declare any responses.",
	})

	if !strings.Contains(responseContent, "This operation does not declare any responses.") {
		t.Fatalf("expected response pane empty state, got %q", responseContent)
	}
}
