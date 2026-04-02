package schemaexplorer

import (
	"fmt"
	"sort"
	"strings"

	"github.com/phergul/apiscope/internal/model"
	"github.com/phergul/apiscope/internal/tui/describe"
	"github.com/phergul/apiscope/internal/tui/widgets"
	"github.com/phergul/apiscope/internal/util"
)

// Project converts operation schemas plus runtime explorer state into render-ready explorer data.
func Project(input ProjectionInput) Projection {
	data := Data{
		LeftTitle:  "Schemas",
		RightTitle: "Preview",
	}
	if input.Operation == nil {
		data.EmptyState = "No operation selected."
		return Projection{Data: data}
	}
	if !Available(input.Operation) {
		data.EmptyState = "No schemas available for this operation."
		return Projection{Data: data}
	}

	leftWidth, rightWidth := columnWidths(input.ContentWidth)
	listHeight := max(input.ContentHeight-2, 1)
	previewHeight := max(input.ContentHeight-2, 1)

	state := syncState(input.Operation, input.State, listHeight)
	rows := currentRows(input.Operation, state)
	selected, hasSelected := activeRow(rows, state.ActiveRow)
	currentNode, currentLabel, currentNote := previewSelection(state, selected, hasSelected)
	if currentNode == nil && len(state.Breadcrumbs) > 0 {
		last := state.Breadcrumbs[len(state.Breadcrumbs)-1]
		currentNode = last.Schema
		currentLabel = last.Label
	}
	if strings.TrimSpace(currentLabel) != "" {
		data.RightTitle = currentLabel
	}

	data.LeftWidth = leftWidth
	data.RightWidth = rightWidth
	data.LeftBody = renderRows(rows, state.ActiveRow, state.RowScrollOffset, listHeight, leftWidth)

	previewBody := previewBody(state, currentNode, currentLabel, currentNote)
	maxPreviewScroll := max(scrollLines(previewBody)-previewHeight, 0)
	previewViewport := widgets.NewViewport(max(rightWidth, 1), previewHeight)
	previewViewport.SetContent(previewBody)
	previewViewport.SetYOffset(util.Clamp(input.State.PreviewScrollOffset, 0, maxPreviewScroll))
	data.RightBody = previewViewport.View()

	return Projection{
		Data:             data,
		MaxPreviewScroll: maxPreviewScroll,
		MaxRowScroll:     max(len(rows)-listHeight, 0),
		VisibleRows:      listHeight,
	}
}

func currentRows(operation *model.Operation, state State) []row {
	if len(state.Breadcrumbs) == 0 {
		return rootRows(operation, state.Breadcrumbs)
	}

	current := state.Breadcrumbs[len(state.Breadcrumbs)-1].Schema
	return childRows(current, state.Breadcrumbs)
}

func rootRows(operation *model.Operation, stack []Breadcrumb) []row {
	if operation == nil {
		return nil
	}

	rows := make([]row, 0)
	for _, parameter := range operation.Parameters {
		if parameter.Schema != nil {
			rows = append(rows, makeRow(parameterEntryLabel(parameter), parameter.Schema, stack))
		}
		for _, content := range parameter.Content {
			if content.Schema == nil {
				continue
			}
			rows = append(rows, makeRow(parameterContentEntryLabel(parameter, content.MediaType), content.Schema, stack))
		}
	}

	if operation.RequestBody != nil {
		for _, content := range operation.RequestBody.Content {
			if content.Schema == nil {
				continue
			}
			rows = append(rows, makeRow("Request body: "+content.MediaType, content.Schema, stack))
		}
	}

	for _, response := range operation.Responses {
		for _, content := range response.Content {
			if content.Schema == nil {
				continue
			}
			rows = append(rows, makeRow(fmt.Sprintf("Response %s: %s", response.StatusCode, content.MediaType), content.Schema, stack))
		}
		for _, header := range response.Headers {
			if header.Schema != nil {
				rows = append(rows, makeRow(fmt.Sprintf("Response %s header: %s", response.StatusCode, header.Name), header.Schema, stack))
			}
			for _, content := range header.Content {
				if content.Schema == nil {
					continue
				}
				rows = append(rows, makeRow(fmt.Sprintf("Response %s header content: %s (%s)", response.StatusCode, header.Name, content.MediaType), content.Schema, stack))
			}
		}
	}

	return rows
}

func childRows(schema *model.Schema, stack []Breadcrumb) []row {
	if schema == nil {
		return nil
	}

	rows := make([]row, 0)
	if len(schema.Properties) > 0 {
		names := make([]string, 0, len(schema.Properties))
		for name := range schema.Properties {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			rows = append(rows, makeRow(name, schema.Properties[name], stack))
		}
	}
	if schema.Items != nil {
		rows = append(rows, makeRow("items", schema.Items, stack))
	}
	rows = append(rows, compositionRows("oneOf", schema.OneOf, stack)...)
	rows = append(rows, compositionRows("anyOf", schema.AnyOf, stack)...)
	rows = append(rows, compositionRows("allOf", schema.AllOf, stack)...)

	return rows
}

func compositionRows(prefix string, schemas []*model.Schema, stack []Breadcrumb) []row {
	rows := make([]row, 0, len(schemas))
	for index, schema := range schemas {
		rows = append(rows, makeRow(fmt.Sprintf("%s[%d]", prefix, index), schema, stack))
	}
	return rows
}

func makeRow(label string, schema *model.Schema, stack []Breadcrumb) row {
	result := row{
		Label:     strings.TrimSpace(label),
		Schema:    schema,
		Meta:      describe.SchemaTypeHint(schema),
		Drillable: schema != nil,
	}

	switch {
	case schema == nil:
		result.Drillable = false
		result.Note = "missing schema"
	case len(stack) >= maxTraversalDepth-1:
		result.Drillable = false
		result.Note = "depth limit reached"
	case recursiveSchema(schema, stack):
		// Ref-based recursion is shown explicitly in the list so users can see the relationship
		// without walking into an infinite drill chain.
		result.Drillable = false
		result.Note = "recursive reference"
	}

	return result
}

func recursiveSchema(schema *model.Schema, stack []Breadcrumb) bool {
	if schema == nil || strings.TrimSpace(schema.Ref) == "" {
		return false
	}

	for _, breadcrumb := range stack {
		if breadcrumb.Schema == nil {
			continue
		}
		if strings.TrimSpace(breadcrumb.Schema.Ref) == strings.TrimSpace(schema.Ref) {
			return true
		}
	}

	return false
}

func activeRow(rows []row, activeRow int) (row, bool) {
	if len(rows) == 0 {
		return row{}, false
	}

	index := util.Clamp(activeRow, 0, len(rows)-1)
	return rows[index], true
}

func previewSelection(state State, selected row, ok bool) (*model.Schema, string, string) {
	if ok {
		return selected.Schema, selected.Label, selected.Note
	}
	if len(state.Breadcrumbs) == 0 {
		return nil, "", ""
	}

	last := state.Breadcrumbs[len(state.Breadcrumbs)-1]
	return last.Schema, last.Label, ""
}

func parameterEntryLabel(parameter model.Parameter) string {
	return humanLocation(parameter.In) + " param: " + parameter.Name
}

func parameterContentEntryLabel(parameter model.Parameter, mediaType string) string {
	return humanLocation(parameter.In) + " content: " + parameter.Name + " (" + mediaType + ")"
}

func humanLocation(location model.ParameterLocation) string {
	switch location {
	case model.ParameterLocationPath:
		return "Path"
	case model.ParameterLocationQuery:
		return "Query"
	case model.ParameterLocationHeader:
		return "Header"
	case model.ParameterLocationCookie:
		return "Cookie"
	case model.ParameterLocationForm:
		return "Form"
	default:
		return "Unknown"
	}
}
