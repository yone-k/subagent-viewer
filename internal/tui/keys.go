package tui

import "github.com/charmbracelet/bubbles/key"

// GlobalKeyMap defines global keybindings.
type GlobalKeyMap struct {
	Tab1    key.Binding
	Tab2    key.Binding
	Tab3    key.Binding
	Tab4    key.Binding
	NextTab key.Binding
	PrevTab key.Binding
	Quit    key.Binding
}

var GlobalKeys = GlobalKeyMap{
	Tab1: key.NewBinding(
		key.WithKeys("1"),
		key.WithHelp("1", "タスク"),
	),
	Tab2: key.NewBinding(
		key.WithKeys("2"),
		key.WithHelp("2", "エージェント"),
	),
	Tab3: key.NewBinding(
		key.WithKeys("3"),
		key.WithHelp("3", "ログ"),
	),
	Tab4: key.NewBinding(
		key.WithKeys("4"),
		key.WithHelp("4", "統計"),
	),
	NextTab: key.NewBinding(
		key.WithKeys("tab", "right"),
		key.WithHelp("tab/→", "次のタブ"),
	),
	PrevTab: key.NewBinding(
		key.WithKeys("shift+tab", "left"),
		key.WithHelp("shift+tab/←", "前のタブ"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q/ctrl+c", "終了"),
	),
}

// LogKeyMap defines keybindings for the Logs tab.
type LogKeyMap struct {
	FilterLeft   key.Binding
	FilterRight  key.Binding
	FilterToggle key.Binding
	Search       key.Binding
	Escape       key.Binding
}

var LogKeys = LogKeyMap{
	FilterLeft: key.NewBinding(
		key.WithKeys("shift+left"),
		key.WithHelp("shift+←", "フィルタ左"),
	),
	FilterRight: key.NewBinding(
		key.WithKeys("shift+right"),
		key.WithHelp("shift+→", "フィルタ右"),
	),
	FilterToggle: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "フィルタ切替"),
	),
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "検索"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "検索終了"),
	),
}

// ConversationKeyMap defines keybindings for the conversation view.
type ConversationKeyMap struct {
	FilterLeft   key.Binding
	FilterRight  key.Binding
	FilterToggle key.Binding
	Escape       key.Binding
}

var ConversationKeys = ConversationKeyMap{
	FilterLeft: key.NewBinding(
		key.WithKeys("shift+left"),
		key.WithHelp("shift+←", "フィルタ左"),
	),
	FilterRight: key.NewBinding(
		key.WithKeys("shift+right"),
		key.WithHelp("shift+→", "フィルタ右"),
	),
	FilterToggle: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "フィルタ切替"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "戻る"),
	),
}

// AgentKeyMap defines keybindings for the Agents tab.
type AgentKeyMap struct {
	Escape key.Binding
}

var AgentKeys = AgentKeyMap{
	Escape: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "戻る"),
	),
}
