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

func TestProjectIncludesOperationScopedSchemaEntrypoints(t *testing.T) {
	t.Parallel()

	projected := Project(ProjectionInput{
		Operation:     schemaOperationFixture(),
		State:         OpenState(schemaOperationFixture()),
		ContentWidth:  140,
		ContentHeight: 12,
	})

	content := ansi.Strip(projected.Data.LeftBody)
	for _, snippet := range []string{
		"Query param: limit",
		"(integer)",
		"Query content: filter (application/json)",
		"(object)",
		"Request body: application/json",
		"Response 200: application/json",
		"Response 200 header: X-Next",
		"(string)",
	} {
		if !strings.Contains(content, snippet) {
			t.Fatalf("expected entrypoint list to include %q, got %q", snippet, content)
		}
	}

	if strings.Index(content, "Query param: limit") > strings.Index(content, "Request body: application/json") {
		t.Fatalf("expected parameter entrypoints before request body entrypoints, got %q", content)
	}
}

func TestUpdateDrillsIntoChildrenAndBacktracks(t *testing.T) {
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

	state := OpenState(operation)
	result := Update(operation, state, UpdateInput{Key: "enter", VisibleRows: 8})
	if len(result.State.Breadcrumbs) != 1 {
		t.Fatalf("expected one breadcrumb after drilling into request body, got %#v", result.State.Breadcrumbs)
	}

	projected := Project(ProjectionInput{
		Operation:     operation,
		State:         result.State,
		ContentWidth:  90,
		ContentHeight: 10,
	})
	content := ansi.Strip(projected.Data.LeftBody)
	if strings.Index(content, "alpha (string)") > strings.Index(content, "beta (string)") {
		t.Fatalf("expected child properties to sort deterministically, got %q", content)
	}

	back := Update(operation, result.State, UpdateInput{Key: "backspace", VisibleRows: 8})
	if len(back.State.Breadcrumbs) != 0 {
		t.Fatalf("expected backspace to return to root, got %#v", back.State.Breadcrumbs)
	}
}

func TestProjectLeafNodeShowsEmptyStateAndPreview(t *testing.T) {
	t.Parallel()

	operation := &model.Operation{
		Key: model.NewOperationKey("POST", "/pets"),
		RequestBody: &model.RequestBodySpec{
			Content: []model.MediaTypeSpec{{
				MediaType: "application/json",
				Schema: &model.Schema{
					Type:        "string",
					Description: "Pet name",
					Example:     "fido",
				},
			}},
		},
	}

	state := Update(operation, OpenState(operation), UpdateInput{Key: "enter", VisibleRows: 8}).State
	projected := Project(ProjectionInput{
		Operation:     operation,
		State:         state,
		ContentWidth:  140,
		ContentHeight: 10,
	})

	left := ansi.Strip(projected.Data.LeftBody)
	if !strings.Contains(left, "No nested schemas.") {
		t.Fatalf("expected leaf node empty state, got %q", left)
	}

	right := ansi.Strip(projected.Data.RightBody)
	for _, snippet := range []string{"Type: string", "Description:", "Pet name", "Example:", "fido"} {
		if !strings.Contains(right, snippet) {
			t.Fatalf("expected preview to include %q, got %q", snippet, right)
		}
	}
}

func TestProjectMarksRecursiveRowsAsNonDrillable(t *testing.T) {
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
		ContentWidth:  90,
		ContentHeight: 10,
	})

	content := ansi.Strip(projected.Data.LeftBody)
	if !strings.Contains(content, "owner (object) - recursive") || !strings.Contains(content, "reference") {
		t.Fatalf("expected recursive child to be marked, got %q", content)
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
