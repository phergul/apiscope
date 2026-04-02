package schemaexplorer

import (
	"strings"
	"testing"

	"github.com/phergul/apiscope/internal/model"

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
	for _, snippet := range []string{"├─", "└─", "│  "} {
		if !strings.Contains(content, snippet) {
			t.Fatalf("expected tree to include %q, got %q", snippet, content)
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
