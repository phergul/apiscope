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

func applyTextInputTheme(model *bubbletextinput.Model) {
	model.TextStyle = InputTextStyle()
	model.PlaceholderStyle = InputPlaceholderStyle()
	model.Cursor.Style = InputCursorStyle()
}

func (i *TextInput) Focus() {
	i.ensure()
	i.model.Focus()
}

func (i *TextInput) Blur() {
	i.ensure()
	i.model.Blur()
}

func (i *TextInput) SetValue(value string) {
	i.ensure()
	i.model.SetValue(value)
}

func (i *TextInput) SetPlaceholder(value string) {
	i.ensure()
	i.model.Placeholder = value
}

func (i TextInput) Value() string {
	if !i.initialized {
		return ""
	}
	return i.model.Value()
}

func (i *TextInput) SetWidth(width int) {
	i.ensure()
	i.model.Width = width
}

func (i *TextInput) Update(msg tea.Msg) tea.Cmd {
	i.ensure()
	var cmd tea.Cmd
	i.model, cmd = i.model.Update(msg)
	return cmd
}

func (i TextInput) View() string {
	if !i.initialized {
		i = NewTextInput()
	}

	return InputFrameStyle(i.model.Focused()).Render(
		InputAreaStyle(i.model.Width, 1).Render(i.model.View()),
	)
}

func (i TextInput) BareView() string {
	if !i.initialized {
		i = NewTextInput()
	}

	return InputAreaStyle(i.model.Width, 1).Render(i.model.View())
}

func (i TextInput) BareFilledView() string {
	if !i.initialized {
		i = NewTextInput()
	}

	return RenderFilledInputArea(i.model.View(), i.model.Width, 1)
}
