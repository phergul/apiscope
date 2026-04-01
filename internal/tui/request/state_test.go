package request

import (
	"testing"

	"github.com/phergul/apiscope/internal/model"
)

func TestSyncRowStateSkipsUnselectableRows(t *testing.T) {
	t.Parallel()

	rows := []RowDescriptor{
		{Kind: RowKindAuthOption, Label: "Option 1", Editable: false},
		{Kind: RowKindAuthField, Label: "basicAuth username", Editable: true},
		{Kind: RowKindAuthField, Label: "basicAuth password", Editable: true},
	}

	state := SyncRowState(rows, RowState{ActiveRow: 0}, model.RequestEditKindNone, 5)
	if state.ActiveRow != 1 {
		t.Fatalf("expected sync to move to first selectable row, got %d", state.ActiveRow)
	}
}

func TestMoveRowStateSkipsReadOnlyRows(t *testing.T) {
	t.Parallel()

	rows := []RowDescriptor{
		{Kind: RowKindParameter, Label: "legacy", Editable: false},
		{Kind: RowKindParameter, Label: "limit", Editable: true},
		{Kind: RowKindParameter, Label: "offset", Editable: true},
	}

	state := MoveRowState(rows, RowState{ActiveRow: 0}, 1, model.RequestEditKindNone, 5)
	if state.ActiveRow != 2 {
		t.Fatalf("expected movement to skip to next selectable row, got %d", state.ActiveRow)
	}
}

func TestBoundaryRowStateUsesSelectableRowsOnly(t *testing.T) {
	t.Parallel()

	rows := []RowDescriptor{
		{Kind: RowKindAuthOption, Editable: false},
		{Kind: RowKindAuthField, Editable: true},
		{Kind: RowKindAuthOption, Editable: false},
		{Kind: RowKindAuthField, Editable: true},
	}

	first := BoundaryRowState(rows, RowState{}, false, model.RequestEditKindNone, 5)
	if first.ActiveRow != 1 {
		t.Fatalf("expected home boundary to land on first selectable row, got %d", first.ActiveRow)
	}

	last := BoundaryRowState(rows, RowState{}, true, model.RequestEditKindNone, 5)
	if last.ActiveRow != 3 {
		t.Fatalf("expected end boundary to land on last selectable row, got %d", last.ActiveRow)
	}
}
