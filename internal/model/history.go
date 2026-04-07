package model

// ExecutedRequestSnapshot stores the immutable request inputs used for one execution.
type ExecutedRequestSnapshot struct {
	OperationKey OperationKey `json:"operation_key"`
	ServerURL    string       `json:"server_url,omitempty"`
	Draft        RequestDraft `json:"draft"`
	// AuthState stays in memory so the current session can inspect its own executions
	// without writing secrets into durable history storage.
	AuthState map[string]AuthValue `json:"-"`
}
