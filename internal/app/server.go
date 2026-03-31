package app

import (
	"strings"

	"github.com/phergul/apiscope/internal/model"
)

// SetSelectedServer updates the session-wide selected server and keeps known drafts in sync.
func SetSelectedServer(session *model.SessionState, serverURL string) bool {
	if session == nil {
		return false
	}

	serverURL = strings.TrimSpace(serverURL)
	if serverURL == "" {
		return false
	}
	if session.SelectedServerURL == serverURL {
		return false
	}

	session.SelectedServerURL = serverURL
	for _, draft := range session.RequestDrafts {
		if draft == nil {
			continue
		}
		draft.ServerURL = serverURL
	}

	return true
}

// CycleSelectedServer advances the session-wide selected server through the provided server list.
func CycleSelectedServer(session *model.SessionState, servers []model.Server) bool {
	if session == nil {
		return false
	}

	urls := make([]string, 0, len(servers))
	for _, server := range servers {
		if strings.TrimSpace(server.URL) == "" {
			continue
		}
		urls = append(urls, server.URL)
	}
	if len(urls) == 0 {
		return false
	}

	current := strings.TrimSpace(session.SelectedServerURL)
	if current == "" {
		return SetSelectedServer(session, urls[0])
	}

	for index, url := range urls {
		if url != current {
			continue
		}

		return SetSelectedServer(session, urls[(index+1)%len(urls)])
	}

	return SetSelectedServer(session, urls[0])
}
