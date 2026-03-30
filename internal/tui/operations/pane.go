package operations

import "github.com/phergul/apiscope/internal/model"

// PaneInput contains the root-owned state needed to project the operations pane.
type PaneInput struct {
	LoadInFlight bool
	LoadFailed   bool
	HasSpec      bool
	Operations   []model.Operation
	VisibleKeys  []model.OperationKey
	SelectedKey  model.OperationKey
	ContentWidth int
	ScrollOffset int
	MaxLines     int
}

// PaneProjection contains the projected operations pane plus scroll metadata.
type PaneProjection struct {
	Data            Data
	VisibleRows     int
	MaxScrollOffset int
}

// ProjectPane projects root operations state into a render-ready operations pane model.
func ProjectPane(input PaneInput) PaneProjection {
	data := projectPaneData(input)
	if len(input.VisibleKeys) == 0 {
		return PaneProjection{Data: data}
	}

	visibleRows := VisibleRowCount(data)
	return PaneProjection{
		Data:            data,
		VisibleRows:     visibleRows,
		MaxScrollOffset: MaxScrollOffset(input),
	}
}

// projectPaneData builds the unwindowed operations pane data.
func projectPaneData(input PaneInput) Data {
	data := Data{
		LoadInFlight:    input.LoadInFlight,
		LoadFailed:      input.LoadFailed,
		HasSpec:         input.HasSpec,
		ContentWidth:    input.ContentWidth,
		ScrollOffset:    input.ScrollOffset,
		MaxLines:        input.MaxLines,
		TotalOperations: len(input.Operations),
	}
	if !input.HasSpec || input.LoadInFlight || input.LoadFailed || len(input.Operations) == 0 {
		return data
	}

	data.Groups = projectGroups(input.Operations, input.VisibleKeys, input.SelectedKey)
	return data
}

// projectGroups projects visible operation keys into grouped render rows.
func projectGroups(operations []model.Operation, visibleKeys []model.OperationKey, selectedKey model.OperationKey) []Group {
	lookup := operationMap(operations)
	keyGroups := GroupKeys(visibleKeys, operations)
	groups := make([]Group, 0, len(keyGroups))
	for _, keyGroup := range keyGroups {
		group := Group{Name: keyGroup.Name}
		for _, key := range keyGroup.Keys {
			operation, ok := lookup[key]
			if !ok {
				continue
			}

			group.Rows = append(group.Rows, Row{
				Method:   operation.Method,
				Path:     operation.Path,
				Selected: operation.Key == selectedKey,
			})
		}
		groups = append(groups, group)
	}

	return groups
}
