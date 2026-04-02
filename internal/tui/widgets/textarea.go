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

func (t *TextArea) refreshTheme() {
	t.ensure()
	applyTextAreaTheme(&t.model)
}

// applyTextAreaTheme applies the shared textarea theme.
func applyTextAreaTheme(model *bubbletextarea.Model) {
	bg := lipgloss.NewStyle().Background(CurrentTheme().Palette.InputBackground)

	focused := model.FocusedStyle
	focused.Base = bg
	focused.CursorLine = bg
	focused.Prompt = bg
	focused.EndOfBuffer = bg
	focused.LineNumber = bg
	focused.CursorLineNumber = bg
	focused.Placeholder = InputPlaceholderStyle()
	focused.Text = InputTextStyle()
	model.FocusedStyle = focused

	blurred := model.BlurredStyle
	blurred.Base = bg
	blurred.Prompt = bg
	blurred.EndOfBuffer = bg
	blurred.LineNumber = bg
	blurred.CursorLineNumber = bg
	blurred.Placeholder = InputPlaceholderStyle()
	blurred.Text = InputTextStyle()
	model.BlurredStyle = blurred
}

func (t *TextArea) Focus() {
	t.refreshTheme()
	t.model.Focus()
}

func (t *TextArea) Blur() {
	t.refreshTheme()
	t.model.Blur()
}

func (t *TextArea) SetValue(value string) {
	t.refreshTheme()
	t.model.SetValue(value)
}

func (t *TextArea) SetPlaceholder(value string) {
	t.refreshTheme()
	t.model.Placeholder = value
}

func (t TextArea) Value() string {
	if !t.initialized {
		return ""
	}
	return t.model.Value()
}

func (t *TextArea) SetSize(width, height int) {
	t.refreshTheme()
	t.model.SetWidth(width)
	t.model.SetHeight(height)
}

func (t *TextArea) Update(msg tea.Msg) tea.Cmd {
	t.refreshTheme()
	var cmd tea.Cmd
	t.model, cmd = t.model.Update(msg)
	return cmd
}

// renderContent returns the themed textarea content, preserving a full filled
// background for empty placeholder states without storing ANSI in the model.
func (t TextArea) renderContent() string {
	content := t.model.View()
	if t.model.Value() == "" && t.model.Placeholder != "" {
		content = InputPlaceholderStyle().Render(t.model.Placeholder)
	}

	return RenderFilledInputArea(content, t.model.Width(), t.model.Height())
}

func (t TextArea) View() string {
	if !t.initialized {
		t = NewTextArea()
	}
	applyTextAreaTheme(&t.model)

	return InputFrameStyle(t.model.Focused()).Render(
		t.renderContent(),
	)
}

func (t TextArea) BareView() string {
	if !t.initialized {
		t = NewTextArea()
	}
	applyTextAreaTheme(&t.model)

	return t.renderContent()
}
