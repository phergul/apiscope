package request

import "github.com/phergul/apiscope/internal/model"

type ValidationState struct {
	MessagesBySection []string
	RowErrors         map[string]string
}

type PaneInput struct {
	LoadInFlight  bool
	Selected      *model.Operation
	Draft         *model.RequestDraft
	Security      *model.SecurityRequirement
	ActiveSection string
	ActiveRow     int
	ScrollOffset  int
	Validation    ValidationState
	Editor        EditorInput
	ContentWidth  int
	ContentHeight int
	HelpOpen      bool
}

type PaneProjection struct {
	Data        Data
	HelpOverlay HelpView
}

// ProjectPane projects root request state into a rendered request-pane view model.
func ProjectPane(input PaneInput) PaneProjection {
	data, editorState := projectPaneData(input)
	help := HelpView{Hint: BuildHelpView(editorState).Hint}
	if input.HelpOpen {
		help = BuildHelpView(editorState)
	}

	if input.ContentHeight > 0 && !data.LoadInFlight && len(data.Sections) > 0 {
		data = WindowVisibleRows(data, input.ScrollOffset, max(input.ContentHeight, 1))
	}

	return PaneProjection{
		Data:        data,
		HelpOverlay: help,
	}
}

// projectPaneData builds the unwindowed request pane data and active editor state.
func projectPaneData(input PaneInput) (Data, EditorState) {
	data := Data{
		LoadInFlight:  input.LoadInFlight,
		ContentWidth:  input.ContentWidth,
		ContentHeight: input.ContentHeight,
	}

	if input.Selected == nil {
		data.EmptyState = "No operation selected.\nChoose an operation in pane 1 to inspect request details."
		return data, EditorState{}
	}

	activeSection := ResolveActiveSection(input.ActiveSection, input.Selected, input.Security)
	sections := AvailableSections(input.Selected, input.Security)
	rows := ActiveRows(input.Selected, input.Draft, activeSection, input.Security)
	editorState := BuildEditorState(input.Editor, rows, input.ActiveRow, input.Selected, input.Draft)

	data.Sections = sections
	data.ActiveSection = activeSection
	data.ActiveRow = input.ActiveRow
	data.Edit = BuildEditView(editorState)
	data.ValidationNotice = input.Validation.MessagesBySection
	data.Rows = projectRows(rows, input.Validation.RowErrors)
	if len(data.Sections) == 0 {
		data.EmptyState = "This operation does not declare request parameters, request body, or auth requirements."
	}

	return data, editorState
}

// projectRows projects internal request row descriptors into render rows with validation errors.
func projectRows(rows []RowDescriptor, rowErrors map[string]string) []Row {
	projected := make([]Row, 0, len(rows))
	for _, row := range rows {
		projected = append(projected, Row{
			Kind:     row.Kind,
			Label:    row.Label,
			Meta:     row.Meta,
			Value:    row.Value,
			Editable: row.Editable,
			Error:    rowErrors[row.ID],
		})
	}

	return projected
}
