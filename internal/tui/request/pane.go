package request

import "github.com/phergul/apiscope/internal/model"

// SupportSeverity describes how strongly the request pane should call out a support note.
type SupportSeverity string

const (
	SupportSeverityUnsupported SupportSeverity = "unsupported"
	SupportSeverityDowngraded  SupportSeverity = "downgraded"
)

// SupportNote describes one non-blocking support note rendered in the request pane.
type SupportNote struct {
	Severity SupportSeverity
	Summary  string
	Detail   string
}

type ValidationState struct {
	MessagesBySection []string
	RowErrors         map[string]string
}

// SupportState carries section-level and row-level support notes for request rendering.
type SupportState struct {
	MessagesBySection []SupportNote
	RowNotes          map[string][]SupportNote
}

// PaneInput contains the root-owned request pane state needed to project a render model.
type PaneInput struct {
	LoadInFlight      bool
	Selected          *model.Operation
	Draft             *model.RequestDraft
	Security          *model.SecurityRequirement
	Servers           []model.Server
	SelectedServerURL string
	SecuritySchemes   map[string]model.SecurityScheme
	AuthState         map[string]model.AuthValue
	ActiveSection     string
	ActiveRow         int
	ScrollOffset      int
	Validation        ValidationState
	Support           SupportState
	Editor            EditorInput
	ContentWidth      int
	ContentHeight     int
	HelpOpen          bool
}

// PaneProjection contains the rendered request pane data and any detached help overlay.
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

	activeSection := ResolveActiveSection(input.ActiveSection, input.Selected, input.Security, input.Servers)
	sections := AvailableSections(input.Selected, input.Security, input.Servers)
	rows := ActiveRows(input.Selected, input.Draft, activeSection, input.Security, input.Servers, input.SelectedServerURL, input.SecuritySchemes, input.AuthState)
	editorState := BuildEditorState(input.Editor, rows, input.ActiveRow, input.Selected, input.Draft)

	data.Sections = sections
	data.ActiveSection = activeSection
	data.ActiveRow = input.ActiveRow
	data.Edit = BuildEditView(editorState)
	data.ValidationNotice = input.Validation.MessagesBySection
	data.SupportNotice = input.Support.MessagesBySection
	data.Rows = projectRows(rows, input.Validation.RowErrors, input.Support.RowNotes)
	if len(data.Sections) == 0 {
		data.EmptyState = "This operation does not declare request parameters, request body, or auth requirements."
	}

	return data, editorState
}

// projectRows projects internal request row descriptors into render rows with validation errors.
func projectRows(rows []RowDescriptor, rowErrors map[string]string, rowNotes map[string][]SupportNote) []Row {
	projected := make([]Row, 0, len(rows))
	for _, row := range rows {
		target := row.ValidationTarget
		if target == "" {
			target = row.ID
		}
		projected = append(projected, Row{
			Kind:     row.Kind,
			Label:    row.Label,
			Meta:     row.Meta,
			Value:    row.Value,
			Editable: row.Editable,
			Error:    rowErrors[target],
			Support:  append([]SupportNote(nil), rowNotes[target]...),
		})
	}

	return projected
}
