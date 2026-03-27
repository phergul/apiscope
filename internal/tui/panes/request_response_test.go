package panes

import (
	"strings"
	"testing"
)

func TestRenderRequestAndResponseExplainFutureOwnership(t *testing.T) {
	t.Parallel()

	requestContent := RenderRequest(RequestData{})
	if !strings.Contains(requestContent, "path/query/header params, auth, and request body input") {
		t.Fatalf("expected request pane copy to explain ownership, got %q", requestContent)
	}

	responseContent := RenderResponse(ResponseData{})
	if !strings.Contains(responseContent, "response details and examples") {
		t.Fatalf("expected response pane copy to explain ownership, got %q", responseContent)
	}
}
