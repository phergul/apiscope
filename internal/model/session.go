package model

type AuthSchemeValueType string

const (
	AuthSchemeValueTypeAPIKey AuthSchemeValueType = "apiKey"
	AuthSchemeValueTypeBasic  AuthSchemeValueType = "basic"
	AuthSchemeValueTypeBearer AuthSchemeValueType = "bearer"
)

type FocusedPane string

const (
	FocusedPaneOperations FocusedPane = "operations"
	FocusedPaneDetails    FocusedPane = "details"
	FocusedPaneRequest    FocusedPane = "request"
	FocusedPaneResponse   FocusedPane = "response"
)

type EditorMode string

const (
	EditorModeBrowse EditorMode = "browse"
	EditorModeEdit   EditorMode = "edit"
	EditorModeFilter EditorMode = "filter"
)

type SessionState struct {
	SpecSource           string
	SpecFingerprint      SpecFingerprint
	Spec                 *APISpec
	SelectedServerURL    string
	SelectedOperationKey OperationKey
	RequestDrafts        map[DraftKey]*RequestDraft
	AuthState            map[string]AuthValue
	LastResponse         *HTTPResponse
	RequestHistory       []HistoryEntry
	ActiveLoadRequestID  uint64
	ActiveExecRequestID  uint64
}

type ViewState struct {
	FocusedPane            FocusedPane
	FilterText             string
	VisibleOperationKeys   []OperationKey
	OperationsCursor       int
	DetailsScrollOffset    int
	RequestScrollOffset    int
	ResponseScrollOffset   int
	ActiveEditorMode       EditorMode
	OperationsPaneVisible  bool
	ResponsePaneExpanded   bool
	RightPaneLayoutPreset  string
	Notice                 string
	LoadInFlight           bool
	ExecuteInFlight        bool
	ActiveLoadRequestID    uint64
	ActiveExecuteRequestID uint64
}

type AuthValue struct {
	Type        AuthSchemeValueType
	APIKey      string
	Username    string
	Password    string
	BearerToken string
}

type HistoryEntry struct {
	RequestID     uint64
	OperationKey  OperationKey
	ServerURL     string
	Response      *HTTPResponse
	TransportNote string
}
