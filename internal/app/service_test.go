package app

import (
	"context"
	"errors"
	"testing"

	"api-tui/internal/model"
	"api-tui/internal/spec"
)

type stubLoader struct {
	result    *model.APISpec
	err       error
	gotSource spec.Source
}

func (l *stubLoader) Load(ctx context.Context, source spec.Source) (*model.APISpec, error) {
	l.gotSource = source
	if l.err != nil {
		return nil, l.err
	}

	return l.result, nil
}

func TestServiceLoadSourceInitializesSessionAndView(t *testing.T) {
	t.Parallel()

	loader := &stubLoader{
		result: &model.APISpec{
			Fingerprint: "spec-123",
			Title:       "Demo API",
			Servers: []model.Server{
				{URL: "https://api.example.com"},
			},
			Operations: []model.Operation{
				{Key: model.NewOperationKey("GET", "/pets"), Method: "GET", Path: "/pets"},
				{Key: model.NewOperationKey("POST", "/pets"), Method: "POST", Path: "/pets"},
			},
		},
	}

	result, err := NewService(loader).LoadSource(context.Background(), "spec.yaml")
	if err != nil {
		t.Fatalf("LoadSource returned error: %v", err)
	}

	if loader.gotSource.Value != "spec.yaml" {
		t.Fatalf("expected raw source to be forwarded, got %#v", loader.gotSource)
	}
	if result.Session.Spec == nil {
		t.Fatal("expected normalised spec to be stored in session")
	}
	if result.Session.SpecFingerprint != "spec-123" {
		t.Fatalf("expected fingerprint spec-123, got %q", result.Session.SpecFingerprint)
	}
	if result.Session.SelectedOperationKey != model.NewOperationKey("GET", "/pets") {
		t.Fatalf("expected first operation to be selected, got %q", result.Session.SelectedOperationKey)
	}
	if result.Session.SelectedServerURL != "https://api.example.com" {
		t.Fatalf("expected first server to be selected, got %q", result.Session.SelectedServerURL)
	}
	if result.View.FocusedPane != model.FocusedPaneOperations {
		t.Fatalf("expected operations pane focus, got %q", result.View.FocusedPane)
	}
	if result.View.ExpandedRightPane != model.FocusedPaneRequest {
		t.Fatalf("expected request pane to start expanded, got %q", result.View.ExpandedRightPane)
	}
	if result.View.ActiveEditorMode != model.EditorModeBrowse {
		t.Fatalf("expected browse mode, got %q", result.View.ActiveEditorMode)
	}
	if !result.View.OperationsPaneVisible {
		t.Fatal("expected operations pane to remain visible")
	}
	if result.View.ZoomedPane {
		t.Fatal("expected zoom mode to start disabled")
	}
	if len(result.View.VisibleOperationKeys) != 2 {
		t.Fatalf("expected 2 visible operations, got %d", len(result.View.VisibleOperationKeys))
	}
}

func TestServiceLoadSourceAllowsSpecsWithNoOperations(t *testing.T) {
	t.Parallel()

	loader := &stubLoader{
		result: &model.APISpec{
			Fingerprint: "empty-spec",
			Title:       "Empty API",
		},
	}

	result, err := NewService(loader).LoadSource(context.Background(), "empty.yaml")
	if err != nil {
		t.Fatalf("LoadSource returned error: %v", err)
	}

	if result.Session.SelectedOperationKey != "" {
		t.Fatalf("expected no selected operation, got %q", result.Session.SelectedOperationKey)
	}
	if len(result.View.VisibleOperationKeys) != 0 {
		t.Fatalf("expected no visible operations, got %d", len(result.View.VisibleOperationKeys))
	}
	if result.Session.RequestDrafts == nil {
		t.Fatal("expected request drafts map to be initialized")
	}
	if result.Session.AuthState == nil {
		t.Fatal("expected auth state map to be initialized")
	}
}

func TestServiceLoadSourceReturnsLoaderErrors(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("boom")
	loader := &stubLoader{err: wantErr}

	_, err := NewService(loader).LoadSource(context.Background(), "broken.yaml")
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
}
