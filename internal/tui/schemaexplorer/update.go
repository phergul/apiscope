package schemaexplorer

import (
	"github.com/phergul/apiscope/internal/model"
	"github.com/phergul/apiscope/internal/util"
)

// Available reports whether the selected operation exposes any reachable schema entrypoints.
func Available(operation *model.Operation) bool {
	return len(rootNodes(operation)) > 0
}

// OpenState returns the initial schema explorer state for one selected operation.
func OpenState(operation *model.Operation) State {
	if operation == nil {
		return State{}
	}

	state := State{
		OperationKey:    operation.Key,
		ExpandedNodeIDs: initialExpandedNodeIDs(operation),
	}
	rows := visibleRows(operation, state)
	if len(rows) > 1 {
		state.ActiveRow = 1
	}

	return state
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
	case "enter":
		result.State = toggleSelectedNode(operation, state, input.VisibleRows)
	case "l", "right":
		result.State = expandSelectedNode(operation, state, input.VisibleRows)
	case "h", "left", "backspace":
		result.State = collapseSelectedNode(operation, state, input.VisibleRows)
	case "esc", "q":
		result.Action.Close = true
	}

	return result
}

func syncState(operation *model.Operation, state State, viewportRows int) State {
	if state.ExpandedNodeIDs == nil {
		state.ExpandedNodeIDs = initialExpandedNodeIDs(operation)
	}

	rows := visibleRows(operation, state)
	if len(rows) == 0 {
		state.ActiveRow = 0
		state.TreeScrollOffset = 0
		return state
	}

	state.ActiveRow = util.Clamp(state.ActiveRow, 0, len(rows)-1)
	if viewportRows <= 0 {
		state.TreeScrollOffset = util.Clamp(state.TreeScrollOffset, 0, len(rows)-1)
		return state
	}

	maxOffset := max(len(rows)-viewportRows, 0)
	state.TreeScrollOffset = min(state.ActiveRow, util.Clamp(state.TreeScrollOffset, 0, maxOffset))
	if state.ActiveRow >= state.TreeScrollOffset+viewportRows {
		state.TreeScrollOffset = state.ActiveRow - viewportRows + 1
	}

	return state
}

func moveActiveRow(operation *model.Operation, state State, delta, viewportRows int) State {
	rows := visibleRows(operation, state)
	if len(rows) == 0 {
		return state
	}

	next := util.Clamp(state.ActiveRow+delta, 0, len(rows)-1)
	if next != state.ActiveRow {
		state.ActiveRow = next
		state.PreviewScrollOffset = 0
	}

	return syncState(operation, state, viewportRows)
}

func boundaryActiveRow(operation *model.Operation, state State, last bool, viewportRows int) State {
	rows := visibleRows(operation, state)
	if len(rows) == 0 {
		return state
	}

	if last {
		state.ActiveRow = len(rows) - 1
	} else {
		state.ActiveRow = 0
	}
	state.PreviewScrollOffset = 0

	return syncState(operation, state, viewportRows)
}

func toggleSelectedNode(operation *model.Operation, state State, viewportRows int) State {
	rows := visibleRows(operation, state)
	selected, ok := activeVisibleRow(rows, state.ActiveRow)
	if !ok || !expandable(selected.Node) {
		return state
	}

	if isExpanded(state.ExpandedNodeIDs, selected.Node.ID) {
		return collapseNodeByID(operation, state, selected.Node.ID, selected.Node.ID, viewportRows, false)
	}

	state.ExpandedNodeIDs = cloneExpandedNodeIDs(state.ExpandedNodeIDs)
	state.ExpandedNodeIDs[selected.Node.ID] = struct{}{}
	return syncState(operation, state, viewportRows)
}

func expandSelectedNode(operation *model.Operation, state State, viewportRows int) State {
	rows := visibleRows(operation, state)
	selected, ok := activeVisibleRow(rows, state.ActiveRow)
	if !ok || !expandable(selected.Node) || isExpanded(state.ExpandedNodeIDs, selected.Node.ID) {
		return state
	}

	state.ExpandedNodeIDs = cloneExpandedNodeIDs(state.ExpandedNodeIDs)
	state.ExpandedNodeIDs[selected.Node.ID] = struct{}{}
	return syncState(operation, state, viewportRows)
}

func collapseSelectedNode(operation *model.Operation, state State, viewportRows int) State {
	rows := visibleRows(operation, state)
	selected, ok := activeVisibleRow(rows, state.ActiveRow)
	if !ok || selected.Node == nil {
		return state
	}

	if expandable(selected.Node) && isExpanded(state.ExpandedNodeIDs, selected.Node.ID) {
		return collapseNodeByID(operation, state, selected.Node.ID, selected.Node.ID, viewportRows, false)
	}

	for ancestor := selected.Node.Parent; ancestor != nil; ancestor = ancestor.Parent {
		if expandable(ancestor) && isExpanded(state.ExpandedNodeIDs, ancestor.ID) {
			// When collapsing an ancestor from a child row, move selection to the ancestor so
			// preview and tree focus stay aligned with the now-hidden branch.
			return collapseNodeByID(operation, state, ancestor.ID, ancestor.ID, viewportRows, true)
		}
	}

	return state
}

func collapseNodeByID(operation *model.Operation, state State, collapseID, selectionID string, viewportRows int, resetPreview bool) State {
	state.ExpandedNodeIDs = cloneExpandedNodeIDs(state.ExpandedNodeIDs)
	delete(state.ExpandedNodeIDs, collapseID)
	rows := visibleRows(operation, state)
	if index := rowIndexByNodeID(rows, selectionID); index >= 0 {
		state.ActiveRow = index
	}
	if resetPreview {
		state.PreviewScrollOffset = 0
	}

	return syncState(operation, state, viewportRows)
}

func cloneExpandedNodeIDs(expanded map[string]struct{}) map[string]struct{} {
	if len(expanded) == 0 {
		return map[string]struct{}{}
	}

	cloned := make(map[string]struct{}, len(expanded))
	for id := range expanded {
		cloned[id] = struct{}{}
	}

	return cloned
}
