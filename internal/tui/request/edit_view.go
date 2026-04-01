package request

import (
	"strings"

	"github.com/phergul/apiscope/internal/model"
)

type EditorState struct {
	Kind           string
	Buffer         string
	View           string
	BodyMediaType  string
	ActiveRowLabel string
	ActiveRowMeta  string
}

type EditorInput struct {
	Kind      model.RequestEditKind
	Buffer    string
	FieldView string
	BodyView  string
}

// BuildEditorState projects low-level editor inputs into the request pane editor view state.
func BuildEditorState(input EditorInput, rows []RowDescriptor, activeRow int, selected *model.Operation, draft *model.RequestDraft) EditorState {
	state := EditorState{
		Kind:   string(input.Kind),
		Buffer: input.Buffer,
		View:   editorView(input),
	}

	switch input.Kind {
	case model.RequestEditKindBody:
		state.BodyMediaType = DraftBodyMediaType(selected, draft)
	case model.RequestEditKindField:
		if row := activeEditorRow(rows, activeRow); row != nil {
			state.ActiveRowLabel = row.Label
			state.ActiveRowMeta = row.Meta
		}
	}

	return state
}

// BuildEditView builds the popup view model for the active request editor.
func BuildEditView(state EditorState) EditView {
	return EditView{
		Kind:      state.Kind,
		Buffer:    state.Buffer,
		MediaType: state.BodyMediaType,
		View:      state.View,
		Title:     editTitle(state.Kind),
		Context:   editContext(state),
	}
}

// editorView selects the current widget view for the active request editor.
func editorView(input EditorInput) string {
	switch input.Kind {
	case model.RequestEditKindBody:
		return input.BodyView
	case model.RequestEditKindField:
		return input.FieldView
	default:
		return ""
	}
}

// activeEditorRow resolves the current row descriptor for field editing metadata.
func activeEditorRow(rows []RowDescriptor, activeRow int) *RowDescriptor {
	if len(rows) == 0 || activeRow < 0 {
		return nil
	}

	index := ClampActiveRow(activeRow, len(rows))
	return &rows[index]
}

// editTitle returns the popup title for the active request editor kind.
func editTitle(kind string) string {
	switch kind {
	case "body":
		return "Edit body"
	case "field":
		return "Edit value"
	default:
		return ""
	}
}

// editContext returns the contextual line shown above the active request editor.
func editContext(state EditorState) string {
	switch state.Kind {
	case "body":
		if strings.TrimSpace(state.BodyMediaType) == "" {
			return ""
		}
		return "Media type: " + state.BodyMediaType
	case "field":
		return formatRowContext(state.ActiveRowLabel, state.ActiveRowMeta)
	default:
		return ""
	}
}

// editHelpBody returns the inline help copy for the active request editor kind.
func editHelpBody(kind string) string {
	switch kind {
	case "body":
		return "Ctrl+S save\nEsc cancel\nEnter newline\n? toggle help"
	case "field":
		return "Enter save\nEsc cancel\n? toggle help"
	default:
		return ""
	}
}

// formatRowContext formats the active request row label and metadata for the editor popup.
func formatRowContext(label, meta string) string {
	label = strings.TrimSpace(label)
	meta = strings.TrimSpace(meta)
	if label == "" {
		return ""
	}
	if meta == "" {
		return label
	}

	return label + " (" + meta + ")"
}
