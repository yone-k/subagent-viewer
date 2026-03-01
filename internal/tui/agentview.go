package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/yone/subagent-viewer/internal/claude"
	"github.com/yone/subagent-viewer/internal/watcher"
)

// AgentViewMode represents the current view mode within the Agents tab.
type AgentViewMode int

const (
	AgentViewModeList         AgentViewMode = iota // Agent list (default)
	AgentViewModeConversation                      // Conversation split view
)

// AgentViewModel manages the Agents tab view.
type AgentViewModel struct {
	agents           []claude.SubagentInfo
	agentSelected    int
	mode             AgentViewMode
	currentAgentID   string
	conversations    map[string][]claude.ConversationEntry
	conversationInfo map[string]*claude.SubagentInfo
	conversationView ConversationViewModel
	width, height    int
}

// NewAgentViewModel creates a new AgentViewModel.
func NewAgentViewModel() AgentViewModel {
	return AgentViewModel{
		conversations:    make(map[string][]claude.ConversationEntry),
		conversationInfo: make(map[string]*claude.SubagentInfo),
		conversationView: NewConversationViewModel(),
	}
}

// SetSize updates the view dimensions.
func (m *AgentViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.conversationView.SetSize(width, height)
}

// Mode returns the current view mode.
func (m AgentViewModel) Mode() AgentViewMode {
	return m.mode
}

// Init initializes the model.
func (m AgentViewModel) Init() tea.Cmd {
	return nil
}

// Update handles messages.
func (m AgentViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case watcher.SubagentsDiscoveredMsg:
		m.agents = msg.Agents
		if len(m.agents) > 0 && m.agentSelected >= len(m.agents) {
			m.agentSelected = len(m.agents) - 1
		}
		if len(m.agents) == 0 {
			m.agentSelected = 0
		}

	case watcher.ConversationUpdatedMsg:
		m.conversations[msg.AgentID] = msg.Entries
		if msg.Info != nil {
			m.conversationInfo[msg.AgentID] = msg.Info
		}
		// Update conversation view if currently viewing this agent
		if m.mode == AgentViewModeConversation && m.currentAgentID == msg.AgentID {
			m.conversationView.UpdateEntries(msg.Entries, msg.Info)
		}

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

// handleKey handles key messages based on current mode.
func (m AgentViewModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case AgentViewModeList:
		switch {
		case key.Matches(msg, AgentKeys.Escape):
			// Do nothing; AppModel handles tab switching
			return m, nil
		}
		switch msg.String() {
		case "up", "k":
			if m.agentSelected > 0 {
				m.agentSelected--
			}
		case "down", "j":
			if m.agentSelected < len(m.agents)-1 {
				m.agentSelected++
			}
		case "enter":
			if len(m.agents) > 0 && m.agentSelected < len(m.agents) {
				agent := m.agents[m.agentSelected]
				m.currentAgentID = agent.AgentID
				entries := m.conversations[agent.AgentID]
				info := m.conversationInfo[agent.AgentID]
				m.conversationView.SetData(agent.AgentID, entries, info)
				m.conversationView.SetSize(m.width, m.height)
				m.mode = AgentViewModeConversation
			}
		}

	case AgentViewModeConversation:
		updated, handled := m.conversationView.Update(msg)
		m.conversationView = updated
		if !handled {
			m.mode = AgentViewModeList
		}
	}

	return m, nil
}

// View renders the agent view.
func (m AgentViewModel) View() string {
	switch m.mode {
	case AgentViewModeConversation:
		return m.conversationView.View()
	default:
		return m.viewAgents()
	}
}

// viewAgents renders the agent list.
func (m AgentViewModel) viewAgents() string {
	if len(m.agents) == 0 {
		return EmptyStateStyle.Render("サブエージェントなし")
	}

	var b strings.Builder
	b.WriteString(TitleStyle.Render("サブエージェント一覧"))
	b.WriteString("\n\n")

	for i, agent := range m.agents {
		label := agent.Description
		if label == "" {
			label = agent.Slug
		}
		if label == "" {
			label = agent.AgentID
		}

		// Status icon (kept separate from label to avoid ANSI nesting issues)
		var statusPrefix string
		switch agent.Status {
		case claude.SubagentRunning:
			statusPrefix = StatusInProgress.String() + " "
		case claude.SubagentClosed:
			statusPrefix = StatusCompleted.String() + " "
		}

		// 4 = indent for prompt line ("    ")
		maxPromptWidth := m.width - 4
		if maxPromptWidth < 10 {
			maxPromptWidth = 10
		}
		prompt := truncateText(agent.Prompt, maxPromptWidth)

		// Build detail parts
		var details []string
		if agent.SubagentType != "" {
			details = append(details, fmt.Sprintf(" [%s]", agent.SubagentType))
		}
		details = append(details, fmt.Sprintf("  (%d entries)", agent.EntryCount))

		selected := i == m.agentSelected
		b.WriteString(renderListItemWithIcon(selected, statusPrefix, label, details...) + "\n")
		if selected {
			b.WriteString(fmt.Sprintf("    %s\n", SelectedDetailStyle.Render(prompt)))
		} else {
			b.WriteString(fmt.Sprintf("    %s\n", DimStyle.Render(prompt)))
		}
		if i < len(m.agents)-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}
