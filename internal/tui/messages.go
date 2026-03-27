package tui

import "api-tui/internal/app"

type specLoadedMsg struct {
	requestID uint64
	result    app.LoadResult
	err       error
}
