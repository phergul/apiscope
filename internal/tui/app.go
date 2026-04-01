package tui

import (
	"context"
	"io"

	"github.com/phergul/apiscope/internal/app"
	"github.com/phergul/apiscope/internal/model"
	detailsui "github.com/phergul/apiscope/internal/tui/details"
	responseui "github.com/phergul/apiscope/internal/tui/response"
	"github.com/phergul/apiscope/internal/tui/widgets"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	defaultWidth       = 120
	defaultHeight      = 32
	layoutPresetWide   = "wide"
	layoutPresetNarrow = "narrow"
)

// Program wraps the Bubble Tea program used by the CLI entrypoint.
type Program struct {
	program *tea.Program
}

// shellState groups root-owned runtime shell state.
type shellState struct {
	width   int
	height  int
	source  string
	loadErr error
}

// paneState groups the root-owned active pane section state.
type paneState struct {
	activeDetailsSection  string
	activeRequestSection  string
	activeResponseSection string
}

// widgetState groups the Bubble widget models owned by the root shell.
type widgetState struct {
	filterInput       widgets.TextInput
	requestFieldInput widgets.TextInput
	requestBodyInput  widgets.TextArea
}

// requestUIState groups request-editor state that still belongs to the root adapters.
type requestUIState struct {
	validation app.RequestValidationResult
}

// historyUIState groups shell-owned previous-request popup state.
type historyUIState struct {
	open       bool
	activeRow  int
	filterText string
}

// helpUIState groups the root-owned contextual help overlay state.
type helpUIState struct {
	open bool
	view widgets.HelpView
}

// Model is the root Bubble Tea model for the TUI shell.
type Model struct {
	service   *app.Service
	session   model.SessionState
	viewState model.ViewState
	shell     shellState
	panes     paneState
	widgets   widgetState
	requestUI requestUIState
	historyUI historyUIState
	helpUI    helpUIState
}

// NewProgram builds the CLI-facing Bubble Tea program wrapper.
func NewProgram(service *app.Service, source string, input io.Reader, output io.Writer) *Program {
	options := []tea.ProgramOption{
		tea.WithAltScreen(),
		tea.WithInput(input),
		tea.WithOutput(output),
	}

	return &Program{
		program: tea.NewProgram(NewModel(service, source), options...),
	}
}

// Run starts the Bubble Tea program and waits for it to exit.
func (p *Program) Run() error {
	_, err := p.program.Run()
	return err
}

// NewModel builds the root TUI model with default shell and pane state.
func NewModel(service *app.Service, source string) *Model {
	if service == nil {
		service = app.NewService(nil, nil)
	}

	filterInput := widgets.NewTextInput()
	filterInput.SetPlaceholder("Filter operations")
	requestFieldInput := widgets.NewTextInput()
	requestFieldInput.SetPlaceholder("Enter value")
	requestBodyInput := widgets.NewTextArea()
	requestBodyInput.SetPlaceholder("Enter raw request body")

	return &Model{
		service: service,
		shell: shellState{
			source: source,
		},
		panes: paneState{
			activeDetailsSection: detailsui.SectionSummary,
		},
		widgets: widgetState{
			filterInput:       filterInput,
			requestFieldInput: requestFieldInput,
			requestBodyInput:  requestBodyInput,
		},
		viewState: model.ViewState{
			FocusedPane:           model.FocusedPaneOperations,
			ExpandedRightPane:     model.FocusedPaneRequest,
			ActiveEditorMode:      model.EditorModeBrowse,
			OperationsPaneVisible: true,
			ZoomedPane:            false,
			RightPaneLayoutPreset: layoutPresetNarrow,
		},
	}
}

// Init starts the initial spec load for the TUI model.
func (m *Model) Init() tea.Cmd {
	m.ensureWidgetDefaults()
	return m.startLoadCmd()
}

// Update applies one Bubble Tea message to the root shell model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.ensureWidgetDefaults()

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.shell.width = msg.Width
		m.shell.height = msg.Height
		m.viewState.RightPaneLayoutPreset = chooseLayoutPreset(msg.Width)
		m.ensureActiveOperationVisible()
		return m, nil
	case tea.KeyMsg:
		return m.updateKey(msg)
	case specLoadedMsg:
		if msg.requestID != m.viewState.ActiveLoadRequestID {
			return m, nil
		}

		m.shell.loadErr = msg.err
		if msg.err != nil {
			m.session.ActiveLoadRequestID = msg.requestID
			m.viewState.ActiveLoadRequestID = msg.requestID
			m.viewState.LoadInFlight = false
			m.viewState.Notice = "Spec load failed"
			return m, nil
		}

		m.session = msg.result.Session
		m.viewState = msg.result.View
		m.requestUI.validation = app.RequestValidationResult{}
		m.historyUI = historyUIState{}
		m.session.ActiveLoadRequestID = msg.requestID
		m.viewState.ActiveLoadRequestID = msg.requestID
		m.viewState.LoadInFlight = false
		m.viewState.RightPaneLayoutPreset = chooseLayoutPreset(m.shell.width)
		m.syncVisibleOperations()
		m.syncActivePaneSections()
		m.viewState.Notice = "Spec loaded"
		return m, nil
	case executeFinishedMsg:
		if msg.requestID != m.viewState.ActiveExecuteRequestID {
			return m, nil
		}

		m.viewState.ExecuteInFlight = false
		if msg.result.Response != nil {
			msg.result.Response.RequestID = msg.requestID
			m.session.LastResponse = msg.result.Response
			m.session.RequestHistory = append(m.session.RequestHistory, model.HistoryEntry{
				RequestID:     msg.requestID,
				OperationKey:  msg.result.OperationKey,
				ServerURL:     msg.result.ServerURL,
				Request:       msg.result.Snapshot,
				Response:      msg.result.Response,
				TransportNote: msg.result.Response.TransportError,
			})
		}
		m.viewState.Notice = "Request succeeded"
		if msg.result.Response != nil && msg.result.Response.TransportError != "" {
			m.viewState.Notice = "Request failed"
		}
		m.viewState.FocusedPane = model.FocusedPaneResponse
		m.viewState.ExpandedRightPane = model.FocusedPaneResponse
		m.panes.activeResponseSection = responseui.SectionLive
		m.viewState.ResponseScrollOffset = 0
		return m, nil
	default:
		return m, nil
	}
}

// View renders the current TUI shell.
func (m *Model) View() string {
	m.ensureWidgetDefaults()
	return m.render()
}

// ensureWidgetDefaults keeps root-owned widgets aligned with the current shell defaults.
func (m *Model) ensureWidgetDefaults() {
	m.widgets.filterInput.SetPlaceholder("Filter operations")
	m.widgets.requestFieldInput.SetPlaceholder("Enter value")
	m.widgets.requestBodyInput.SetPlaceholder("Enter raw request body")
}

// startLoadCmd starts a new asynchronous spec load request.
func (m *Model) startLoadCmd() tea.Cmd {
	requestID := m.viewState.ActiveLoadRequestID + 1
	m.session.ActiveLoadRequestID = requestID
	m.viewState.ActiveLoadRequestID = requestID
	m.viewState.LoadInFlight = true
	m.viewState.Notice = "Loading spec"
	m.shell.loadErr = nil

	service := m.service
	source := m.shell.source

	return func() tea.Msg {
		result, err := service.LoadSource(context.Background(), source)
		return specLoadedMsg{
			requestID: requestID,
			result:    result,
			err:       err,
		}
	}
}
