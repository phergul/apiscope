package model

type DraftKey string

func NewDraftKey(specFingerprint SpecFingerprint, operationKey OperationKey) DraftKey {
	return DraftKey(string(specFingerprint) + "::" + operationKey.String())
}

type RequestDraft struct {
	Key              DraftKey          `json:"key"`
	SpecFingerprint  SpecFingerprint   `json:"spec_fingerprint"`
	OperationKey     OperationKey      `json:"operation_key"`
	ServerURL        string            `json:"server_url,omitempty"`
	PathParams       map[string]string `json:"path_params,omitempty"`
	QueryParams      map[string]string `json:"query_params,omitempty"`
	HeaderParams     map[string]string `json:"header_params,omitempty"`
	CookieParams     map[string]string `json:"cookie_params,omitempty"`
	FormParams       map[string]string `json:"form_params,omitempty"`
	FormFileParams   map[string]string `json:"form_file_params,omitempty"`
	BodyMediaType    string            `json:"body_media_type,omitempty"`
	BodyRaw          string            `json:"body_raw,omitempty"`
	SelectedExamples map[string]string `json:"selected_examples,omitempty"`
}
