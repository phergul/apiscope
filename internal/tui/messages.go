package tui

import "github.com/phergul/apiscope/internal/app"

type specLoadedMsg struct {
	requestID uint64
	result    app.LoadResult
	err       error
}

type executeFinishedMsg struct {
	requestID uint64
	result    app.ExecuteResult
}
