package tui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yone/subagent-viewer/internal/claude"
	"github.com/yone/subagent-viewer/internal/watcher"
)

const maxProgressBarWidth = 40

// TaskViewModel manages the Tasks tab view.
type TaskViewModel struct {
	tasks        []claude.Task
	selected     int
	showDetail   bool
	scrollOffset int
	width        int
	height       int
}

// NewTaskViewModel creates a new TaskViewModel.
func NewTaskViewModel() TaskViewModel {
	return TaskViewModel{}
}

// SetSize uses a pointer receiver because app.go calls it through a pointer to AppModel's field.
func (m *TaskViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Init initializes the model.
func (m TaskViewModel) Init() tea.Cmd {
	return nil
}

// Update handles messages.
func (m TaskViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case watcher.TasksUpdatedMsg:
		m.tasks = msg.Tasks
		if m.selected >= len(m.tasks) {
			m.selected = 0
		}
		m = m.clampScroll()
	case watcher.TaskChangedMsg:
		found := false
		for i, task := range m.tasks {
			if task.ID == msg.Task.ID {
				m.tasks[i] = msg.Task
				found = true
				break
			}
		}
		if !found {
			m.tasks = append(m.tasks, msg.Task)
			sort.Slice(m.tasks, func(i, j int) bool {
				ni, errI := strconv.Atoi(m.tasks[i].ID)
				nj, errJ := strconv.Atoi(m.tasks[j].ID)
				if errI != nil || errJ != nil {
					return m.tasks[i].ID < m.tasks[j].ID
				}
				return ni < nj
			})
		}
		m = m.clampScroll()
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m TaskViewModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.selected > 0 {
			m.selected--
		}
	case "down", "j":
		if m.selected < len(m.tasks)-1 {
			m.selected++
		}
	case "enter":
		m.showDetail = !m.showDetail
	}
	m = m.clampScroll()
	return m, nil
}

func (m TaskViewModel) viewHeight() int {
	h := m.height - 2 // subtract progress bar (2 lines)
	if h < 1 {
		h = 1
	}
	return h
}

func (m TaskViewModel) clampScroll() TaskViewModel {
	viewHeight := m.viewHeight()
	if m.selected < m.scrollOffset {
		m.scrollOffset = m.selected
	}
	if m.selected >= m.scrollOffset+viewHeight {
		m.scrollOffset = m.selected - viewHeight + 1
	}
	maxOffset := max(0, len(m.tasks)-viewHeight)
	if m.scrollOffset > maxOffset {
		m.scrollOffset = maxOffset
	}
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
	return m
}

// View renders the task list.
func (m TaskViewModel) View() string {
	return m.viewTasks()
}

func (m TaskViewModel) viewTasks() string {
	if len(m.tasks) == 0 {
		return EmptyStateStyle.Render("タスクなし")
	}

	var b strings.Builder

	// Progress bar
	completed := 0
	for _, t := range m.tasks {
		if t.Status == "completed" {
			completed++
		}
	}
	total := len(m.tasks)
	b.WriteString(renderProgressBar(completed, total, m.width-4))
	b.WriteString(fmt.Sprintf("  %d/%d\n\n", completed, total))

	// Task list with scroll
	viewHeight := m.viewHeight()
	linesRendered := 0
	for i := m.scrollOffset; i < len(m.tasks) && linesRendered < viewHeight; i++ {
		task := m.tasks[i]
		icon := statusIcon(task)

		// Build detail parts
		var details []string
		if len(task.BlockedBy) > 0 {
			refs := make([]string, len(task.BlockedBy))
			for j, id := range task.BlockedBy {
				refs[j] = "#" + id
			}
			details = append(details, fmt.Sprintf(" (blocked by %s)", strings.Join(refs, ", ")))
		}
		if task.Status == "in_progress" && task.ActiveForm != "" {
			details = append(details, fmt.Sprintf(" — %s", task.ActiveForm))
		}

		iconPrefix := icon + " "
		b.WriteString(renderListItemWithIcon(i == m.selected, iconPrefix, task.Subject, details...) + "\n")
		linesRendered++

		// Show detail for selected task
		if i == m.selected && m.showDetail && task.Description != "" {
			b.WriteString(BorderStyle.Render(task.Description))
			b.WriteString("\n")
			linesRendered++
		}
	}

	return b.String()
}

func statusIcon(task claude.Task) string {
	if len(task.BlockedBy) > 0 && task.Status != "completed" {
		return StatusBlocked.String()
	}
	switch task.Status {
	case "completed":
		return StatusCompleted.String()
	case "in_progress":
		return StatusInProgress.String()
	default:
		return StatusPending.String()
	}
}

func renderProgressBar(completed, total, width int) string {
	if total == 0 || width <= 0 {
		return ""
	}
	barWidth := width
	if barWidth > maxProgressBarWidth {
		barWidth = maxProgressBarWidth
	}
	filled := barWidth * completed / total
	empty := barWidth - filled

	bar := ProgressBarFilled.Render(strings.Repeat("█", filled))
	bar += ProgressBarEmpty.Render(strings.Repeat("░", empty))
	return bar
}
