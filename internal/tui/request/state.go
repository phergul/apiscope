package request

import "github.com/phergul/apiscope/internal/model"

type RowState struct {
	ActiveRow    int
	ScrollOffset int
}

// RowSelectable reports whether the request row can receive the active cursor.
func RowSelectable(row RowDescriptor) bool {
	return row.Editable
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

// ClampSelectableRow clamps the active row into the nearest selectable row.
func ClampSelectableRow(rows []RowDescriptor, activeRow int) int {
	if len(rows) == 0 {
		return 0
	}

	if index, ok := selectableRowIndex(rows, ClampActiveRow(activeRow, len(rows)), 1); ok {
		return index
	}
	if index, ok := selectableRowIndex(rows, ClampActiveRow(activeRow, len(rows)), -1); ok {
		return index
	}

	return 0
}

// MoveActiveRow moves the active request row by the given delta.
func MoveActiveRow(activeRow, rowCount, direction int) int {
	return ClampActiveRow(activeRow+direction, rowCount)
}

// MoveSelectableRow moves to the next selectable row in the given direction.
func MoveSelectableRow(rows []RowDescriptor, activeRow, direction int) int {
	if len(rows) == 0 {
		return 0
	}
	if direction == 0 {
		return ClampSelectableRow(rows, activeRow)
	}

	start := ClampSelectableRow(rows, activeRow) + direction
	if index, ok := selectableRowIndex(rows, start, direction); ok {
		return index
	}

	return ClampSelectableRow(rows, activeRow)
}

// BoundaryActiveRow returns the first or last selectable request row.
func BoundaryActiveRow(rowCount int, last bool) int {
	if rowCount <= 0 || !last {
		return 0
	}

	return rowCount - 1
}

// BoundarySelectableRow returns the first or last selectable request row.
func BoundarySelectableRow(rows []RowDescriptor, last bool) int {
	if len(rows) == 0 {
		return 0
	}

	direction := 1
	start := 0
	if last {
		direction = -1
		start = len(rows) - 1
	}

	if index, ok := selectableRowIndex(rows, start, direction); ok {
		return index
	}

	return 0
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

	state.ActiveRow = ClampSelectableRow(rows, state.ActiveRow)
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

	state.ActiveRow = MoveSelectableRow(rows, state.ActiveRow, direction)
	return SyncRowState(rows, state, editKind, visibleLines)
}

// BoundaryRowState updates the request pane row state after jumping to a row boundary.
func BoundaryRowState(rows []RowDescriptor, state RowState, last bool, editKind model.RequestEditKind, visibleLines int) RowState {
	if len(rows) == 0 {
		return ResetRowState()
	}

	state.ActiveRow = BoundarySelectableRow(rows, last)
	return SyncRowState(rows, state, editKind, visibleLines)
}

// RowIndexByID returns the matching request row index for the given row or validation target.
func RowIndexByID(rows []RowDescriptor, id string) int {
	for index, row := range rows {
		if row.ValidationTarget == id || row.ID == id {
			return index
		}
	}

	return -1
}

func selectableRowIndex(rows []RowDescriptor, start, direction int) (int, bool) {
	if len(rows) == 0 {
		return 0, false
	}

	start = ClampActiveRow(start, len(rows))
	if direction >= 0 {
		for index := start; index < len(rows); index++ {
			if RowSelectable(rows[index]) {
				return index, true
			}
		}
		return 0, false
	}

	for index := start; index >= 0; index-- {
		if RowSelectable(rows[index]) {
			return index, true
		}
	}

	return 0, false
}
