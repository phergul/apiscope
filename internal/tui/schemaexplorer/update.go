package schemaexplorer

import (
	"github.com/phergul/apiscope/internal/model"
	"github.com/phergul/apiscope/internal/util"
)

// Available reports whether the selected operation exposes any reachable schema entrypoints.
func Available(operation *model.Operation) bool {
	return len(rootRows(operation, nil)) > 0
}

// OpenState returns the initial schema explorer state for one selected operation.
func OpenState(operation *model.Operation) State {
	if operation == nil {
		return State{}
	}

	return State{
		OperationKey: operation.Key,
	}
}

// Update applies one explorer key event and returns the next state plus shell-level actions.
func Update(operation *model.Operation, state State, input UpdateInput) UpdateResult {
	state = syncState(operation, state, input.VisibleRows)
	result := UpdateResult{State: state}

	switch input.Key {
	case "j", "down":
		result.State = moveActiveRow(operation, state, 1, input.VisibleRows)
	case "k", "up":
		result.State = moveActiveRow(operation, state, -1, input.VisibleRows)
	case "home":
		result.State = boundaryActiveRow(operation, state, false, input.VisibleRows)
	case "end":
		result.State = boundaryActiveRow(operation, state, true, input.VisibleRows)
	case "ctrl+u":
		result.State.PreviewScrollOffset = max(result.State.PreviewScrollOffset-5, 0)
	case "ctrl+d":
		result.State.PreviewScrollOffset = min(result.State.PreviewScrollOffset+5, input.MaxPreviewScroll)
	case "enter", "l":
		rows := currentRows(operation, state)
		if selected, ok := activeRow(rows, state.ActiveRow); ok && selected.Drillable && selected.Schema != nil {
			result.State.Breadcrumbs = append(append([]Breadcrumb(nil), state.Breadcrumbs...), Breadcrumb{
				Label:  selected.Label,
				Schema: selected.Schema,
			})
			result.State.ActiveRow = 0
			result.State.RowScrollOffset = 0
			result.State.PreviewScrollOffset = 0
			result.State = syncState(operation, result.State, input.VisibleRows)
		}
	case "h", "left", "backspace":
		if len(state.Breadcrumbs) > 0 {
			result.State.Breadcrumbs = append([]Breadcrumb(nil), state.Breadcrumbs[:len(state.Breadcrumbs)-1]...)
			result.State.ActiveRow = 0
			result.State.RowScrollOffset = 0
			result.State.PreviewScrollOffset = 0
			result.State = syncState(operation, result.State, input.VisibleRows)
		}
	case "esc", "q":
		result.Action.Close = true
	}

	return result
}

func syncState(operation *model.Operation, state State, visibleRows int) State {
	rows := currentRows(operation, state)
	if len(rows) == 0 {
		state.ActiveRow = 0
		state.RowScrollOffset = 0
		return state
	}

	state.ActiveRow = util.Clamp(state.ActiveRow, 0, len(rows)-1)
	if visibleRows <= 0 {
		state.RowScrollOffset = util.Clamp(state.RowScrollOffset, 0, len(rows)-1)
		return state
	}

	maxOffset := max(len(rows)-visibleRows, 0)
	state.RowScrollOffset = min(state.ActiveRow, util.Clamp(state.RowScrollOffset, 0, maxOffset))
	if state.ActiveRow >= state.RowScrollOffset+visibleRows {
		state.RowScrollOffset = state.ActiveRow - visibleRows + 1
	}

	return state
}

func moveActiveRow(operation *model.Operation, state State, delta, visibleRows int) State {
	rows := currentRows(operation, state)
	if len(rows) == 0 {
		return state
	}

	next := util.Clamp(state.ActiveRow+delta, 0, len(rows)-1)
	if next != state.ActiveRow {
		state.ActiveRow = next
		state.PreviewScrollOffset = 0
	}

	return syncState(operation, state, visibleRows)
}

func boundaryActiveRow(operation *model.Operation, state State, last bool, visibleRows int) State {
	rows := currentRows(operation, state)
	if len(rows) == 0 {
		return state
	}

	if last {
		state.ActiveRow = len(rows) - 1
	} else {
		state.ActiveRow = 0
	}
	state.PreviewScrollOffset = 0

	return syncState(operation, state, visibleRows)
}
