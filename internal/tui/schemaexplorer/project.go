package schemaexplorer

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/phergul/apiscope/internal/model"
	"github.com/phergul/apiscope/internal/tui/describe"
	"github.com/phergul/apiscope/internal/tui/widgets"
	"github.com/phergul/apiscope/internal/util"
)

const (
	groupIDPathParams            = "group:path-params"
	groupIDQueryParams           = "group:query-params"
	groupIDHeaderParams          = "group:header-params"
	groupIDCookieParams          = "group:cookie-params"
	groupIDFormParams            = "group:form-params"
	groupIDParameterContent      = "group:parameter-content"
	groupIDRequestBodies         = "group:request-bodies"
	groupIDResponses             = "group:responses"
	groupIDResponseHeaders       = "group:response-headers"
	groupIDResponseHeaderContent = "group:response-header-content"
)

type groupSpec struct {
	ID    string
	Label string
}

var groupOrder = []groupSpec{
	{ID: groupIDPathParams, Label: "Path params"},
	{ID: groupIDQueryParams, Label: "Query params"},
	{ID: groupIDHeaderParams, Label: "Header params"},
	{ID: groupIDCookieParams, Label: "Cookie params"},
	{ID: groupIDFormParams, Label: "Form params"},
	{ID: groupIDParameterContent, Label: "Parameter content"},
	{ID: groupIDRequestBodies, Label: "Request bodies"},
	{ID: groupIDResponses, Label: "Responses"},
	{ID: groupIDResponseHeaders, Label: "Response headers"},
	{ID: groupIDResponseHeaderContent, Label: "Response header content"},
}

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
	rows := visibleRows(input.Operation, state)
	selected, hasSelected := activeVisibleRow(rows, state.ActiveRow)
	if hasSelected {
		data.RightTitle = previewTitle(selected.Node)
	}

	data.LeftWidth = leftWidth
	data.RightWidth = rightWidth
	data.LeftBody = renderRows(rows, state.ActiveRow, state.TreeScrollOffset, listHeight, leftWidth)

	previewBody := previewBody(selectedNode(selected, hasSelected), rightWidth)
	maxPreviewScroll := max(scrollLines(previewBody)-previewHeight, 0)
	previewViewport := widgets.NewViewport(max(rightWidth, 1), previewHeight)
	previewViewport.SetContent(previewBody)
	previewViewport.SetYOffset(util.Clamp(state.PreviewScrollOffset, 0, maxPreviewScroll))
	data.RightBody = previewViewport.View()

	return Projection{
		Data:             data,
		MaxPreviewScroll: maxPreviewScroll,
		VisibleRows:      listHeight,
	}
}

func selectedNode(selected visibleRow, ok bool) *treeNode {
	if !ok {
		return nil
	}

	return selected.Node
}

func rootNodes(operation *model.Operation) []*treeNode {
	if operation == nil {
		return nil
	}

	grouped := make(map[string][]*treeNode)

	for _, parameter := range operation.Parameters {
		if parameter.Schema != nil {
			id := fmt.Sprintf("parameter:%s:%s:schema", parameter.In, parameter.Name)
			grouped[parameterGroupID(parameter.In)] = append(grouped[parameterGroupID(parameter.In)],
				buildSchemaNode(id, nodeLabel{
					Prefix: singularParameterPrefix(parameter.In) + ":",
					Name:   parameter.Name,
					Meta:   describe.SchemaTypeHint(parameter.Schema),
				}, parameter.Schema, nil, 0, nil))
		}
		for _, content := range parameter.Content {
			if content.Schema == nil {
				continue
			}
			id := fmt.Sprintf("parameter:%s:%s:content:%s", parameter.In, parameter.Name, content.MediaType)
			grouped[groupIDParameterContent] = append(grouped[groupIDParameterContent],
				buildSchemaNode(id, nodeLabel{
					Prefix: singularParameterContentPrefix(parameter.In) + ":",
					Name:   parameter.Name + " (" + content.MediaType + ")",
					Meta:   describe.SchemaTypeHint(content.Schema),
				}, content.Schema, nil, 0, nil))
		}
	}

	if operation.RequestBody != nil {
		for _, content := range operation.RequestBody.Content {
			if content.Schema == nil {
				continue
			}
			id := fmt.Sprintf("request-body:%s", content.MediaType)
			grouped[groupIDRequestBodies] = append(grouped[groupIDRequestBodies],
				buildSchemaNode(id, nodeLabel{
					Prefix: "Request body:",
					Name:   content.MediaType,
					Meta:   describe.SchemaTypeHint(content.Schema),
				}, content.Schema, nil, 0, nil))
		}
	}

	for _, response := range operation.Responses {
		for _, content := range response.Content {
			if content.Schema == nil {
				continue
			}
			id := fmt.Sprintf("response:%s:content:%s", response.StatusCode, content.MediaType)
			grouped[groupIDResponses] = append(grouped[groupIDResponses],
				buildSchemaNode(id, nodeLabel{
					Prefix: "Response " + response.StatusCode + ":",
					Name:   content.MediaType,
					Meta:   describe.SchemaTypeHint(content.Schema),
				}, content.Schema, nil, 0, nil))
		}
		for _, header := range response.Headers {
			if header.Schema != nil {
				id := fmt.Sprintf("response:%s:header:%s:schema", response.StatusCode, header.Name)
				grouped[groupIDResponseHeaders] = append(grouped[groupIDResponseHeaders],
					buildSchemaNode(id, nodeLabel{
						Prefix: "Response " + response.StatusCode + " header:",
						Name:   header.Name,
						Meta:   describe.SchemaTypeHint(header.Schema),
					}, header.Schema, nil, 0, nil))
			}
			for _, content := range header.Content {
				if content.Schema == nil {
					continue
				}
				id := fmt.Sprintf("response:%s:header:%s:content:%s", response.StatusCode, header.Name, content.MediaType)
				grouped[groupIDResponseHeaderContent] = append(grouped[groupIDResponseHeaderContent],
					buildSchemaNode(id, nodeLabel{
						Prefix: "Response " + response.StatusCode + " header content:",
						Name:   header.Name + " (" + content.MediaType + ")",
						Meta:   describe.SchemaTypeHint(content.Schema),
					}, content.Schema, nil, 0, nil))
			}
		}
	}

	roots := make([]*treeNode, 0, len(groupOrder))
	for _, group := range groupOrder {
		children := grouped[group.ID]
		if len(children) == 0 {
			continue
		}

		root := &treeNode{
			ID:    group.ID,
			Label: nodeLabel{Name: group.Label},
		}
		for _, child := range children {
			child.Parent = root
		}
		root.Children = children
		roots = append(roots, root)
	}

	return roots
}

func buildSchemaNode(id string, label nodeLabel, schema *model.Schema, parent *treeNode, depth int, ancestorRefs []string) *treeNode {
	node := &treeNode{
		ID:     id,
		Parent: parent,
		Label:  label,
		Schema: schema,
	}
	if schema == nil {
		node.Note = "missing schema"
		return node
	}
	if depth >= maxTraversalDepth {
		node.Note = "depth limit reached"
		return node
	}
	if recursiveSchema(schema, ancestorRefs) {
		// ref-based recursion is shown explicitly in the tree so users can see the edge
		// without following an infinite expansion chain.
		node.Note = "recursive reference"
		return node
	}

	nextRefs := nextAncestorRefs(ancestorRefs, schema)
	children := schemaChildren(node, schema, depth+1, nextRefs)
	for _, child := range children {
		child.Parent = node
	}
	node.Children = children
	return node
}

func schemaChildren(parent *treeNode, schema *model.Schema, depth int, ancestorRefs []string) []*treeNode {
	if schema == nil {
		return nil
	}

	children := make([]*treeNode, 0, len(schema.Properties)+1+len(schema.OneOf)+len(schema.AnyOf)+len(schema.AllOf))
	if len(schema.Properties) > 0 {
		names := make([]string, 0, len(schema.Properties))
		for name := range schema.Properties {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			childID := parent.ID + "/property:" + name
			children = append(children, buildSchemaNode(childID, nodeLabel{
				Prefix: "Property:",
				Name:   name,
				Meta:   describe.SchemaTypeHint(schema.Properties[name]),
			}, schema.Properties[name], parent, depth, ancestorRefs))
		}
	}
	if schema.Items != nil {
		children = append(children, buildSchemaNode(parent.ID+"/items", nodeLabel{
			Prefix: "Array item:",
			Name:   "items",
			Meta:   describe.SchemaTypeHint(schema.Items),
		}, schema.Items, parent, depth, ancestorRefs))
	}
	children = append(children, compositionRows(parent, "oneOf", schema.OneOf, depth, ancestorRefs)...)
	children = append(children, compositionRows(parent, "anyOf", schema.AnyOf, depth, ancestorRefs)...)
	children = append(children, compositionRows(parent, "allOf", schema.AllOf, depth, ancestorRefs)...)
	return children
}

func compositionRows(parent *treeNode, prefix string, schemas []*model.Schema, depth int, ancestorRefs []string) []*treeNode {
	rows := make([]*treeNode, 0, len(schemas))
	for index, schema := range schemas {
		label := fmt.Sprintf("%s[%d]", prefix, index)
		rows = append(rows, buildSchemaNode(parent.ID+"/"+label, nodeLabel{
			Prefix: "Composition:",
			Name:   label,
			Meta:   describe.SchemaTypeHint(schema),
		}, schema, parent, depth, ancestorRefs))
	}
	return rows
}

func recursiveSchema(schema *model.Schema, ancestorRefs []string) bool {
	if schema == nil || strings.TrimSpace(schema.Ref) == "" {
		return false
	}

	ref := strings.TrimSpace(schema.Ref)
	return slices.Contains(ancestorRefs, ref)
}

func nextAncestorRefs(ancestorRefs []string, schema *model.Schema) []string {
	if schema == nil || strings.TrimSpace(schema.Ref) == "" {
		return append([]string(nil), ancestorRefs...)
	}

	next := append([]string(nil), ancestorRefs...)
	next = append(next, strings.TrimSpace(schema.Ref))
	return next
}

func visibleRows(operation *model.Operation, state State) []visibleRow {
	roots := rootNodes(operation)
	if len(roots) == 0 {
		return nil
	}

	rows := make([]visibleRow, 0)
	for index, root := range roots {
		appendVisibleRows(&rows, root, 0, nil, index < len(roots)-1, state.ExpandedNodeIDs)
	}

	return rows
}

func appendVisibleRows(rows *[]visibleRow, node *treeNode, depth int, ancestorHasNext []bool, hasNextSibling bool, expanded map[string]struct{}) {
	row := visibleRow{
		Node:            node,
		Depth:           depth,
		Expanded:        isExpanded(expanded, node.ID),
		HasNextSibling:  hasNextSibling,
		AncestorHasNext: append([]bool(nil), ancestorHasNext...),
	}
	*rows = append(*rows, row)
	if !isExpanded(expanded, node.ID) || !expandable(node) {
		return
	}

	// Persisting the ancestor sibling state lets the renderer draw a stable inline tree
	// without recomputing parent relationships during styling.
	nextAncestors := append(append([]bool(nil), ancestorHasNext...), hasNextSibling)
	for index, child := range node.Children {
		appendVisibleRows(rows, child, depth+1, nextAncestors, index < len(node.Children)-1, expanded)
	}
}

func initialExpandedNodeIDs(operation *model.Operation) map[string]struct{} {
	expanded := make(map[string]struct{})
	for _, root := range rootNodes(operation) {
		expanded[root.ID] = struct{}{}
	}
	return expanded
}

func activeVisibleRow(rows []visibleRow, activeRow int) (visibleRow, bool) {
	if len(rows) == 0 {
		return visibleRow{}, false
	}

	index := util.Clamp(activeRow, 0, len(rows)-1)
	return rows[index], true
}

func previewTitle(node *treeNode) string {
	if node == nil {
		return "Preview"
	}

	return previewLabel(node)
}

func previewLabel(node *treeNode) string {
	if node == nil {
		return ""
	}

	parts := make([]string, 0, 2)
	if prefix := strings.TrimSpace(node.Label.Prefix); prefix != "" {
		parts = append(parts, prefix)
	}
	if name := strings.TrimSpace(node.Label.Name); name != "" {
		parts = append(parts, name)
	}

	return strings.TrimSpace(strings.Join(parts, " "))
}

func previewPath(node *treeNode) string {
	if node == nil {
		return ""
	}

	parts := make([]string, 0, 4)
	for current := node; current != nil; current = current.Parent {
		label := previewLabel(current)
		if strings.TrimSpace(label) == "" {
			continue
		}
		parts = append(parts, label)
	}
	slices.Reverse(parts)
	return strings.Join(parts, " > ")
}

func expandable(node *treeNode) bool {
	return node != nil && len(node.Children) > 0 && strings.TrimSpace(node.Note) == ""
}

func isExpanded(expanded map[string]struct{}, id string) bool {
	if len(expanded) == 0 {
		return false
	}

	_, ok := expanded[id]
	return ok
}

func parameterGroupID(location model.ParameterLocation) string {
	switch location {
	case model.ParameterLocationPath:
		return groupIDPathParams
	case model.ParameterLocationQuery:
		return groupIDQueryParams
	case model.ParameterLocationHeader:
		return groupIDHeaderParams
	case model.ParameterLocationCookie:
		return groupIDCookieParams
	case model.ParameterLocationForm:
		return groupIDFormParams
	default:
		return groupIDQueryParams
	}
}

func singularParameterPrefix(location model.ParameterLocation) string {
	switch location {
	case model.ParameterLocationPath:
		return "Path param"
	case model.ParameterLocationQuery:
		return "Query param"
	case model.ParameterLocationHeader:
		return "Header param"
	case model.ParameterLocationCookie:
		return "Cookie param"
	case model.ParameterLocationForm:
		return "Form param"
	default:
		return "Param"
	}
}

func singularParameterContentPrefix(location model.ParameterLocation) string {
	switch location {
	case model.ParameterLocationPath:
		return "Path content"
	case model.ParameterLocationQuery:
		return "Query content"
	case model.ParameterLocationHeader:
		return "Header content"
	case model.ParameterLocationCookie:
		return "Cookie content"
	case model.ParameterLocationForm:
		return "Form content"
	default:
		return "Parameter content"
	}
}

func rowIndexByNodeID(rows []visibleRow, id string) int {
	for index, row := range rows {
		if row.Node != nil && row.Node.ID == id {
			return index
		}
	}

	return -1
}
