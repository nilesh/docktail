package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nilesh/docktail/internal/model"
	"github.com/nilesh/docktail/internal/theme"
)

// SidebarModel manages the container list sidebar.
type SidebarModel struct {
	Containers     []*model.Container
	Cursor         int
	Focused        bool
	Width          int
	Height         int
	ShellContainer *model.Container  // currently open shell container
	HideStopped    bool              // hide stopped/exited containers
	ActionMenu     *ActionMenuModel  // action menu to render inline (set by app)
}

// SidebarKeyMap holds sidebar-specific key bindings.
type SidebarKeyMap struct {
	Up          key.Binding
	Down        key.Binding
	Toggle      key.Binding
	Action      key.Binding
	All         key.Binding
	Shell       key.Binding
	HideStopped key.Binding
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
	visible := m.VisibleContainers()
	switch {
	case key.Matches(msg, keys.Up):
		if m.Cursor > 0 {
			m.Cursor--
		}
	case key.Matches(msg, keys.Down):
		if m.Cursor < len(visible)-1 {
			m.Cursor++
		}
	case key.Matches(msg, keys.Toggle):
		if m.Cursor < len(visible) {
			c := visible[m.Cursor]
			c.Visible = !c.Visible
			return m, func() tea.Msg { return RefilterMsg{} }
		}
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
		vc := m.VisibleContainers()
		if m.Cursor >= len(vc) {
			break
		}
		c := vc[m.Cursor]
		if c.Status == model.StatusRunning {
			return m, func() tea.Msg {
				return OpenShellMsg{Container: c}
			}
		}
	case key.Matches(msg, keys.HideStopped):
		m.HideStopped = !m.HideStopped
		// Clamp cursor
		visible := m.VisibleContainers()
		if m.Cursor >= len(visible) {
			m.Cursor = len(visible) - 1
		}
		if m.Cursor < 0 {
			m.Cursor = 0
		}
	}
	return m, nil
}

func (m SidebarModel) View() string {
	t := theme.Current
	lines := make([]string, 0, len(m.Containers)+10)

	// Header
	headerColor := t.Muted
	if m.Focused {
		headerColor = t.Accent
	}
	headerText := "Containers"
	if m.Focused {
		headerText += " ▸"
	}
	header := lipgloss.NewStyle().
		Foreground(headerColor).
		Background(t.Background).
		Bold(true).
		Width(m.Width).
		Render(headerText)
	lines = append(lines, header)

	// Container list
	visible := m.VisibleContainers()
	for i, c := range visible {
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

		// Truncate name to fit
		maxNameLen := m.Width - 8
		if maxNameLen < 4 {
			maxNameLen = 4
		}
		displayName := c.Name
		if len(displayName) > maxNameLen {
			displayName = displayName[:maxNameLen-1] + "…"
		}

		// Left border indicator for focused item
		border := " "
		if focused {
			border = lipgloss.NewStyle().Foreground(t.Accent).Render("│")
		}

		style := lipgloss.NewStyle().Width(m.Width - 1).Background(t.Background) // -1 for border
		if focused {
			style = style.Background(t.FocusBg)
		}

		line := lipgloss.NewStyle().Foreground(visColor).Render(vis) + " "
		line += lipgloss.NewStyle().Foreground(lipgloss.Color(c.Color)).Bold(true).Render(displayName) + " "
		line += lipgloss.NewStyle().Foreground(t.Muted).Render(statusIcon)

		// Shell indicator
		if m.ShellContainer != nil && m.ShellContainer.ID == c.ID {
			line += " " + lipgloss.NewStyle().Foreground(t.Accent).Render(">_")
		}

		lines = append(lines, border+style.Render(line))

		// Render action menu inline after the focused container
		if focused && m.ActionMenu != nil && m.ActionMenu.Open {
			menuLines := m.ActionMenu.InlineView(m.Width, t)
			lines = append(lines, menuLines...)
		}
	}

	// Hidden containers note
	bgStyle := lipgloss.NewStyle().Width(m.Width).Background(t.Background)
	hiddenCount := m.HiddenCount()
	if hiddenCount > 0 {
		note := fmt.Sprintf("(%d hidden)", hiddenCount)
		lines = append(lines, lipgloss.NewStyle().Foreground(t.OrangeColor).Background(t.Background).Width(m.Width).Render(note))
	}

	// Fill middle space
	hintLines := 6
	contentLines := len(lines)
	filler := m.Height - contentLines - hintLines
	for i := 0; i < filler; i++ {
		lines = append(lines, bgStyle.Render(""))
	}

	// Keyboard hints at bottom
	hintStyle := lipgloss.NewStyle().Foreground(t.Muted).Background(t.Background).Width(m.Width)
	hideLabel := "h hide stopped"
	if m.HideStopped {
		hideLabel = "h show stopped"
	}
	lines = append(lines,
		hintStyle.Render("⇥ Tab focus"),
		hintStyle.Render("⎵ toggle log"),
		hintStyle.Render("↵ actions"),
		hintStyle.Render("s shell"),
		hintStyle.Render("a select all"),
		hintStyle.Render(hideLabel),
	)

	// Ensure we fill exactly to Height
	for len(lines) < m.Height {
		lines = append(lines, bgStyle.Render(""))
	}
	if len(lines) > m.Height {
		lines = lines[:m.Height]
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
	visible := m.VisibleContainers()
	if m.Cursor >= 0 && m.Cursor < len(visible) {
		return visible[m.Cursor]
	}
	return nil
}

// VisibleContainers returns containers filtered by HideStopped.
func (m SidebarModel) VisibleContainers() []*model.Container {
	if !m.HideStopped {
		return m.Containers
	}
	var result []*model.Container
	for _, c := range m.Containers {
		if c.Status != model.StatusStopped && c.Status != model.StatusExited {
			result = append(result, c)
		}
	}
	return result
}

// HiddenCount returns the number of hidden stopped containers.
func (m SidebarModel) HiddenCount() int {
	if !m.HideStopped {
		return 0
	}
	count := 0
	for _, c := range m.Containers {
		if c.Status == model.StatusStopped || c.Status == model.StatusExited {
			count++
		}
	}
	return count
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
