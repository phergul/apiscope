package tui

import (
	"context"
	"io"

	"api-tui/internal/app"
	"api-tui/internal/model"

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
	service              *app.Service
	session              model.SessionState
	viewState            model.ViewState
	width                int
	height               int
	source               string
	loadErr              error
	activeDetailsSection detailsSection
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

	return &Model{
		service:              service,
		source:               source,
		activeDetailsSection: detailsSectionSummary,
		viewState: model.ViewState{
			FocusedPane:           model.FocusedPaneOperations,
			ActiveEditorMode:      model.EditorModeBrowse,
			OperationsPaneVisible: true,
			ResponsePaneExpanded:  false,
			RightPaneLayoutPreset: layoutPresetNarrow,
		},
	}
}

func (m *Model) Init() tea.Cmd {
	return m.startLoadCmd()
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		m.syncActiveDetailsSection()
		return m, nil
	default:
		return m, nil
	}
}

func (m *Model) View() string {
	return m.render()
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
