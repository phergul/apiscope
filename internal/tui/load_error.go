package tui

import (
	"fmt"
	"strings"

	"github.com/phergul/apiscope/internal/app"
	"github.com/phergul/apiscope/internal/tui/widgets"
	"github.com/phergul/apiscope/internal/util"
)

// loadErrorView returns the structured load error description for the current shell state.
func (m *Model) loadErrorView() app.LoadErrorView {
	return app.DescribeLoadError(m.shell.loadErr, m.shell.source)
}

// loadErrorBody renders the shared load-error body, optionally including the blocking quit action.
func (m *Model) loadErrorBody(includeQuit bool) string {
	view := m.loadErrorView()
	lines := []string{
		view.Title,
		"",
		fmt.Sprintf("Category: %s", view.Category),
		fmt.Sprintf("Source: %s", util.FallbackText(view.Source, m.shell.source)),
		"",
		view.Summary,
		"",
		fmt.Sprintf("Try this: %s", view.Hint),
	}
	if includeQuit {
		lines = append(lines, "", "[ Quit ]")
	}

	return strings.Join(lines, "\n")
}

// renderLoadErrorContent renders the non-blocking load-error content used inside the details pane.
func (m *Model) renderLoadErrorContent() string {
	return m.loadErrorBody(false)
}

// renderBlockingLoadError renders the centered blocking load-error modal.
func (m *Model) renderBlockingLoadError(width, height int) string {
	// size the blocking modal wide enough for the structured copy without overwhelming the screen.
	popupWidth := util.Clamp(int(float64(width)*0.68), 56, 92)
	return widgets.RenderCenteredModal(width, height, widgets.CenteredModalData{
		Body:  m.loadErrorBody(true),
		Width: popupWidth,
	})
}

// hasBlockingLoadError reports whether a load error should replace the pane layout.
func (m *Model) hasBlockingLoadError() bool {
	return m.shell.loadErr != nil
}
