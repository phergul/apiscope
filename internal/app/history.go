package app

import (
	"github.com/phergul/apiscope/internal/model"
)

// CloneExecutionSession copies the mutable execution inputs so asynchronous requests
// keep using the exact values that were present when execution started.
func CloneExecutionSession(session model.SessionState) model.SessionState {
	session.RequestDrafts = cloneRequestDraftMap(session.RequestDrafts)
	session.AuthState = cloneAuthState(session.AuthState)
	session.LastResponse = cloneHTTPResponse(session.LastResponse)
	session.RequestHistory = cloneHistoryEntries(session.RequestHistory)
	return session
}

// BuildExecutedRequestSnapshot captures the executed request inputs for history recall.
func BuildExecutedRequestSnapshot(session model.SessionState, draft *model.RequestDraft) model.ExecutedRequestSnapshot {
	snapshot := model.ExecutedRequestSnapshot{
		ServerURL: session.SelectedServerURL,
		AuthState: cloneAuthState(session.AuthState),
	}
	if draft != nil {
		snapshot.OperationKey = draft.OperationKey
		snapshot.Draft = *cloneRequestDraft(draft)
	}

	return snapshot
}

// HistoryForOperation returns newest-first history entries for one operation.
func HistoryForOperation(session model.SessionState, operationKey model.OperationKey) []model.HistoryEntry {
	if operationKey == "" || len(session.RequestHistory) == 0 {
		return nil
	}

	entries := make([]model.HistoryEntry, 0, len(session.RequestHistory))
	for index := len(session.RequestHistory) - 1; index >= 0; index-- {
		entry := session.RequestHistory[index]
		if entry.OperationKey != operationKey {
			continue
		}
		entries = append(entries, cloneHistoryEntry(entry))
	}

	return entries
}

// LoadHistoryResponse restores only the stored response from one history entry.
func LoadHistoryResponse(session *model.SessionState, entry model.HistoryEntry) bool {
	if session == nil || entry.Response == nil {
		return false
	}

	session.LastResponse = cloneHTTPResponse(entry.Response)
	return true
}

// RestoreHistoryRequest restores request inputs from one history entry so the user can rerun it.
func RestoreHistoryRequest(session *model.SessionState, entry model.HistoryEntry) bool {
	if session == nil {
		return false
	}

	if entry.Request.ServerURL != "" {
		SetSelectedServer(session, entry.Request.ServerURL)
	}

	if session.RequestDrafts == nil {
		session.RequestDrafts = make(map[model.DraftKey]*model.RequestDraft)
	}
	draft := cloneRequestDraft(&entry.Request.Draft)
	if draft != nil {
		session.RequestDrafts[draft.Key] = draft
	}

	session.AuthState = cloneAuthState(entry.Request.AuthState)
	return true
}

func cloneHistoryEntries(entries []model.HistoryEntry) []model.HistoryEntry {
	if len(entries) == 0 {
		return nil
	}

	cloned := make([]model.HistoryEntry, 0, len(entries))
	for _, entry := range entries {
		cloned = append(cloned, cloneHistoryEntry(entry))
	}

	return cloned
}

func cloneHistoryEntry(entry model.HistoryEntry) model.HistoryEntry {
	entry.Request = cloneExecutedRequestSnapshot(entry.Request)
	entry.Response = cloneHTTPResponse(entry.Response)
	return entry
}

func cloneExecutedRequestSnapshot(snapshot model.ExecutedRequestSnapshot) model.ExecutedRequestSnapshot {
	snapshot.Draft = *cloneRequestDraft(&snapshot.Draft)
	snapshot.AuthState = cloneAuthState(snapshot.AuthState)
	return snapshot
}

func cloneRequestDraftMap(drafts map[model.DraftKey]*model.RequestDraft) map[model.DraftKey]*model.RequestDraft {
	if len(drafts) == 0 {
		return nil
	}

	cloned := make(map[model.DraftKey]*model.RequestDraft, len(drafts))
	for key, draft := range drafts {
		cloned[key] = cloneRequestDraft(draft)
	}

	return cloned
}

func cloneRequestDraft(draft *model.RequestDraft) *model.RequestDraft {
	if draft == nil {
		return nil
	}

	cloned := *draft
	cloned.PathParams = cloneStringMap(draft.PathParams)
	cloned.QueryParams = cloneStringMap(draft.QueryParams)
	cloned.HeaderParams = cloneStringMap(draft.HeaderParams)
	cloned.CookieParams = cloneStringMap(draft.CookieParams)
	cloned.SelectedExamples = cloneStringMap(draft.SelectedExamples)
	return &cloned
}

func cloneAuthState(state map[string]model.AuthValue) map[string]model.AuthValue {
	if len(state) == 0 {
		return nil
	}

	cloned := make(map[string]model.AuthValue, len(state))
	for name, value := range state {
		cloned[name] = value
	}

	return cloned
}

func cloneHTTPResponse(response *model.HTTPResponse) *model.HTTPResponse {
	if response == nil {
		return nil
	}

	cloned := *response
	cloned.Headers = cloneHeaderMap(response.Headers)
	cloned.Body = append([]byte(nil), response.Body...)
	return &cloned
}

func cloneHeaderMap(headers map[string][]string) map[string][]string {
	if len(headers) == 0 {
		return nil
	}

	cloned := make(map[string][]string, len(headers))
	for name, values := range headers {
		cloned[name] = append([]string(nil), values...)
	}

	return cloned
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}

	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}

	return cloned
}
