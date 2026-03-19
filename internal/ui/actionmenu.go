package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nilesh/docktail/internal/model"
)

// ContainerAction represents an available action for a container.
type ContainerAction struct {
	Key   string
	Label string
}

// ActionMenuModel manages the container action menu overlay.
type ActionMenuModel struct {
	Open    bool
	Actions []ContainerAction
	Cursor  int
}

// ExecuteActionMsg is sent when a user selects an action from the menu.
type ExecuteActionMsg struct {
	Container *model.Container
	Action    string
}

// GetContainerActions returns available actions for a container based on its status.
func GetContainerActions(c *model.Container) []ContainerAction {
	switch c.Status {
	case model.StatusRunning:
		return []ContainerAction{
			{"stop", "Stop"},
			{"restart", "Restart"},
			{"pause", "Pause"},
			{"shell", "Shell"},
		}
	case model.StatusPaused:
		return []ContainerAction{
			{"unpause", "Unpause"},
			{"stop", "Stop"},
		}
	default:
		return []ContainerAction{
			{"start", "Start"},
		}
	}
}

// OpenMenu opens the action menu for a container.
func (m *ActionMenuModel) OpenMenu(c *model.Container) {
	m.Open = true
	m.Cursor = 0
	m.Actions = GetContainerActions(c)
}

// Close closes the action menu.
func (m *ActionMenuModel) Close() {
	m.Open = false
	m.Actions = nil
	m.Cursor = 0
}

func (m ActionMenuModel) Update(msg tea.KeyMsg, container *model.Container) (ActionMenuModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.Open = false
	case "up", "k":
		if m.Cursor > 0 {
			m.Cursor--
		}
	case "down", "j":
		if m.Cursor < len(m.Actions)-1 {
			m.Cursor++
		}
	case "enter":
		if m.Cursor < len(m.Actions) {
			action := m.Actions[m.Cursor]
			m.Open = false
			return m, func() tea.Msg {
				return ExecuteActionMsg{
					Container: container,
					Action:    action.Key,
				}
			}
		}
	}
	return m, nil
}

// View renders the action menu (simple text for now; overlay rendering is Phase 6).
func (m ActionMenuModel) View(width int) string {
	if !m.Open || len(m.Actions) == 0 {
		return ""
	}

	var lines []string
	for i, a := range m.Actions {
		prefix := "  "
		if i == m.Cursor {
			prefix = "> "
		}
		lines = append(lines, prefix+a.Label)
	}
	return strings.Join(lines, "\n")
}
