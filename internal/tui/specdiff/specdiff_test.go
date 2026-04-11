package specdiff

import (
	"strings"
	"testing"

	"github.com/phergul/apiscope/internal/app"
	"github.com/phergul/apiscope/internal/model"
)

func TestRenderIncludesOperationAndCapabilitySections(t *testing.T) {
	t.Parallel()

	body := Render(app.SpecDiffResult{
		Changed:           true,
		FromFingerprint:   "a",
		ToFingerprint:     "b",
		FromSourceFamily:  model.SourceFamilySwagger2,
		ToSourceFamily:    model.SourceFamilyOpenAPI3,
		FromSourceVersion: "2.0",
		ToSourceVersion:   "3.0.3",
		AddedOperations:   []model.OperationKey{model.NewOperationKey("GET", "/admin")},
		CapabilityChanges: []app.SpecDiffCapabilityChange{{Name: "SupportsOpenAPI3", From: false, To: true}},
	})

	for _, snippet := range []string{"Fingerprint: a -> b", "Added operations", "GET /admin", "Capability changes", "SupportsOpenAPI3"} {
		if !strings.Contains(body, snippet) {
			t.Fatalf("expected body to include %q, got %q", snippet, body)
		}
	}
}

func TestBuildHelpViewIncludesReloadAndCloseControls(t *testing.T) {
	t.Parallel()

	help := BuildHelpView()
	for _, snippet := range []string{"d open or close spec diff", "R or Ctrl+L reload spec", "Esc or q close spec diff"} {
		if !strings.Contains(help.Body, snippet) {
			t.Fatalf("expected spec diff help to include %q, got %q", snippet, help.Body)
		}
	}
}
