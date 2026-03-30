package request

import (
	"testing"

	"github.com/phergul/apiscope/internal/model"
)

func TestMoveActiveSectionUsesAvailableRequestSections(t *testing.T) {
	t.Parallel()

	selected := &model.Operation{
		Parameters: []model.Parameter{
			{Name: "petId", In: model.ParameterLocationPath},
		},
		RequestBody: &model.RequestBodySpec{},
	}

	got := MoveActiveSection("Path", 1, selected, nil)
	if got != SectionBody {
		t.Fatalf("expected section movement to advance to Body, got %q", got)
	}
}
