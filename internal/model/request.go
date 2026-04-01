package model

type DraftKey string

func NewDraftKey(specFingerprint SpecFingerprint, operationKey OperationKey) DraftKey {
	return DraftKey(string(specFingerprint) + "::" + operationKey.String())
}

type RequestDraft struct {
	Key              DraftKey
	SpecFingerprint  SpecFingerprint
	OperationKey     OperationKey
	ServerURL        string
	PathParams       map[string]string
	QueryParams      map[string]string
	HeaderParams     map[string]string
	CookieParams     map[string]string
	FormParams       map[string]string
	BodyMediaType    string
	BodyRaw          string
	SelectedExamples map[string]string
}
