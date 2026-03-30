package operations

import (
	"strconv"
	"testing"

	"github.com/phergul/apiscope/internal/model"
)

func TestFilterVisibleKeysPreservesGroupedRenderedOrder(t *testing.T) {
	t.Parallel()

	operations := []model.Operation{
		{Key: model.NewOperationKey("GET", "/albums"), Method: "GET", Path: "/albums", Tags: []string{"albums"}},
		{Key: model.NewOperationKey("GET", "/artists"), Method: "GET", Path: "/artists", Tags: []string{"artists"}},
		{Key: model.NewOperationKey("GET", "/me/albums"), Method: "GET", Path: "/me/albums", Tags: []string{"albums"}},
	}

	got := FilterVisibleKeys(operations, "")
	if len(got) != 3 ||
		got[0] != model.NewOperationKey("GET", "/albums") ||
		got[1] != model.NewOperationKey("GET", "/me/albums") ||
		got[2] != model.NewOperationKey("GET", "/artists") {
		t.Fatalf("expected grouped visible order, got %#v", got)
	}
}

func TestMoveListStateKeepsFiveRowsBelowCursorWhenMovingDown(t *testing.T) {
	t.Parallel()

	operations := make([]model.Operation, 0, 20)
	visibleKeys := make([]model.OperationKey, 0, 20)
	for index := 0; index < 20; index++ {
		path := "/pets/" + strconv.Itoa(index)
		key := model.NewOperationKey("GET", path)
		operations = append(operations, model.Operation{
			Key:    key,
			Method: "GET",
			Path:   path,
			Tags:   []string{"pets"},
		})
		visibleKeys = append(visibleKeys, key)
	}

	state := ResetListState()
	input := StateInput{
		Operations:   operations,
		VisibleKeys:  visibleKeys,
		ContentWidth: 76,
		MaxLines:     12,
	}
	for range 10 {
		state = MoveListState(input, state, 1)
	}

	if state.Cursor != 10 {
		t.Fatalf("expected cursor to move to row 10, got %d", state.Cursor)
	}
	if state.ScrollOffset != 5 {
		t.Fatalf("expected scroll offset 5 to preserve five-row scrolloff, got %d", state.ScrollOffset)
	}
}

func TestAdjacentGroupTargetMovesToNextAndPreviousGroups(t *testing.T) {
	t.Parallel()

	operations := []model.Operation{
		{Key: model.NewOperationKey("GET", "/pets"), Method: "GET", Path: "/pets", Tags: []string{"pets"}},
		{Key: model.NewOperationKey("POST", "/pets"), Method: "POST", Path: "/pets", Tags: []string{"pets"}},
		{Key: model.NewOperationKey("GET", "/admin"), Method: "GET", Path: "/admin", Tags: []string{"admin"}},
	}
	visibleKeys := FilterVisibleKeys(operations, "")

	next := AdjacentGroupTarget(operations, visibleKeys, model.NewOperationKey("GET", "/pets"), 1)
	if next != model.NewOperationKey("GET", "/admin") {
		t.Fatalf("expected next group target /admin, got %q", next)
	}

	previous := AdjacentGroupTarget(operations, visibleKeys, model.NewOperationKey("GET", "/admin"), -1)
	if previous != model.NewOperationKey("GET", "/pets") {
		t.Fatalf("expected previous group target /pets, got %q", previous)
	}
}

func TestMaxScrollOffsetAlignsWithRenderedListBottom(t *testing.T) {
	t.Parallel()

	operations := make([]model.Operation, 0, 8)
	visibleKeys := make([]model.OperationKey, 0, 8)
	for index := 0; index < 8; index++ {
		path := "/pets/" + strconv.Itoa(index)
		key := model.NewOperationKey("GET", path)
		operations = append(operations, model.Operation{
			Key:    key,
			Method: "GET",
			Path:   path,
			Tags:   []string{"pets"},
		})
		visibleKeys = append(visibleKeys, key)
	}

	input := PaneInput{
		HasSpec:      true,
		Operations:   operations,
		VisibleKeys:  visibleKeys,
		ContentWidth: 28,
		MaxLines:     6,
	}
	maxOffset := MaxScrollOffset(input)
	projected := ProjectPane(PaneInput{
		HasSpec:      true,
		Operations:   operations,
		VisibleKeys:  visibleKeys,
		ContentWidth: 28,
		ScrollOffset: maxOffset,
		MaxLines:     6,
	})
	if projected.VisibleRows != len(visibleKeys)-maxOffset {
		t.Fatalf("expected bottom-aligned visible rows, got %d for max offset %d", projected.VisibleRows, maxOffset)
	}
}
