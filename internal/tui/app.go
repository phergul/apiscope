package tui

import (
	"context"
	"io"

	"github.com/phergul/apiscope/internal/app"
	"github.com/phergul/apiscope/internal/model"
	"github.com/phergul/apiscope/internal/tui/widgets"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	defaultWidth       = 120
	defaultHeight      = 32
	layoutPresetWide   = "wide"
	layoutPresetNarrow = "narrow"
)

type Program struct {
	program *tea.Program
}

type Model struct {
	service               *app.Service
	session               model.SessionState
	viewState             model.ViewState
	width                 int
	height                int
	source                string
	loadErr               error
	activeDetailsSection  detailsSection
	activeRequestSection  string
	activeResponseSection string
	filterInput           widgets.TextInput
	requestFieldInput     widgets.TextInput
	requestBodyInput      widgets.TextArea
}

func NewProgram(service *app.Service, source string, input io.Reader, output io.Writer) *Program {
	options := []tea.ProgramOption{
		tea.WithInput(input),
		tea.WithOutput(output),
	}

	return &Program{
		program: tea.NewProgram(NewModel(service, source), options...),
	}
}

func (p *Program) Run() error {
	_, err := p.program.Run()
	return err
}

func NewModel(service *app.Service, source string) *Model {
	if service == nil {
		service = app.NewService(nil)
	}

	filterInput := widgets.NewTextInput()
	filterInput.SetPlaceholder("Filter operations")
	requestFieldInput := widgets.NewTextInput()
	requestFieldInput.SetPlaceholder("Enter value")
	requestBodyInput := widgets.NewTextArea()
	requestBodyInput.SetPlaceholder("Enter raw request body")

	return &Model{
		service:               service,
		source:                source,
		activeDetailsSection:  detailsSectionSummary,
		activeRequestSection:  "",
		activeResponseSection: "",
		filterInput:           filterInput,
		requestFieldInput:     requestFieldInput,
		requestBodyInput:      requestBodyInput,
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

func (m *Model) Init() tea.Cmd {
	m.ensureWidgetDefaults()
	return m.startLoadCmd()
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.ensureWidgetDefaults()

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewState.RightPaneLayoutPreset = chooseLayoutPreset(msg.Width)
		return m, nil
	case tea.KeyMsg:
		return m.updateKey(msg)
	case specLoadedMsg:
		if msg.requestID != m.viewState.ActiveLoadRequestID {
			return m, nil
		}

		m.loadErr = msg.err
		if msg.err != nil {
			m.session.ActiveLoadRequestID = msg.requestID
			m.viewState.ActiveLoadRequestID = msg.requestID
			m.viewState.LoadInFlight = false
			m.viewState.Notice = "load failed"
			return m, nil
		}

		m.session = msg.result.Session
		m.viewState = msg.result.View
		m.session.ActiveLoadRequestID = msg.requestID
		m.viewState.ActiveLoadRequestID = msg.requestID
		m.viewState.LoadInFlight = false
		m.viewState.RightPaneLayoutPreset = chooseLayoutPreset(m.width)
		m.syncVisibleOperations()
		m.syncActivePaneSections()
		return m, nil
	default:
		return m, nil
	}
}

func (m *Model) View() string {
	m.ensureWidgetDefaults()
	return m.render()
}

func (m *Model) ensureWidgetDefaults() {
	m.filterInput.SetPlaceholder("Filter operations")
	m.requestFieldInput.SetPlaceholder("Enter value")
	m.requestBodyInput.SetPlaceholder("Enter raw request body")
}

func (m *Model) startLoadCmd() tea.Cmd {
	requestID := m.viewState.ActiveLoadRequestID + 1
	m.session.ActiveLoadRequestID = requestID
	m.viewState.ActiveLoadRequestID = requestID
	m.viewState.LoadInFlight = true
	m.viewState.Notice = "loading spec"
	m.loadErr = nil

	service := m.service
	source := m.source

	return func() tea.Msg {
		result, err := service.LoadSource(context.Background(), source)
		return specLoadedMsg{
			requestID: requestID,
			result:    result,
			err:       err,
		}
	}
}
