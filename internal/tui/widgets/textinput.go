package widgets

import (
	bubbletextinput "github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type TextInput struct {
	model       bubbletextinput.Model
	initialized bool
}

func NewTextInput() TextInput {
	model := bubbletextinput.New()
	model.Prompt = ""
	model.Width = 24
	applyTextInputTheme(&model)
	return TextInput{model: model, initialized: true}
}

func (i *TextInput) ensure() {
	if i.initialized {
		return
	}
	*i = NewTextInput()
}

func (i *TextInput) refreshTheme() {
	i.ensure()
	applyTextInputTheme(&i.model)
}

func applyTextInputTheme(model *bubbletextinput.Model) {
	model.TextStyle = InputTextStyle()
	model.PlaceholderStyle = InputPlaceholderStyle()
	model.Cursor.Style = InputCursorStyle()
	model.Cursor.TextStyle = InputCursorStyle()
}

func (i *TextInput) Focus() {
	i.refreshTheme()
	i.model.Focus()
}

func (i *TextInput) Blur() {
	i.refreshTheme()
	i.model.Blur()
}

func (i *TextInput) SetValue(value string) {
	i.refreshTheme()
	i.model.SetValue(value)
}

func (i *TextInput) SetPlaceholder(value string) {
	i.refreshTheme()
	i.model.Placeholder = value
}

func (i TextInput) Value() string {
	if !i.initialized {
		return ""
	}
	return i.model.Value()
}

func (i *TextInput) SetWidth(width int) {
	i.refreshTheme()
	i.model.Width = width
}

func (i *TextInput) Update(msg tea.Msg) tea.Cmd {
	i.refreshTheme()
	var cmd tea.Cmd
	i.model, cmd = i.model.Update(msg)
	return cmd
}

func (i TextInput) BareView() string {
	if !i.initialized {
		i = NewTextInput()
	}
	applyTextInputTheme(&i.model)

	return RenderFilledInputArea(i.model.View(), i.model.Width, 1)
}

func (i TextInput) BareFilledView() string {
	if !i.initialized {
		i = NewTextInput()
	}
	applyTextInputTheme(&i.model)

	return RenderFilledInputArea(i.model.View(), i.model.Width, 1)
}
