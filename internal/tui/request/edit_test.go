package request

import (
	"testing"

	"github.com/phergul/apiscope/internal/app"
	"github.com/phergul/apiscope/internal/model"
)

func TestStartEditReturnsFieldEditorStateForEditableParameter(t *testing.T) {
	t.Parallel()

	parameter := model.Parameter{
		Name: "petId",
		In:   model.ParameterLocationPath,
	}
	got := StartEdit(
		&model.Operation{Parameters: []model.Parameter{parameter}},
		&model.RequestDraft{PathParams: map[string]string{"petId": "42"}},
		[]RowDescriptor{{
			ID:        "path:petId",
			Kind:      RowKindParameter,
			Parameter: &parameter,
			Editable:  true,
		}},
		0,
	)

	if got.Kind != model.RequestEditKindField {
		t.Fatalf("expected field edit kind, got %q", got.Kind)
	}
	if got.Buffer != "42" {
		t.Fatalf("expected draft value to seed edit buffer, got %q", got.Buffer)
	}
	if !got.FocusField {
		t.Fatal("expected field editor to request focus")
	}
}

func TestCycleBodyMediaTypeAdvancesDraftSelection(t *testing.T) {
	t.Parallel()

	selected := &model.Operation{
		Key: model.NewOperationKey("POST", "/pets"),
		RequestBody: &model.RequestBodySpec{
			Content: []model.MediaTypeSpec{
				{MediaType: "application/json"},
				{MediaType: "application/xml"},
			},
		},
	}
	session := model.SessionState{
		RequestDrafts: map[model.DraftKey]*model.RequestDraft{},
	}
	app.EnsureRequestDraft(&session, selected)

	ok := CycleBodyMediaType(&session, selected)
	if !ok {
		t.Fatal("expected body media type cycle to succeed")
	}

	draft := app.EnsureRequestDraft(&session, selected)
	if draft.BodyMediaType != "application/xml" {
		t.Fatalf("expected body media type to advance, got %q", draft.BodyMediaType)
	}
}

func TestSaveEditWritesBodyBufferToDraft(t *testing.T) {
	t.Parallel()

	selected := &model.Operation{Key: model.NewOperationKey("POST", "/pets")}
	session := model.SessionState{
		RequestDrafts: map[model.DraftKey]*model.RequestDraft{},
	}

	ok := SaveEdit(
		&session,
		selected,
		[]RowDescriptor{{ID: "body:raw", Kind: RowKindBodyText, Editable: true}},
		0,
		model.RequestEditKindBody,
		"{\"name\":\"fido\"}",
	)
	if !ok {
		t.Fatal("expected save to succeed")
	}

	draft := app.EnsureRequestDraft(&session, selected)
	if draft.BodyRaw != "{\"name\":\"fido\"}" {
		t.Fatalf("expected body draft to be saved, got %q", draft.BodyRaw)
	}
}
