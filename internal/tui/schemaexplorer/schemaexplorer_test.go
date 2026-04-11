package schemaexplorer

import (
	"strings"
	"testing"

	"github.com/phergul/apiscope/internal/model"
	"github.com/phergul/apiscope/internal/tui/widgets"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func TestAvailableReturnsTrueWhenOperationHasReachableSchemas(t *testing.T) {
	t.Parallel()

	if !Available(schemaOperationFixture()) {
		t.Fatal("expected schema explorer to be available")
	}
	if Available(&model.Operation{Key: model.NewOperationKey("GET", "/empty")}) {
		t.Fatal("expected schema explorer to stay unavailable when no schemas exist")
	}
}

func TestProjectGroupsEntrypointsBySourceTypeAndStartsExpanded(t *testing.T) {
	t.Parallel()

	projected := Project(ProjectionInput{
		Operation:     schemaOperationFixture(),
		State:         OpenState(schemaOperationFixture()),
		ContentWidth:  220,
		ContentHeight: 14,
	})

	content := ansi.Strip(projected.Data.LeftBody)
	for _, snippet := range []string{
		"Query params",
		"Query param: limit (integer)",
		"Parameter content",
		"Query content: filter",
		"(object)",
		"Request bodies",
		"Request body: application/json (object)",
		"Responses",
		"Response 200: application/json (object)",
		"Response headers",
		"Response 200 header: X-Next (string)",
	} {
		if !strings.Contains(content, snippet) {
			t.Fatalf("expected grouped tree to include %q, got %q", snippet, content)
		}
	}

	if strings.Index(content, "Query params") > strings.Index(content, "Parameter content") {
		t.Fatalf("expected query params before parameter content, got %q", content)
	}
	if strings.Index(content, "Parameter content") > strings.Index(content, "Request bodies") {
		t.Fatalf("expected parameter content before request bodies, got %q", content)
	}
	if strings.Index(content, "Request bodies") > strings.Index(content, "Responses") {
		t.Fatalf("expected request bodies before responses, got %q", content)
	}
}

func TestUpdateTogglesInlineChildrenAndMarkers(t *testing.T) {
	t.Parallel()

	operation := &model.Operation{
		Key: model.NewOperationKey("POST", "/pets"),
		RequestBody: &model.RequestBodySpec{
			Content: []model.MediaTypeSpec{{
				MediaType: "application/json",
				Schema: &model.Schema{
					Type: "object",
					Properties: map[string]*model.Schema{
						"beta":  {Type: "string"},
						"alpha": {Type: "string"},
					},
				},
			}},
		},
	}

	initial := Project(ProjectionInput{
		Operation:     operation,
		State:         OpenState(operation),
		ContentWidth:  220,
		ContentHeight: 10,
	})
	initialLeft := ansi.Strip(initial.Data.LeftBody)
	if !strings.Contains(initialLeft, "[+] Request body:") {
		t.Fatalf("expected closed marker for request body entry, got %q", initialLeft)
	}
	if strings.Contains(initialLeft, "Property: alpha") {
		t.Fatalf("expected nested properties to stay hidden until expanded, got %q", initialLeft)
	}

	state := Update(operation, OpenState(operation), UpdateInput{Key: "enter", VisibleRows: 8}).State
	projected := Project(ProjectionInput{
		Operation:     operation,
		State:         state,
		ContentWidth:  220,
		ContentHeight: 12,
	})
	content := ansi.Strip(projected.Data.LeftBody)
	if !strings.Contains(content, "[-] Request body:") {
		t.Fatalf("expected open marker after expansion, got %q", content)
	}
	if strings.Index(content, "Property: alpha (string)") > strings.Index(content, "Property: beta (string)") {
		t.Fatalf("expected child properties to sort deterministically, got %q", content)
	}
}

func TestUpdateCollapseMovesSelectionToAncestor(t *testing.T) {
	t.Parallel()

	operation := &model.Operation{
		Key: model.NewOperationKey("POST", "/pets"),
		RequestBody: &model.RequestBodySpec{
			Content: []model.MediaTypeSpec{{
				MediaType: "application/json",
				Schema: &model.Schema{
					Type: "object",
					Properties: map[string]*model.Schema{
						"owner": {
							Type: "object",
							Properties: map[string]*model.Schema{
								"name": {Type: "string"},
							},
						},
					},
				},
			}},
		},
	}

	state := OpenState(operation)
	state = Update(operation, state, UpdateInput{Key: "enter", VisibleRows: 10}).State
	state = Update(operation, state, UpdateInput{Key: "j", VisibleRows: 10}).State
	state = Update(operation, state, UpdateInput{Key: "enter", VisibleRows: 10}).State
	state = Update(operation, state, UpdateInput{Key: "j", VisibleRows: 10}).State
	state = Update(operation, state, UpdateInput{Key: "left", VisibleRows: 10}).State

	projected := Project(ProjectionInput{
		Operation:     operation,
		State:         state,
		ContentWidth:  220,
		ContentHeight: 12,
	})
	left := ansi.Strip(projected.Data.LeftBody)
	rightTitle := projected.Data.RightTitle

	if strings.Contains(left, "Property: name (string)") {
		t.Fatalf("expected ancestor collapse to hide child rows, got %q", left)
	}
	if rightTitle != "Property: owner" {
		t.Fatalf("expected selection to move to collapsed ancestor, got %q", rightTitle)
	}
}

func TestProjectWrapsPreviewContentToPaneWidth(t *testing.T) {
	t.Parallel()

	operation := &model.Operation{
		Key: model.NewOperationKey("POST", "/pets"),
		RequestBody: &model.RequestBodySpec{
			Content: []model.MediaTypeSpec{{
				MediaType: "application/json",
				Schema: &model.Schema{
					Type:        "string",
					Description: "This description should wrap inside the preview pane instead of clipping off the right edge.",
					Example:     "fido",
				},
			}},
		},
	}

	projected := Project(ProjectionInput{
		Operation:     operation,
		State:         OpenState(operation),
		ContentWidth:  64,
		ContentHeight: 10,
	})
	right := ansi.Strip(projected.Data.RightBody)
	for line := range strings.SplitSeq(right, "\n") {
		if len(line) > projected.Data.RightWidth {
			t.Fatalf("expected wrapped preview lines to stay within width %d, got line %q", projected.Data.RightWidth, line)
		}
	}
}

func TestProjectMarksRecursiveRowsAsNonExpandable(t *testing.T) {
	t.Parallel()

	root := &model.Schema{
		Ref:  "#/components/schemas/Pet",
		Type: "object",
		Properties: map[string]*model.Schema{
			"owner": {
				Ref:  "#/components/schemas/Pet",
				Type: "object",
			},
		},
	}
	operation := &model.Operation{
		Key: model.NewOperationKey("POST", "/pets"),
		RequestBody: &model.RequestBodySpec{
			Content: []model.MediaTypeSpec{{
				MediaType: "application/json",
				Schema:    root,
			}},
		},
	}

	state := Update(operation, OpenState(operation), UpdateInput{Key: "enter", VisibleRows: 8}).State
	projected := Project(ProjectionInput{
		Operation:     operation,
		State:         state,
		ContentWidth:  220,
		ContentHeight: 10,
	})

	content := ansi.Strip(projected.Data.LeftBody)
	if !strings.Contains(content, "Property: owner (object)") || !strings.Contains(content, "recursive") {
		t.Fatalf("expected recursive child to be marked, got %q", content)
	}
}

func TestRenderUsesBoxDrawingTreeGuides(t *testing.T) {
	t.Parallel()

	operation := &model.Operation{
		Key: model.NewOperationKey("POST", "/pets"),
		RequestBody: &model.RequestBodySpec{
			Content: []model.MediaTypeSpec{{
				MediaType: "application/json",
				Schema: &model.Schema{
					Type: "object",
					Properties: map[string]*model.Schema{
						"count": {Type: "integer"},
						"owner": {
							Type: "object",
							Properties: map[string]*model.Schema{
								"name":   {Type: "string"},
								"status": {Type: "string"},
							},
						},
						"zeta": {Type: "string"},
					},
				},
			}},
		},
	}

	state := OpenState(operation)
	state = Update(operation, state, UpdateInput{Key: "enter", VisibleRows: 10}).State
	state = Update(operation, state, UpdateInput{Key: "j", VisibleRows: 10}).State
	state = Update(operation, state, UpdateInput{Key: "j", VisibleRows: 10}).State
	state = Update(operation, state, UpdateInput{Key: "enter", VisibleRows: 10}).State
	projected := Project(ProjectionInput{
		Operation:     operation,
		State:         state,
		ContentWidth:  220,
		ContentHeight: 12,
	})

	content := ansi.Strip(projected.Data.LeftBody)
	for _, snippet := range []string{"├─", "└─"} {
		if !strings.Contains(content, snippet) {
			t.Fatalf("expected tree to include %q, got %q", snippet, content)
		}
	}
}

func TestRenderTreePrefixPreservesDepthForAncestorsWithoutNextSibling(t *testing.T) {
	t.Parallel()

	prefix := ansi.Strip(renderTreePrefix(visibleRow{
		Depth:           3,
		AncestorHasNext: []bool{false, true, false},
		HasNextSibling:  false,
	}))

	if prefix != " │     └─" {
		t.Fatalf("expected depth-preserving prefix, got %q", prefix)
	}
}

func TestRenderTreePrefixDoesNotIndentFirstLevelByParentColumn(t *testing.T) {
	t.Parallel()

	prefix := ansi.Strip(renderTreePrefix(visibleRow{
		Depth:           1,
		AncestorHasNext: []bool{false},
		HasNextSibling:  true,
	}))

	if prefix != " ├─" {
		t.Fatalf("expected first-level rows to branch from marker center, got %q", prefix)
	}
}

func TestRenderTreePrefixShowsGuideWhenParentHasNextSibling(t *testing.T) {
	t.Parallel()

	prefix := ansi.Strip(renderTreePrefix(visibleRow{
		Depth:           2,
		AncestorHasNext: []bool{false, true},
		HasNextSibling:  false,
	}))

	if prefix != " │  └─" {
		t.Fatalf("expected parent guide column for continuing siblings, got %q", prefix)
	}
}

func TestSelectedRowStylesOnlyHighlightHeaderAndName(t *testing.T) {
	originalTheme := widgets.CurrentTheme()
	t.Cleanup(func() {
		widgets.SetTheme(originalTheme)
	})
	widgets.SetTheme(widgets.DefaultTheme())

	if headerRowStyle(true).GetBackground() != widgets.CurrentTheme().Palette.Selection {
		t.Fatalf("expected selected header background %q, got %q", widgets.CurrentTheme().Palette.Selection, headerRowStyle(true).GetBackground())
	}
	if nameRowStyle(true).GetBackground() != widgets.CurrentTheme().Palette.Selection {
		t.Fatalf("expected selected name background %q, got %q", widgets.CurrentTheme().Palette.Selection, nameRowStyle(true).GetBackground())
	}
	for label, background := range map[string]any{
		"tree marker": treeMarkerStyle().GetBackground(),
		"meta":        metaRowStyle().GetBackground(),
		"note":        noteRowStyle().GetBackground(),
	} {
		if hasBackground(background) {
			t.Fatalf("expected %s background to stay unset, got %#v", label, background)
		}
	}
}

func TestRenderUsesFullHeightDividerBetweenTreeAndPreview(t *testing.T) {
	t.Parallel()

	rendered := ansi.Strip(Render(Data{
		LeftTitle:  "Schemas",
		RightTitle: "Preview",
		LeftBody:   "left one\nleft two",
		RightBody:  "right one\nright two",
		LeftWidth:  14,
		RightWidth: 14,
	}))

	lines := strings.Split(rendered, "\n")
	if len(lines) < 4 {
		t.Fatalf("expected multi-line explorer render, got %q", rendered)
	}
	for _, line := range lines {
		if !strings.Contains(line, " │ ") {
			t.Fatalf("expected each render line to include the split divider, got %q", rendered)
		}
	}
}

func TestUpdateReturnsCloseActionOnEscape(t *testing.T) {
	t.Parallel()

	result := Update(schemaOperationFixture(), OpenState(schemaOperationFixture()), UpdateInput{
		Key:         "esc",
		VisibleRows: 8,
	})
	if !result.Action.Close {
		t.Fatal("expected escape to request explorer close")
	}
}

func schemaOperationFixture() *model.Operation {
	return &model.Operation{
		Key: model.NewOperationKey("GET", "/pets"),
		Parameters: []model.Parameter{
			{
				Name:   "limit",
				In:     model.ParameterLocationQuery,
				Schema: &model.Schema{Type: "integer"},
			},
			{
				Name: "filter",
				In:   model.ParameterLocationQuery,
				Content: []model.MediaTypeSpec{{
					MediaType: "application/json",
					Schema: &model.Schema{
						Type: "object",
						Properties: map[string]*model.Schema{
							"status": {Type: "string"},
						},
					},
				}},
			},
		},
		RequestBody: &model.RequestBodySpec{
			Content: []model.MediaTypeSpec{{
				MediaType: "application/json",
				Schema: &model.Schema{
					Type: "object",
					Properties: map[string]*model.Schema{
						"name": {Type: "string"},
					},
				},
			}},
		},
		Responses: []model.ResponseSpec{{
			StatusCode: "200",
			Content: []model.MediaTypeSpec{{
				MediaType: "application/json",
				Schema:    &model.Schema{Type: "object"},
			}},
			Headers: []model.Parameter{{
				Name:   "X-Next",
				In:     model.ParameterLocationHeader,
				Schema: &model.Schema{Type: "string"},
			}},
		}},
	}
}

func hasBackground(background any) bool {
	switch typed := background.(type) {
	case nil:
		return false
	case string:
		return strings.TrimSpace(typed) != ""
	case lipgloss.Color:
		return strings.TrimSpace(string(typed)) != ""
	case lipgloss.NoColor:
		return false
	default:
		return true
	}
}
