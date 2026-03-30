package request

import "github.com/phergul/apiscope/internal/model"

type RowState struct {
	ActiveRow    int
	ScrollOffset int
}

// ClampActiveRow clamps a request row index into the available row count.
func ClampActiveRow(activeRow, rowCount int) int {
	if rowCount <= 0 || activeRow < 0 {
		return 0
	}
	if activeRow >= rowCount {
		return rowCount - 1
	}

	return activeRow
}

// MoveActiveRow moves the active request row by the given delta.
func MoveActiveRow(activeRow, rowCount, direction int) int {
	return ClampActiveRow(activeRow+direction, rowCount)
}

// BoundaryActiveRow returns the first or last selectable request row.
func BoundaryActiveRow(rowCount int, last bool) int {
	if rowCount <= 0 || !last {
		return 0
	}

	return rowCount - 1
}

// EnsureVisibleOffset adjusts the scroll offset so the active request row remains visible.
func EnsureVisibleOffset(activeRow, scrollOffset, visible int) int {
	if visible <= 0 {
		visible = 1
	}
	if activeRow < scrollOffset {
		return activeRow
	}
	if activeRow >= scrollOffset+visible {
		return activeRow - visible + 1
	}

	return scrollOffset
}

// ResetRowState returns the default row cursor and scroll state for the request pane.
func ResetRowState() RowState {
	return RowState{}
}

// SyncRowState clamps the request pane row state and keeps the active row visible when needed.
func SyncRowState(rows []RowDescriptor, state RowState, editKind model.RequestEditKind, visibleLines int) RowState {
	if len(rows) == 0 {
		return ResetRowState()
	}

	state.ActiveRow = ClampActiveRow(state.ActiveRow, len(rows))
	if editKind != model.RequestEditKindBody {
		state.ScrollOffset = EnsureVisibleOffset(state.ActiveRow, state.ScrollOffset, visibleLines)
	}

	return state
}

// MoveRowState updates the request pane row state after a relative row movement.
func MoveRowState(rows []RowDescriptor, state RowState, direction int, editKind model.RequestEditKind, visibleLines int) RowState {
	if len(rows) == 0 {
		return ResetRowState()
	}

	state.ActiveRow = MoveActiveRow(state.ActiveRow, len(rows), direction)
	return SyncRowState(rows, state, editKind, visibleLines)
}

// BoundaryRowState updates the request pane row state after jumping to a row boundary.
func BoundaryRowState(rows []RowDescriptor, state RowState, last bool, editKind model.RequestEditKind, visibleLines int) RowState {
	if len(rows) == 0 {
		return ResetRowState()
	}

	state.ActiveRow = BoundaryActiveRow(len(rows), last)
	return SyncRowState(rows, state, editKind, visibleLines)
}

// RowIndexByID returns the matching request row index for the given row identifier.
func RowIndexByID(rows []RowDescriptor, id string) int {
	for index, row := range rows {
		if row.ID == id {
			return index
		}
	}

	return -1
}
