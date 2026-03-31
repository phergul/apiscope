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

	got := MoveActiveSection("Path", 1, selected, nil, nil)
	if got != SectionBody {
		t.Fatalf("expected section movement to advance to Body, got %q", got)
	}
}

func TestAvailableSectionsIncludesServerWhenMultipleTopLevelServersExist(t *testing.T) {
	t.Parallel()

	selected := &model.Operation{
		RequestBody: &model.RequestBodySpec{},
	}

	sections := AvailableSections(selected, nil, []model.Server{
		{URL: "https://api.example.com"},
		{URL: "https://staging.example.com"},
	})
	if len(sections) != 2 {
		t.Fatalf("expected server and body sections, got %#v", sections)
	}
	if sections[1] != SectionServer {
		t.Fatalf("expected server section last, got %#v", sections)
	}
}
