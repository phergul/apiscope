package widgets

import bubbleviewport "github.com/charmbracelet/bubbles/viewport"

type Viewport struct {
	model bubbleviewport.Model
}

func NewViewport(width, height int) Viewport {
	model := bubbleviewport.New(width, height)
	model.Style = ViewportStyle()
	return Viewport{model: model}
}

func (v *Viewport) SetSize(width, height int) {
	v.model.Width = width
	v.model.Height = height
	v.model.Style = ViewportStyle()
}

func (v *Viewport) SetContent(content string) {
	v.model.SetContent(content)
}

func (v *Viewport) SetYOffset(offset int) {
	v.model.SetYOffset(offset)
}

func (v Viewport) View() string {
	return v.model.View()
}
