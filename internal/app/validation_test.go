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
			{Name: "name", In: model.ParameterLocationForm, Required: true},
			{Name: "file", In: model.ParameterLocationForm, FormInputKind: model.FormInputKindFile, Required: true},
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
	if len(result.MessagesForSection("Form")) != 2 {
		t.Fatalf("expected two form validation issues, got %d", len(result.MessagesForSection("Form")))
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
			{Name: "name", In: model.ParameterLocationForm, Required: true},
			{Name: "file", In: model.ParameterLocationForm, FormInputKind: model.FormInputKindFile, Required: true},
		},
		RequestBody: &model.RequestBodySpec{
			Required: true,
		},
	}
	draft := &model.RequestDraft{
		PathParams:     map[string]string{"petId": "abc"},
		FormParams:     map[string]string{"name": "fido"},
		FormFileParams: map[string]string{"file": "/tmp/demo.txt"},
		BodyMediaType:  "application/json",
		BodyRaw:        "{}",
	}

	result := ValidateRequest(operation, draft)
	if result.HasIssues() {
		t.Fatalf("expected validation to pass, got %#v", result.Issues)
	}
}

func TestValidateRequestReportsRequiredMultipartBodyFields(t *testing.T) {
	t.Parallel()

	operation := &model.Operation{
		RequestBody: &model.RequestBodySpec{
			Required: true,
			Content: []model.MediaTypeSpec{{
				MediaType: "multipart/form-data",
				Schema: &model.Schema{
					Type:     "object",
					Required: []string{"description", "file"},
					Properties: map[string]*model.Schema{
						"description": {Type: "string"},
						"file":        {Type: "string", Format: "binary"},
					},
				},
			}},
		},
	}

	result := ValidateRequest(operation, &model.RequestDraft{BodyMediaType: "multipart/form-data"})
	if len(result.MessagesForSection("Body")) != 2 {
		t.Fatalf("expected multipart body field issues, got %#v", result.Issues)
	}
	if _, ok := result.IssueForTarget("form:description"); !ok {
		t.Fatalf("expected description body-field issue, got %#v", result.Issues)
	}
	if _, ok := result.IssueForTarget("form:file"); !ok {
		t.Fatalf("expected file body-field issue, got %#v", result.Issues)
	}
	if _, ok := result.IssueForTarget(ValidationTargetBodyRaw); ok {
		t.Fatalf("expected raw body validation to be skipped for multipart fields, got %#v", result.Issues)
	}
}
