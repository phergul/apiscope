package widgets

import "github.com/charmbracelet/lipgloss"

// CenteredModalData describes a centered shell modal.
type CenteredModalData struct {
	Body  string
	Width int
}

// BottomRightOverlayData describes a popup anchored to the bottom-right of a shell view.
type BottomRightOverlayData struct {
	Base        string
	Popup       string
	BottomInset int
}

// RenderCenteredModal renders a centered modal within the available viewport.
func RenderCenteredModal(viewportWidth, viewportHeight int, data CenteredModalData) string {
	// remove the modal frame chrome before sizing the wrapped body content.
	modal := ModalStyle(max(data.Width-4, 1)).Render(data.Body)
	return lipgloss.Place(viewportWidth, viewportHeight, lipgloss.Center, lipgloss.Center, modal)
}

// OverlayBottomRight anchors a popup to the bottom-right corner of a rendered shell view.
func OverlayBottomRight(data BottomRightOverlayData) string {
	// align the popup flush with the right edge of the rendered shell body.
	x := max(lipgloss.Width(data.Base)-lipgloss.Width(data.Popup), 0)
	// keep the popup above reserved bottom rows such as the status bar.
	y := max(lipgloss.Height(data.Base)-data.BottomInset-lipgloss.Height(data.Popup), 0)
	return Overlay(data.Base, data.Popup, x, y)
}
