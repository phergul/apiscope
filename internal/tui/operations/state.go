package operations

import (
	"slices"

	"github.com/phergul/apiscope/internal/model"
	"github.com/phergul/apiscope/internal/util"
)

// ListState contains the cursor and scroll state for the operations pane.
type ListState struct {
	Cursor       int
	ScrollOffset int
}

// StateInput contains the data needed to update operations list state.
type StateInput struct {
	Operations   []model.Operation
	VisibleKeys  []model.OperationKey
	ContentWidth int
	MaxLines     int
}

// ResetListState returns the zeroed operations list state.
func ResetListState() ListState {
	return ListState{}
}

// SyncListState clamps operations cursor and scroll state to the visible list.
func SyncListState(input StateInput, state ListState) ListState {
	if len(input.VisibleKeys) == 0 {
		return ResetListState()
	}

	state.Cursor = util.Clamp(state.Cursor, 0, len(input.VisibleKeys)-1)
	maxOffset := min(len(input.VisibleKeys)-1, MaxScrollOffset(PaneInput{
		HasSpec:      true,
		Operations:   input.Operations,
		VisibleKeys:  input.VisibleKeys,
		ContentWidth: input.ContentWidth,
		MaxLines:     input.MaxLines,
	}))
	state.ScrollOffset = util.Clamp(state.ScrollOffset, 0, max(maxOffset, 0))

	// keep roughly five rows of context above and below the cursor.
	scrolloff := 5
	for range 3 {
		visibleRows := visibleRowCountForOffset(input, state.ScrollOffset)
		if visibleRows <= 0 {
			state.ScrollOffset = 0
			return state
		}

		maxScrolloff := max(visibleRows-1, 0)
		if scrolloff > maxScrolloff {
			scrolloff = maxScrolloff
		}

		minCursor := state.ScrollOffset + scrolloff
		maxCursor := state.ScrollOffset + visibleRows - scrolloff - 1
		if maxCursor < minCursor {
			maxCursor = minCursor
		}

		nextOffset := state.ScrollOffset
		switch {
		case state.Cursor < minCursor:
			nextOffset = state.Cursor - scrolloff
		case state.Cursor > maxCursor:
			nextOffset = state.Cursor - visibleRows + scrolloff + 1
		default:
			return state
		}

		nextOffset = util.Clamp(nextOffset, 0, max(maxOffset, 0))
		if nextOffset == state.ScrollOffset {
			return state
		}
		state.ScrollOffset = nextOffset
	}

	return state
}

// MoveListState moves the operations cursor by the requested direction.
func MoveListState(input StateInput, state ListState, direction int) ListState {
	state.Cursor += direction
	return SyncListState(input, state)
}

// BoundaryListState moves the operations cursor to the first or last visible row.
func BoundaryListState(input StateInput, state ListState, last bool) ListState {
	if len(input.VisibleKeys) == 0 {
		return ResetListState()
	}
	if last {
		state.Cursor = len(input.VisibleKeys) - 1
	} else {
		state.Cursor = 0
	}

	return SyncListState(input, state)
}

// MaxScrollOffset returns the largest valid operations scroll offset for the current pane size.
func MaxScrollOffset(input PaneInput) int {
	totalRows := len(input.VisibleKeys)
	if totalRows <= 1 {
		return 0
	}

	for offset := 0; offset < totalRows; offset++ {
		if visibleRowCountForOffset(StateInput{
			Operations:   input.Operations,
			VisibleKeys:  input.VisibleKeys,
			ContentWidth: input.ContentWidth,
			MaxLines:     input.MaxLines,
		}, offset) == totalRows-offset {
			return offset
		}
	}

	return totalRows - 1
}

// AdjacentGroupTarget returns the first key in the adjacent rendered group, if any.
func AdjacentGroupTarget(operations []model.Operation, visibleKeys []model.OperationKey, currentKey model.OperationKey, direction int) model.OperationKey {
	groups := GroupKeys(visibleKeys, operations)
	if len(groups) == 0 {
		return ""
	}

	currentGroupIndex := -1
	for index, group := range groups {
		if slices.Contains(group.Keys, currentKey) {
			currentGroupIndex = index
			break
		}
	}
	if currentGroupIndex < 0 {
		currentGroupIndex = 0
	}

	targetIndex := currentGroupIndex + direction
	if targetIndex < 0 || targetIndex >= len(groups) || len(groups[targetIndex].Keys) == 0 {
		return ""
	}

	return groups[targetIndex].Keys[0]
}

// visibleRowCountForOffset returns the number of visible rows for a given operations scroll offset.
func visibleRowCountForOffset(input StateInput, offset int) int {
	return VisibleRowCount(projectPaneData(PaneInput{
		HasSpec:      true,
		Operations:   input.Operations,
		VisibleKeys:  input.VisibleKeys,
		ContentWidth: input.ContentWidth,
		ScrollOffset: offset,
		MaxLines:     input.MaxLines,
	}))
}
