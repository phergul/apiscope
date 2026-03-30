package request

import "strings"

type EditorState struct {
	Kind           string
	Buffer         string
	View           string
	BodyMediaType  string
	ActiveRowLabel string
	ActiveRowMeta  string
}

type HelpView struct {
	Hint  string
	Title string
	Body  string
}

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

func BuildHelpView(state EditorState) HelpView {
	if strings.TrimSpace(state.Kind) == "" {
		return HelpView{}
	}

	return HelpView{
		Hint:  "Help - ?",
		Title: "Help",
		Body:  editHelpBody(state.Kind),
	}
}

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
