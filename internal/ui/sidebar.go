package ui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nilesh/docktail/internal/model"
	"github.com/nilesh/docktail/internal/theme"
)

// SidebarModel manages the container list sidebar.
type SidebarModel struct {
	Containers []*model.Container
	Cursor     int
	Focused    bool
	Width      int
	Height     int
}

// SidebarKeyMap holds sidebar-specific key bindings.
type SidebarKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Toggle key.Binding
	Action key.Binding
	All    key.Binding
	Shell  key.Binding
}

// OpenShellMsg requests opening a shell for a container.
type OpenShellMsg struct {
	Container *model.Container
}

// OpenActionMenuMsg requests opening the action menu.
type OpenActionMenuMsg struct{}

// RefilterMsg requests a log refilter after visibility change.
type RefilterMsg struct{}

func (m SidebarModel) Update(msg tea.KeyMsg, keys SidebarKeyMap) (SidebarModel, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Up):
		if m.Cursor > 0 {
			m.Cursor--
		}
	case key.Matches(msg, keys.Down):
		if m.Cursor < len(m.Containers)-1 {
			m.Cursor++
		}
	case key.Matches(msg, keys.Toggle):
		c := m.Containers[m.Cursor]
		c.Visible = !c.Visible
		return m, func() tea.Msg { return RefilterMsg{} }
	case key.Matches(msg, keys.Action):
		return m, func() tea.Msg { return OpenActionMenuMsg{} }
	case key.Matches(msg, keys.All):
		allVisible := true
		for _, c := range m.Containers {
			if !c.Visible {
				allVisible = false
				break
			}
		}
		for _, c := range m.Containers {
			c.Visible = !allVisible
		}
		return m, func() tea.Msg { return RefilterMsg{} }
	case key.Matches(msg, keys.Shell):
		c := m.Containers[m.Cursor]
		if c.Status == model.StatusRunning {
			return m, func() tea.Msg {
				return OpenShellMsg{Container: c}
			}
		}
	}
	return m, nil
}

func (m SidebarModel) View() string {
	t := theme.Current
	lines := make([]string, 0, len(m.Containers)+3)

	headerColor := t.Muted
	if m.Focused {
		headerColor = t.Accent
	}
	header := lipgloss.NewStyle().
		Foreground(headerColor).
		Bold(true).
		Width(m.Width).
		Render("CONTAINERS")
	lines = append(lines, header)

	for i, c := range m.Containers {
		focused := m.Focused && m.Cursor == i
		vis := "●"
		visColor := t.GreenColor
		if !c.Visible {
			vis = "○"
			visColor = t.Muted
		}

		statusIcon := "▸"
		switch c.Status {
		case model.StatusPaused:
			statusIcon = "⏸"
		case model.StatusStopped, model.StatusExited:
			statusIcon = "■"
		}

		style := lipgloss.NewStyle().Width(m.Width)
		if focused {
			style = style.Background(lipgloss.Color("#1f2937"))
		}

		line := lipgloss.NewStyle().Foreground(visColor).Render(vis) + " "
		line += lipgloss.NewStyle().Foreground(lipgloss.Color(c.Color)).Bold(true).Render(c.Name) + " "
		line += lipgloss.NewStyle().Foreground(t.Muted).Render(statusIcon)

		lines = append(lines, style.Render(line))
	}

	for len(lines) < m.Height {
		lines = append(lines, lipgloss.NewStyle().Width(m.Width).Render(""))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// HandleClick processes a left-click on the sidebar.
func (m *SidebarModel) HandleClick(contentY int) {
	containerIdx := contentY - 1
	if containerIdx >= 0 && containerIdx < len(m.Containers) {
		m.Cursor = containerIdx
		m.Containers[containerIdx].Visible = !m.Containers[containerIdx].Visible
	}
}

// HandleRightClick processes a right-click, returning whether an action menu should open.
func (m *SidebarModel) HandleRightClick(contentY int) bool {
	containerIdx := contentY - 1
	if containerIdx >= 0 && containerIdx < len(m.Containers) {
		m.Cursor = containerIdx
		return true
	}
	return false
}

// SelectedContainer returns the container at the current cursor position.
func (m SidebarModel) SelectedContainer() *model.Container {
	if m.Cursor >= 0 && m.Cursor < len(m.Containers) {
		return m.Containers[m.Cursor]
	}
	return nil
}

// VisibleCount returns how many containers have logs visible.
func (m SidebarModel) VisibleCount() int {
	count := 0
	for _, c := range m.Containers {
		if c.Visible {
			count++
		}
	}
	return count
}
