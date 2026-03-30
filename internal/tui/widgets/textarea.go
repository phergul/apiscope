package widgets

import (
	bubbletextarea "github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type TextArea struct {
	model       bubbletextarea.Model
	initialized bool
}

func NewTextArea() TextArea {
	model := bubbletextarea.New()
	model.Prompt = ""
	model.ShowLineNumbers = false
	model.SetWidth(40)
	model.SetHeight(6)
	applyTextAreaTheme(&model)
	return TextArea{model: model, initialized: true}
}

func (t *TextArea) ensure() {
	if t.initialized {
		return
	}
	*t = NewTextArea()
}

func applyTextAreaTheme(model *bubbletextarea.Model) {
	focused := model.FocusedStyle
	focused.Base = lipgloss.NewStyle()
	focused.CursorLine = lipgloss.NewStyle()
	focused.Placeholder = InputPlaceholderStyle()
	focused.Text = InputTextStyle()
	model.FocusedStyle = focused

	blurred := model.BlurredStyle
	blurred.Base = lipgloss.NewStyle()
	blurred.Placeholder = InputPlaceholderStyle()
	blurred.Text = InputTextStyle()
	model.BlurredStyle = blurred
}

func (t *TextArea) Focus() {
	t.ensure()
	t.model.Focus()
}

func (t *TextArea) Blur() {
	t.ensure()
	t.model.Blur()
}

func (t *TextArea) SetValue(value string) {
	t.ensure()
	t.model.SetValue(value)
}

func (t *TextArea) SetPlaceholder(value string) {
	t.ensure()
	t.model.Placeholder = value
}

func (t TextArea) Value() string {
	if !t.initialized {
		return ""
	}
	return t.model.Value()
}

func (t *TextArea) SetSize(width, height int) {
	t.ensure()
	t.model.SetWidth(width)
	t.model.SetHeight(height)
}

func (t *TextArea) Update(msg tea.Msg) tea.Cmd {
	t.ensure()
	var cmd tea.Cmd
	t.model, cmd = t.model.Update(msg)
	return cmd
}

func (t TextArea) View() string {
	if !t.initialized {
		t = NewTextArea()
	}
	return InputFrameStyle(t.model.Focused()).Render(t.model.View())
}

func (t TextArea) BareView() string {
	if !t.initialized {
		t = NewTextArea()
	}
	return t.model.View()
}
