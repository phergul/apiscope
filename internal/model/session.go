package model

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

type RequestEditKind string

const (
	RequestEditKindNone  RequestEditKind = ""
	RequestEditKindField RequestEditKind = "field"
	RequestEditKindBody  RequestEditKind = "body"
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
	ExpandedRightPane      FocusedPane
	FilterText             string
	VisibleOperationKeys   []OperationKey
	OperationsCursor       int
	OperationsScrollOffset int
	DetailsScrollOffset    int
	RequestActiveRow       int
	RequestScrollOffset    int
	ResponseScrollOffset   int
	ActiveEditorMode       EditorMode
	RequestEditKind        RequestEditKind
	RequestEditBuffer      string
	RequestEditTarget      string
	ZoomedPane             bool
	OperationsPaneVisible  bool
	RightPaneLayoutPreset  string
	Notice                 string
	LoadInFlight           bool
	ExecuteInFlight        bool
	ActiveLoadRequestID    uint64
	ActiveExecuteRequestID uint64
}

type HistoryEntry struct {
	RequestID     uint64
	OperationKey  OperationKey
	ServerURL     string
	Request       ExecutedRequestSnapshot
	Response      *HTTPResponse
	TransportNote string
}
