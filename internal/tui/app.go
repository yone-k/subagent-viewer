package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/yone/subagent-viewer/internal/claude"
	"github.com/yone/subagent-viewer/internal/watcher"
)

// AppState represents the application state.
type AppState int

const (
	StateSelector AppState = iota
	StateViewer
)

// headerAndTabsHeight is the vertical space consumed by header and tab bar.
const headerAndTabsHeight = 4

// startWatchersMsg is sent to trigger watcher initialization.
type startWatchersMsg struct{}

// AppModel is the root model for the TUI application.
type AppModel struct {
	state         AppState
	width         int
	height        int
	session       claude.SessionInfo
	sessionActive bool
	lastError     string

	// Sub-models
	selector  SelectorModel
	tabs      TabsModel
	taskView  TaskViewModel
	agentView AgentViewModel
	logView   LogViewModel
	fileView  FileViewModel
	statsView StatsViewModel

	// Watcher lifecycle
	program    *tea.Program
	cancelFunc context.CancelFunc
}

// NewAppModel creates a new AppModel.
// statsView is zero-valued; initialized when session is selected.
func NewAppModel(sessions []claude.SessionInfo) AppModel {
	return AppModel{
		state:     StateSelector,
		selector:  NewSelectorModel(sessions),
		tabs:      NewTabsModel(),
		taskView:  NewTaskViewModel(),
		agentView: NewAgentViewModel(),
		logView:   NewLogViewModel(),
		fileView:  NewFileViewModel(),
	}
}

// NewAppModelWithSession creates an AppModel that starts directly in viewer mode.
// selector is zero-valued; not needed in direct viewer mode.
func NewAppModelWithSession(session claude.SessionInfo) AppModel {
	return AppModel{
		state:     StateViewer,
		session:   session,
		tabs:      NewTabsModel(),
		taskView:  NewTaskViewModel(),
		agentView: NewAgentViewModel(),
		logView:   NewLogViewModel(),
		fileView:  NewFileViewModel(),
		statsView: NewStatsViewModel(session.SessionID),
	}
}

// SetProgram sets the tea.Program reference for watcher communication.
func (m *AppModel) SetProgram(p *tea.Program) {
	m.program = p
}

// Init initializes the model.
func (m *AppModel) Init() tea.Cmd {
	if m.state == StateViewer {
		return m.startWatchersCmd()
	}
	return nil
}

// Update handles messages.
func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.selector.SetSize(msg.Width, msg.Height)
		contentHeight := msg.Height - headerAndTabsHeight
		m.taskView.SetSize(msg.Width, contentHeight)
		m.agentView.SetSize(msg.Width, contentHeight)
		m.logView.SetSize(msg.Width, contentHeight)
		m.fileView.SetSize(msg.Width, contentHeight)
		m.statsView.SetSize(msg.Width, contentHeight)
		return m, nil

	case SessionSelectedMsg:
		m.state = StateViewer
		m.session = msg.Session
		m.statsView = NewStatsViewModel(msg.Session.SessionID)
		m.statsView.SetSize(m.width, m.height-headerAndTabsHeight)
		// Check if session is active
		lockPath := filepath.Join(claude.TasksDir(msg.Session.SessionID), ".lock")
		if _, err := os.Stat(lockPath); err == nil {
			m.sessionActive = true
		}
		// Send stats if available
		var cmds []tea.Cmd
		cmds = append(cmds, m.startWatchersCmd())
		if msg.Session.Stats != nil {
			cmds = append(cmds, func() tea.Msg {
				return StatsUpdatedMsg{Stats: msg.Session.Stats}
			})
		}
		return m, tea.Batch(cmds...)

	case startWatchersMsg:
		if m.program != nil {
			m.StartWatchers(m.program)
		}
		return m, nil

	case watcher.TasksUpdatedMsg:
		newModel, cmd := m.taskView.Update(msg)
		m.taskView = newModel.(TaskViewModel)
		m.tabs.SetBadge(0, len(msg.Tasks))
		return m, cmd

	case watcher.TaskChangedMsg:
		newModel, cmd := m.taskView.Update(msg)
		m.taskView = newModel.(TaskViewModel)
		m.tabs.SetBadge(0, len(m.taskView.tasks))
		return m, cmd

	case watcher.LogEntriesMsg:
		newModel, cmd := m.logView.Update(msg)
		m.logView = newModel.(LogViewModel)
		return m, cmd

	case watcher.FileHistoryUpdatedMsg:
		newModel, cmd := m.fileView.Update(msg)
		m.fileView = newModel.(FileViewModel)
		return m, cmd

	case watcher.SubagentsDiscoveredMsg:
		newModel, cmd := m.agentView.Update(msg)
		m.agentView = newModel.(AgentViewModel)
		m.tabs.SetBadge(1, len(msg.Agents))
		return m, cmd

	case watcher.ConversationUpdatedMsg:
		newModel, cmd := m.agentView.Update(msg)
		m.agentView = newModel.(AgentViewModel)
		return m, cmd

	case StatsUpdatedMsg:
		newModel, cmd := m.statsView.Update(msg)
		m.statsView = newModel.(StatsViewModel)
		return m, cmd

	case watcher.WatcherErrorMsg:
		m.lastError = fmt.Sprintf("%s: %v", msg.Source, msg.Err)
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	}

	// Delegate to current view
	if m.state == StateSelector {
		newModel, cmd := m.selector.Update(msg)
		m.selector = newModel.(SelectorModel)
		return m, cmd
	}

	return m, nil
}

func (m *AppModel) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// In selector state, delegate everything to selector
	if m.state == StateSelector {
		newModel, cmd := m.selector.Update(msg)
		m.selector = newModel.(SelectorModel)
		return m, cmd
	}

	// In viewer state, handle global keys first
	// Check if the log view is in search mode - if so, delegate to log view
	if m.tabs.Active == 2 && m.logView.searching {
		newModel, cmd := m.logView.Update(msg)
		m.logView = newModel.(LogViewModel)
		return m, cmd
	}

	switch {
	case key.Matches(msg, GlobalKeys.Quit):
		if m.cancelFunc != nil {
			m.cancelFunc()
		}
		return m, tea.Quit

	case key.Matches(msg, GlobalKeys.Tab1):
		m.tabs.SetActive(0)
		return m, nil
	case key.Matches(msg, GlobalKeys.Tab2):
		m.tabs.SetActive(1)
		return m, nil
	case key.Matches(msg, GlobalKeys.Tab3):
		m.tabs.SetActive(2)
		return m, nil
	case key.Matches(msg, GlobalKeys.Tab4):
		m.tabs.SetActive(3)
		return m, nil
	case key.Matches(msg, GlobalKeys.Tab5):
		m.tabs.SetActive(4)
		return m, nil
	case key.Matches(msg, GlobalKeys.NextTab):
		// In agent conversation mode, block tab key (used for pane switching)
		if m.tabs.Active == 1 && m.agentView.Mode() == AgentViewModeConversation {
			break // fall through to tab delegation below
		}
		m.tabs.NextTab()
		return m, nil
	case key.Matches(msg, GlobalKeys.PrevTab):
		if m.tabs.Active == 1 && m.agentView.Mode() == AgentViewModeConversation {
			break
		}
		m.tabs.PrevTab()
		return m, nil
	}

	// Delegate to active tab
	var cmd tea.Cmd
	switch m.tabs.Active {
	case 0:
		newModel, c := m.taskView.Update(msg)
		m.taskView = newModel.(TaskViewModel)
		cmd = c
	case 1:
		newModel, c := m.agentView.Update(msg)
		m.agentView = newModel.(AgentViewModel)
		cmd = c
	case 2:
		newModel, c := m.logView.Update(msg)
		m.logView = newModel.(LogViewModel)
		cmd = c
	case 3:
		newModel, c := m.fileView.Update(msg)
		m.fileView = newModel.(FileViewModel)
		cmd = c
	case 4:
		newModel, c := m.statsView.Update(msg)
		m.statsView = newModel.(StatsViewModel)
		cmd = c
	}

	return m, cmd
}

// View renders the application.
func (m *AppModel) View() string {
	if m.state == StateSelector {
		return m.selector.View()
	}

	var b strings.Builder

	// Header
	header := TitleStyle.Render("subagent-viewer")
	if m.sessionActive {
		header += "  " + ActiveSessionStyle.Render("● セッションアクティブ")
	}
	header += "  " + HelpStyle.Render(m.session.Project)
	b.WriteString(header)
	b.WriteString("\n")

	// Tabs
	b.WriteString(m.tabs.View())
	b.WriteString("\n\n")

	// Active tab content
	switch m.tabs.Active {
	case 0:
		b.WriteString(m.taskView.View())
	case 1:
		b.WriteString(m.agentView.View())
	case 2:
		b.WriteString(m.logView.View())
	case 3:
		b.WriteString(m.fileView.View())
	case 4:
		b.WriteString(m.statsView.View())
	}

	// Footer
	b.WriteString("\n")
	if m.lastError != "" {
		b.WriteString(WarningStyle.Render("⚠ " + m.lastError))
		b.WriteString("\n")
	}
	b.WriteString(HelpStyle.Render(m.footerHelp()))

	return b.String()
}

func (m *AppModel) footerHelp() string {
	base := fmt.Sprintf("Session: %s", m.session.SessionID)

	if m.tabs.Active == 1 {
		switch m.agentView.Mode() {
		case AgentViewModeList:
			return fmt.Sprintf("enter: 会話表示  1-5: タブ切替  q: 終了  %s", base)
		case AgentViewModeConversation:
			return fmt.Sprintf("j/k: 移動  tab: パネル切替  esc: 戻る  q: 終了  %s", base)
		}
	}

	return fmt.Sprintf("1-5/←→: タブ切替  q: 終了  %s", base)
}

func (m *AppModel) startWatchersCmd() tea.Cmd {
	return func() tea.Msg {
		return startWatchersMsg{}
	}
}

// StartWatchers initializes and starts all file watchers.
// This should be called after the tea.Program is created.
func (m *AppModel) StartWatchers(program *tea.Program) {
	if m.cancelFunc != nil {
		m.cancelFunc()
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.cancelFunc = cancel

	sessionID := m.session.SessionID

	// Always start log watcher — it polls and handles missing files gracefully
	logPath := claude.DebugLogPath(sessionID)
	lw := watcher.NewLogWatcher(logPath, program)
	go lw.Start(ctx)

	// Start task watcher if tasks directory exists (fsnotify requires existing dir)
	tasksDir := claude.TasksDir(sessionID)
	if _, err := os.Stat(tasksDir); err == nil {
		tw := watcher.NewTaskWatcher(tasksDir, program)
		go tw.Start(ctx)
	}

	// Start file history watcher if directory exists (fsnotify requires existing dir)
	fhDir := claude.FileHistoryDir(sessionID)
	if _, err := os.Stat(fhDir); err == nil {
		fw := watcher.NewFileWatcher(fhDir, program)
		go fw.Start(ctx)
	}

	// Start conversation watcher for subagent JSONL files
	var subagentsDir string
	if m.session.Project != "" {
		subagentsDir = claude.SubagentsDir(m.session.Project, sessionID)
	}
	cw := watcher.NewConversationWatcher(subagentsDir, sessionID, program, m.session.Project, claude.FindSubagentsDirBySessionID)
	go cw.Start(ctx)
}

// Cleanup stops all watchers.
func (m *AppModel) Cleanup() {
	if m.cancelFunc != nil {
		m.cancelFunc()
	}
}
