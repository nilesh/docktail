package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nilesh/docktail/internal/model"
	"github.com/nilesh/docktail/internal/theme"
)

// ShellModel manages the shell panel.
type ShellModel struct {
	Container *model.Container
	Lines     []string
	Input     string
	CmdHist   []string
	CmdIdx    int
	Height    int
	Focused   bool
}

// ShellFocusLogs is sent when the user presses Esc in the shell.
type ShellFocusLogs struct{}

func NewShellModel() ShellModel {
	return ShellModel{
		Height: 10,
		CmdIdx: -1,
	}
}

func (m ShellModel) Update(msg tea.KeyMsg) (ShellModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m, func() tea.Msg { return ShellFocusLogs{} }
	case "enter":
		if m.Input != "" {
			m.Lines = append(m.Lines, "$ "+m.Input)
			// In real implementation, this sends to docker exec
			m.Lines = append(m.Lines, "(shell output would appear here)")
			if len(m.CmdHist) == 0 || m.CmdHist[0] != m.Input {
				m.CmdHist = append([]string{m.Input}, m.CmdHist...)
			}
			m.Input = ""
			m.CmdIdx = -1
		}
	case "up":
		if len(m.CmdHist) > 0 {
			m.CmdIdx++
			if m.CmdIdx >= len(m.CmdHist) {
				m.CmdIdx = len(m.CmdHist) - 1
			}
			m.Input = m.CmdHist[m.CmdIdx]
		}
	case "down":
		if m.CmdIdx > 0 {
			m.CmdIdx--
			m.Input = m.CmdHist[m.CmdIdx]
		} else {
			m.CmdIdx = -1
			m.Input = ""
		}
	case "backspace":
		if len(m.Input) > 0 {
			m.Input = m.Input[:len(m.Input)-1]
		}
	default:
		if len(msg.String()) == 1 {
			m.Input += msg.String()
		}
	}
	return m, nil
}

func (m ShellModel) View(width int) string {
	t := theme.Current
	shellWidth := width

	// Tab bar
	tabContent := lipgloss.NewStyle().Foreground(t.Accent).Render(">_ ") +
		lipgloss.NewStyle().Foreground(lipgloss.Color(m.Container.Color)).Render(m.Container.Name) +
		lipgloss.NewStyle().Foreground(t.Muted).Render(" shell")
	closeBtn := lipgloss.NewStyle().Foreground(t.Muted).Render("✕")
	pad := shellWidth - lipgloss.Width(tabContent) - lipgloss.Width(closeBtn)
	if pad < 0 {
		pad = 0
	}
	tabBar := lipgloss.NewStyle().
		Width(shellWidth).
		Background(lipgloss.Color("#161b22")).
		Render(tabContent + strings.Repeat(" ", pad) + closeBtn)

	// Shell content
	var shellLines []string
	visibleLines := m.Height - 1
	start := 0
	if len(m.Lines) > visibleLines {
		start = len(m.Lines) - visibleLines
	}
	for i := start; i < len(m.Lines); i++ {
		shellLines = append(shellLines, lipgloss.NewStyle().Width(shellWidth).Foreground(t.Foreground).Render(m.Lines[i]))
	}

	// Input line
	prompt := "$ "
	inputLine := lipgloss.NewStyle().Foreground(t.GreenColor).Render(prompt) +
		lipgloss.NewStyle().Foreground(t.Foreground).Render(m.Input)
	if m.Focused {
		inputLine += lipgloss.NewStyle().Foreground(t.Accent).Render("▌")
	}
	shellLines = append(shellLines, lipgloss.NewStyle().Width(shellWidth).Render(inputLine))

	for len(shellLines) < m.Height {
		shellLines = append(shellLines, lipgloss.NewStyle().Width(shellWidth).Render(""))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, shellLines...)
	return lipgloss.JoinVertical(lipgloss.Left, tabBar, content)
}

// Open opens a shell for the given container.
func (m *ShellModel) Open(c *model.Container) {
	m.Container = c
	m.Lines = []string{fmt.Sprintf("Connecting to %s...", c.Name)}
	m.Input = ""
	m.CmdIdx = -1
}

// Close closes the shell panel.
func (m *ShellModel) Close() {
	m.Container = nil
	m.Lines = nil
	m.Input = ""
}

// IsOpen returns whether the shell is open.
func (m ShellModel) IsOpen() bool {
	return m.Container != nil
}
