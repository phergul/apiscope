package app

import (
	"testing"

	"github.com/phergul/apiscope/internal/model"
)

func TestValidateRequestReportsRequiredParamAndBodyIssues(t *testing.T) {
	t.Parallel()

	operation := &model.Operation{
		Parameters: []model.Parameter{
			{Name: "petId", In: model.ParameterLocationPath, Required: true},
			{Name: "market", In: model.ParameterLocationQuery, Required: true},
		},
		RequestBody: &model.RequestBodySpec{
			Required: true,
			Content:  []model.MediaTypeSpec{{MediaType: "application/json"}},
		},
	}

	result := ValidateRequest(operation, &model.RequestDraft{})
	if !result.HasIssues() {
		t.Fatal("expected validation issues")
	}
	if len(result.MessagesForSection("Path")) != 1 {
		t.Fatalf("expected one path validation issue, got %d", len(result.MessagesForSection("Path")))
	}
	if len(result.MessagesForSection("Query")) != 1 {
		t.Fatalf("expected one query validation issue, got %d", len(result.MessagesForSection("Query")))
	}
	if len(result.MessagesForSection("Body")) != 2 {
		t.Fatalf("expected two body validation issues, got %d", len(result.MessagesForSection("Body")))
	}
}

func TestValidateRequestPassesWhenRequiredInputsArePresent(t *testing.T) {
	t.Parallel()

	operation := &model.Operation{
		Parameters: []model.Parameter{
			{Name: "petId", In: model.ParameterLocationPath, Required: true},
		},
		RequestBody: &model.RequestBodySpec{
			Required: true,
		},
	}
	draft := &model.RequestDraft{
		PathParams:    map[string]string{"petId": "abc"},
		BodyMediaType: "application/json",
		BodyRaw:       "{}",
	}

	result := ValidateRequest(operation, draft)
	if result.HasIssues() {
		t.Fatalf("expected validation to pass, got %#v", result.Issues)
	}
}
