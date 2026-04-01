package model

// ExecutedRequestSnapshot stores the immutable request inputs used for one execution.
type ExecutedRequestSnapshot struct {
	OperationKey OperationKey
	ServerURL    string
	Draft        RequestDraft
	AuthState    map[string]AuthValue
}
