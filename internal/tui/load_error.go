package tui

import (
	"fmt"
	"strings"

	"github.com/phergul/apiscope/internal/app"
)

func (m *Model) loadErrorView() app.LoadErrorView {
	return app.DescribeLoadError(m.loadErr, m.source)
}

func (m *Model) renderLoadErrorContent() string {
	view := m.loadErrorView()
	lines := []string{
		view.Title,
		"",
		fmt.Sprintf("Category: %s", view.Category),
		fmt.Sprintf("Source: %s", fallbackText(view.Source, m.source)),
		"",
		view.Summary,
		"",
		fmt.Sprintf("Try this: %s", view.Hint),
	}

	return strings.Join(lines, "\n")
}

func (m *Model) hasBlockingLoadError() bool {
	return m.loadErr != nil
}

func fallbackText(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}

	return value
}
