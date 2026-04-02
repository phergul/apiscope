package schemaexplorer

import "github.com/phergul/apiscope/internal/model"

const maxTraversalDepth = 12

// Breadcrumb describes one drilled schema node in the current explorer path.
type Breadcrumb struct {
	Label  string
	Schema *model.Schema
}

// State holds the root-owned runtime state for the schema explorer.
type State struct {
	OperationKey        model.OperationKey
	Breadcrumbs         []Breadcrumb
	ActiveRow           int
	RowScrollOffset     int
	PreviewScrollOffset int
}

// Action describes one shell-level effect requested by the explorer update logic.
type Action struct {
	Close bool
}

// UpdateInput carries the current shell key plus viewport bounds into explorer updates.
type UpdateInput struct {
	Key              string
	VisibleRows      int
	MaxPreviewScroll int
}

// UpdateResult returns the next explorer state plus any shell-level action.
type UpdateResult struct {
	State  State
	Action Action
}

// ProjectionInput contains the operation, runtime state, and available body size.
type ProjectionInput struct {
	Operation     *model.Operation
	State         State
	ContentWidth  int
	ContentHeight int
}

// Projection contains render-ready explorer data plus scroll metadata for root routing.
type Projection struct {
	Data             Data
	MaxPreviewScroll int
	MaxRowScroll     int
	VisibleRows      int
}

// Data contains the render-ready full-window explorer content.
type Data struct {
	EmptyState string
	LeftTitle  string
	RightTitle string
	LeftBody   string
	RightBody  string
	LeftWidth  int
	RightWidth int
}

type row struct {
	Label     string
	Meta      string
	Schema    *model.Schema
	Drillable bool
	Note      string
}
