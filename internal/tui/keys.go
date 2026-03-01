package tui

import "github.com/charmbracelet/bubbles/key"

// GlobalKeyMap defines global keybindings.
type GlobalKeyMap struct {
	Tab1    key.Binding
	Tab2    key.Binding
	Tab3    key.Binding
	Tab4    key.Binding
	Tab5    key.Binding
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
		key.WithHelp("4", "ファイル"),
	),
	Tab5: key.NewBinding(
		key.WithKeys("5"),
		key.WithHelp("5", "統計"),
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
	FilterDEBUG      key.Binding
	FilterERROR      key.Binding
	FilterWARN       key.Binding
	FilterMCP        key.Binding
	FilterSTARTUP    key.Binding
	FilterMETA       key.Binding
	FilterATTACHMENT key.Binding
	Search           key.Binding
	AutoScroll       key.Binding
	Escape           key.Binding
}

var LogKeys = LogKeyMap{
	FilterDEBUG: key.NewBinding(
		key.WithKeys("D"),
		key.WithHelp("D", "DEBUG"),
	),
	FilterERROR: key.NewBinding(
		key.WithKeys("E"),
		key.WithHelp("E", "ERROR"),
	),
	FilterWARN: key.NewBinding(
		key.WithKeys("W"),
		key.WithHelp("W", "WARN"),
	),
	FilterMCP: key.NewBinding(
		key.WithKeys("M"),
		key.WithHelp("M", "MCP"),
	),
	FilterSTARTUP: key.NewBinding(
		key.WithKeys("S"),
		key.WithHelp("S", "STARTUP"),
	),
	// T is used because M is taken by MCP
	FilterMETA: key.NewBinding(
		key.WithKeys("T"),
		key.WithHelp("T", "META"),
	),
	FilterATTACHMENT: key.NewBinding(
		key.WithKeys("A"),
		key.WithHelp("A", "ATTACHMENT"),
	),
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "検索"),
	),
	AutoScroll: key.NewBinding(
		key.WithKeys("f"),
		key.WithHelp("f", "自動スクロール"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "検索終了"),
	),
}

// FileKeyMap defines keybindings for the Files tab.
type FileKeyMap struct {
	Enter  key.Binding
	Escape key.Binding
}

var FileKeys = FileKeyMap{
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "展開/表示"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "一覧に戻る"),
	),
}

// ConversationKeyMap defines keybindings for the conversation split panel.
type ConversationKeyMap struct {
	SwitchPane key.Binding
	Escape     key.Binding
}

var ConversationKeys = ConversationKeyMap{
	SwitchPane: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "パネル切替"),
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
