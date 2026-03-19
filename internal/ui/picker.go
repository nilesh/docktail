package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nilesh/docktail/internal/theme"
)

// PickerModel is a simple list picker for selecting a project.
type PickerModel struct {
	Title    string
	Items    []string
	Cursor   int
	Selected string
	Quit     bool
}

func NewPickerModel(title string, items []string) PickerModel {
	return PickerModel{
		Title: title,
		Items: items,
	}
}

func (m PickerModel) Init() tea.Cmd {
	return nil
}

func (m PickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.Quit = true
			return m, tea.Quit
		case "up", "k":
			if m.Cursor > 0 {
				m.Cursor--
			}
		case "down", "j":
			if m.Cursor < len(m.Items)-1 {
				m.Cursor++
			}
		case "enter":
			m.Selected = m.Items[m.Cursor]
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m PickerModel) View() string {
	t := theme.Current

	s := lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("◉ docktail") + "\n\n"
	s += lipgloss.NewStyle().Foreground(t.Foreground).Bold(true).Render(m.Title) + "\n\n"

	for i, item := range m.Items {
		cursor := "  "
		style := lipgloss.NewStyle().Foreground(t.Muted)
		if i == m.Cursor {
			cursor = lipgloss.NewStyle().Foreground(t.Accent).Render("> ")
			style = lipgloss.NewStyle().Foreground(t.Foreground).Bold(true)
		}
		s += cursor + style.Render(item) + "\n"
	}

	s += fmt.Sprintf("\n%s to select, %s to quit",
		lipgloss.NewStyle().Foreground(t.Accent).Render("enter"),
		lipgloss.NewStyle().Foreground(t.Accent).Render("q"))

	return s + "\n"
}
