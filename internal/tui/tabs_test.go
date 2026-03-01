package tui

import (
	"strings"
	"testing"
)

func TestTabsModel_SwitchTab(t *testing.T) {
	m := NewTabsModel()
	// Default should be first tab (0)
	if m.Active != 0 {
		t.Errorf("initial Active = %d, want 0", m.Active)
	}

	// Switch to tab 2
	m.SetActive(2)
	if m.Active != 2 {
		t.Errorf("Active = %d, want 2", m.Active)
	}

	// Switch to tab 3
	m.SetActive(3)
	if m.Active != 3 {
		t.Errorf("Active = %d, want 3", m.Active)
	}
}

func TestTabsModel_View(t *testing.T) {
	m := NewTabsModel()
	m.SetActive(0)
	view := m.View()

	// Active tab should be present
	if !strings.Contains(view, "Tasks") {
		t.Error("View should contain 'Tasks'")
	}
	if !strings.Contains(view, "Agents") {
		t.Error("View should contain 'Agents'")
	}
	if !strings.Contains(view, "Logs") {
		t.Error("View should contain 'Logs'")
	}
	if !strings.Contains(view, "Stats") {
		t.Error("View should contain 'Stats'")
	}
}

func TestTabsModel_BadgeCounts(t *testing.T) {
	m := NewTabsModel()
	m.SetBadge(0, 5) // 5 tasks
	view := m.View()
	if !strings.Contains(view, "5") {
		t.Error("View should contain badge count '5'")
	}
}

func TestTabsModel_SetActiveBoundary(t *testing.T) {
	m := NewTabsModel()
	m.SetActive(1) // set to a known valid value first
	if m.Active != 1 {
		t.Fatalf("precondition: Active = %d, want 1", m.Active)
	}

	// SetActive(-1) should not change active
	m.SetActive(-1)
	if m.Active != 1 {
		t.Errorf("SetActive(-1): Active = %d, want 1 (unchanged)", m.Active)
	}

	// SetActive(99) should not change active
	m.SetActive(99)
	if m.Active != 1 {
		t.Errorf("SetActive(99): Active = %d, want 1 (unchanged)", m.Active)
	}
}

func TestTabsModel_PrevTabWrap(t *testing.T) {
	m := NewTabsModel()
	// Active starts at 0
	if m.Active != 0 {
		t.Fatalf("precondition: Active = %d, want 0", m.Active)
	}

	// PrevTab at 0 should wrap to last tab (3)
	m.PrevTab()
	if m.Active != 3 {
		t.Errorf("PrevTab from 0: Active = %d, want 3 (wrap to last)", m.Active)
	}
}
