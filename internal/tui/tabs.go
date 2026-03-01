package tui

import (
	"fmt"
	"strings"
)

// TabsModel manages the tab bar.
type TabsModel struct {
	Active int
	tabs   []string
	badges map[int]int
}

// NewTabsModel creates a new TabsModel with default tabs.
func NewTabsModel() TabsModel {
	return TabsModel{
		Active: 0,
		tabs:   []string{"Tasks", "Agents", "Logs", "Files", "Stats"},
		badges: make(map[int]int),
	}
}

// SetActive sets the active tab index.
func (m *TabsModel) SetActive(index int) {
	if index >= 0 && index < len(m.tabs) {
		m.Active = index
	}
}

// NextTab cycles to the next tab.
func (m *TabsModel) NextTab() {
	m.Active = (m.Active + 1) % len(m.tabs)
}

// PrevTab cycles to the previous tab.
func (m *TabsModel) PrevTab() {
	m.Active = (m.Active - 1 + len(m.tabs)) % len(m.tabs)
}

// SetBadge sets the badge count for a tab.
func (m *TabsModel) SetBadge(index, count int) {
	m.badges[index] = count
}

// View renders the tab bar.
func (m TabsModel) View() string {
	var tabs []string
	for i, tab := range m.tabs {
		label := tab
		if count, ok := m.badges[i]; ok && count > 0 {
			label = fmt.Sprintf("%s (%d)", tab, count)
		}
		if i == m.Active {
			tabs = append(tabs, ActiveTabStyle.Render(label))
		} else {
			tabs = append(tabs, InactiveTabStyle.Render(label))
		}
	}
	return strings.Join(tabs, TabGapStyle.Render("|"))
}
