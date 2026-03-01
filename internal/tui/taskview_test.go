package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yone/subagent-viewer/internal/claude"
	"github.com/yone/subagent-viewer/internal/watcher"
)

func TestTaskView_UpdateWithTasks(t *testing.T) {
	m := NewTaskViewModel()
	m.SetSize(80, 24)

	tasks := []claude.Task{
		{ID: "1", Subject: "Task One", Status: "completed"},
		{ID: "2", Subject: "Task Two", Status: "in_progress"},
	}
	newModel, _ := m.Update(watcher.TasksUpdatedMsg{Tasks: tasks})
	m = newModel.(TaskViewModel)

	if len(m.tasks) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(m.tasks))
	}
}

func TestTaskView_UpdateSingleTask(t *testing.T) {
	m := NewTaskViewModel()
	m.SetSize(80, 24)

	// Load initial tasks
	tasks := []claude.Task{
		{ID: "1", Subject: "Task One", Status: "pending"},
		{ID: "2", Subject: "Task Two", Status: "pending"},
	}
	newModel, _ := m.Update(watcher.TasksUpdatedMsg{Tasks: tasks})
	m = newModel.(TaskViewModel)

	// Update single task
	newModel, _ = m.Update(watcher.TaskChangedMsg{Task: claude.Task{
		ID: "1", Subject: "Task One", Status: "completed",
	}})
	m = newModel.(TaskViewModel)

	if m.tasks[0].Status != "completed" {
		t.Errorf("task 1 status = %q, want \"completed\"", m.tasks[0].Status)
	}
}

func TestTaskView_ProgressBar(t *testing.T) {
	m := NewTaskViewModel()
	m.SetSize(80, 24)

	tasks := []claude.Task{
		{ID: "1", Subject: "Done", Status: "completed"},
		{ID: "2", Subject: "WIP", Status: "in_progress"},
		{ID: "3", Subject: "Wait", Status: "pending"},
	}
	newModel, _ := m.Update(watcher.TasksUpdatedMsg{Tasks: tasks})
	m = newModel.(TaskViewModel)

	view := m.View()
	// Should contain progress info like "1/3"
	if !strings.Contains(view, "1/3") {
		t.Errorf("view should contain progress '1/3', got:\n%s", view)
	}
}

func TestTaskView_StatusIcons(t *testing.T) {
	m := NewTaskViewModel()
	m.SetSize(80, 24)

	tasks := []claude.Task{
		{ID: "1", Subject: "Completed", Status: "completed"},
		{ID: "2", Subject: "InProgress", Status: "in_progress"},
		{ID: "3", Subject: "Pending", Status: "pending"},
		{ID: "4", Subject: "Blocked", Status: "pending", BlockedBy: []string{"1"}},
	}
	newModel, _ := m.Update(watcher.TasksUpdatedMsg{Tasks: tasks})
	m = newModel.(TaskViewModel)

	view := m.View()
	if !strings.Contains(view, "✓") {
		t.Error("view should contain completed icon ✓")
	}
	if !strings.Contains(view, "●") {
		t.Error("view should contain in_progress icon ●")
	}
	if !strings.Contains(view, "○") {
		t.Error("view should contain pending icon ○")
	}
	if !strings.Contains(view, "✗") {
		t.Error("view should contain blocked icon ✗")
	}
}

func TestTaskView_BlockedBy(t *testing.T) {
	m := NewTaskViewModel()
	m.SetSize(80, 24)

	tasks := []claude.Task{
		{ID: "1", Subject: "First", Status: "completed"},
		{ID: "2", Subject: "Blocked Task", Status: "pending", BlockedBy: []string{"1", "3"}},
	}
	newModel, _ := m.Update(watcher.TasksUpdatedMsg{Tasks: tasks})
	m = newModel.(TaskViewModel)

	view := m.View()
	if !strings.Contains(view, "blocked by") || !strings.Contains(view, "#1") {
		t.Errorf("view should show blocked by info, got:\n%s", view)
	}
}

func TestTaskView_EmptyState(t *testing.T) {
	m := NewTaskViewModel()
	m.SetSize(80, 24)

	view := m.View()
	if !strings.Contains(view, "タスクなし") {
		t.Errorf("empty state not shown, got:\n%s", view)
	}
}

func TestTaskView_DetailToggle(t *testing.T) {
	m := NewTaskViewModel()
	m.SetSize(80, 24)

	tasks := []claude.Task{
		{ID: "1", Subject: "Task", Description: "Detailed description here", Status: "pending"},
	}
	newModel, _ := m.Update(watcher.TasksUpdatedMsg{Tasks: tasks})
	m = newModel.(TaskViewModel)

	// Initially no detail
	if m.showDetail {
		t.Error("showDetail should be false initially")
	}

	// Toggle detail with Enter (simulated by toggling directly for unit test)
	m.showDetail = true
	view := m.View()
	if !strings.Contains(view, "Detailed description here") {
		t.Errorf("detail view should show description, got:\n%s", view)
	}
}

func TestTaskView_CursorMovement(t *testing.T) {
	m := NewTaskViewModel()
	m.SetSize(80, 24)

	tasks := []claude.Task{
		{ID: "1", Subject: "Task One", Status: "pending"},
		{ID: "2", Subject: "Task Two", Status: "pending"},
		{ID: "3", Subject: "Task Three", Status: "pending"},
	}
	newModel, _ := m.Update(watcher.TasksUpdatedMsg{Tasks: tasks})
	m = newModel.(TaskViewModel)

	// Initially selected=0
	if m.selected != 0 {
		t.Errorf("initial selected = %d, want 0", m.selected)
	}

	// Move down with "j" key
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = newModel.(TaskViewModel)
	if m.selected != 1 {
		t.Errorf("after j, selected = %d, want 1", m.selected)
	}

	// Move down with "down" key
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newModel.(TaskViewModel)
	if m.selected != 2 {
		t.Errorf("after down, selected = %d, want 2", m.selected)
	}

	// At the end, down should not change
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newModel.(TaskViewModel)
	if m.selected != 2 {
		t.Errorf("at end after down, selected = %d, want 2", m.selected)
	}

	// Move up with "k" key
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m = newModel.(TaskViewModel)
	if m.selected != 1 {
		t.Errorf("after k, selected = %d, want 1", m.selected)
	}

	// Move up with "up" key
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = newModel.(TaskViewModel)
	if m.selected != 0 {
		t.Errorf("after up, selected = %d, want 0", m.selected)
	}

	// At the beginning, up should not change
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = newModel.(TaskViewModel)
	if m.selected != 0 {
		t.Errorf("at beginning after up, selected = %d, want 0", m.selected)
	}
}

func TestTaskView_TaskChangedMsg_NewTask(t *testing.T) {
	m := NewTaskViewModel()
	m.SetSize(80, 24)

	tasks := []claude.Task{
		{ID: "1", Subject: "Task One", Status: "pending"},
		{ID: "3", Subject: "Task Three", Status: "pending"},
	}
	newModel, _ := m.Update(watcher.TasksUpdatedMsg{Tasks: tasks})
	m = newModel.(TaskViewModel)

	// Add a new task via TaskChangedMsg with an ID that does not exist
	newModel, _ = m.Update(watcher.TaskChangedMsg{Task: claude.Task{
		ID: "2", Subject: "Task Two", Status: "in_progress",
	}})
	m = newModel.(TaskViewModel)

	if len(m.tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(m.tasks))
	}

	// Verify sort order by ID (numeric)
	expectedIDs := []string{"1", "2", "3"}
	for i, id := range expectedIDs {
		if m.tasks[i].ID != id {
			t.Errorf("tasks[%d].ID = %q, want %q", i, m.tasks[i].ID, id)
		}
	}
}

func TestTaskView_EnterToggle(t *testing.T) {
	m := NewTaskViewModel()
	m.SetSize(80, 24)

	tasks := []claude.Task{
		{ID: "1", Subject: "Task", Description: "Detail text", Status: "pending"},
	}
	newModel, _ := m.Update(watcher.TasksUpdatedMsg{Tasks: tasks})
	m = newModel.(TaskViewModel)

	// Initially showDetail is false
	if m.showDetail {
		t.Error("showDetail should be false initially")
	}

	// Press Enter via Update
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(TaskViewModel)
	if !m.showDetail {
		t.Error("showDetail should be true after Enter")
	}

	// Press Enter again via Update
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(TaskViewModel)
	if m.showDetail {
		t.Error("showDetail should be false after second Enter")
	}
}

func TestTaskView_ScrollFollowsSelection(t *testing.T) {
	m := NewTaskViewModel()
	m.SetSize(80, 7) // height=7: viewHeight = 7-2 = 5 items visible

	// Create 10 tasks
	tasks := make([]claude.Task, 10)
	for i := range tasks {
		tasks[i] = claude.Task{
			ID:      fmt.Sprintf("%d", i+1),
			Subject: fmt.Sprintf("Task %d", i+1),
			Status:  "pending",
		}
	}
	newModel, _ := m.Update(watcher.TasksUpdatedMsg{Tasks: tasks})
	m = newModel.(TaskViewModel)

	if m.scrollOffset != 0 {
		t.Errorf("initial scrollOffset = %d, want 0", m.scrollOffset)
	}

	// Move down to item 5 (index 4) — should still be visible without scrolling
	for i := 0; i < 4; i++ {
		newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		m = newModel.(TaskViewModel)
	}
	if m.selected != 4 {
		t.Errorf("selected = %d, want 4", m.selected)
	}
	if m.scrollOffset != 0 {
		t.Errorf("scrollOffset = %d, want 0 (item 4 fits in viewHeight 5)", m.scrollOffset)
	}

	// Move down to item 6 (index 5) — should start scrolling
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = newModel.(TaskViewModel)
	if m.selected != 5 {
		t.Errorf("selected = %d, want 5", m.selected)
	}
	if m.scrollOffset != 1 {
		t.Errorf("scrollOffset = %d, want 1", m.scrollOffset)
	}

	// Move to last item (index 9)
	for i := 0; i < 4; i++ {
		newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		m = newModel.(TaskViewModel)
	}
	if m.selected != 9 {
		t.Errorf("selected = %d, want 9", m.selected)
	}
	if m.scrollOffset != 5 {
		t.Errorf("scrollOffset = %d, want 5", m.scrollOffset)
	}

	// Move back up — scroll should follow
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m = newModel.(TaskViewModel)
	if m.selected != 8 {
		t.Errorf("selected = %d, want 8", m.selected)
	}

	// Move all the way up
	for i := 0; i < 8; i++ {
		newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
		m = newModel.(TaskViewModel)
	}
	if m.selected != 0 {
		t.Errorf("selected = %d, want 0", m.selected)
	}
	if m.scrollOffset != 0 {
		t.Errorf("scrollOffset = %d, want 0", m.scrollOffset)
	}
}
