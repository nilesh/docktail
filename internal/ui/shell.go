package ui

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nilesh/docktail/internal/model"
	"github.com/nilesh/docktail/internal/theme"
)

// ansiRegex matches ANSI escape sequences (CSI, OSC, and single-char escapes).
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\][^\x07]*\x07|\x1b[()][0-9A-B]|\x1b\[[\?]?[0-9;]*[hlJK]`)

// ExecSession is the interface the shell model uses to communicate with a
// Docker exec session. This keeps the ui package decoupled from the docker
// package.
type ExecSession interface {
	Write(data []byte) (int, error)
	Reader() io.Reader
	Close() error
}

// ShellOutputMsg carries output read from the exec session back into the
// Bubbletea update loop.
type ShellOutputMsg struct {
	Output string
	Err    error
}

// ShellModel manages the shell panel.
type ShellModel struct {
	Container *model.Container
	Lines     []string
	Height    int
	Focused   bool

	exec       ExecSession
	scrollBack int // how many lines scrolled back from bottom (0 = at bottom)
}

// ShellFocusLogs is sent when the user presses Esc in the shell.
type ShellFocusLogs struct{}

func NewShellModel() ShellModel {
	return ShellModel{
		Height: 10,
	}
}

// SetExec attaches an exec session to the shell model.
func (m *ShellModel) SetExec(s ExecSession) {
	m.exec = s
}


// ReadExecOutput returns a tea.Cmd that reads from the exec session and sends
// ShellOutputMsg messages. It should be started once when the exec session is
// attached.
func (m *ShellModel) ReadExecOutput() tea.Cmd {
	if m.exec == nil {
		return nil
	}
	reader := m.exec.Reader()
	return func() tea.Msg {
		buf := make([]byte, 4096)
		n, err := reader.Read(buf)
		if err != nil {
			return ShellOutputMsg{Err: err}
		}
		return ShellOutputMsg{Output: string(buf[:n])}
	}
}

// HandleOutput appends output from the exec session to the shell lines.
func (m *ShellModel) HandleOutput(output string) {
	// Snap to bottom on new output
	m.scrollBack = 0
	// Detect clear screen sequences and reset the buffer
	if strings.Contains(output, "\033[2J") || strings.Contains(output, "\033[H\033[2J") {
		m.Lines = nil
	}

	// Strip ANSI escape sequences
	clean := ansiRegex.ReplaceAllString(output, "")

	// Handle carriage returns (overwrite current line)
	clean = strings.ReplaceAll(clean, "\r\n", "\n")
	clean = strings.ReplaceAll(clean, "\r", "\n")

	parts := strings.Split(clean, "\n")

	for i, part := range parts {
		if i == 0 && len(m.Lines) > 0 {
			m.Lines[len(m.Lines)-1] += part
		} else {
			m.Lines = append(m.Lines, part)
		}
	}

	// Cap buffer
	if len(m.Lines) > 5000 {
		m.Lines = m.Lines[len(m.Lines)-5000:]
	}
}

// Handled returns true if the shell consumed the key. When false, the caller
// should process the key itself (e.g. tab to cycle focus, x to close shell).
func (m ShellModel) Handled(msg tea.KeyMsg) bool {
	if msg.String() == "esc" {
		return true
	}
	if m.exec == nil {
		// Dead session — let the app handle all keys
		return false
	}
	return true
}

func (m ShellModel) Update(msg tea.KeyMsg) (ShellModel, tea.Cmd) {
	// Esc always returns focus to logs
	if msg.String() == "esc" {
		return m, func() tea.Msg { return ShellFocusLogs{} }
	}

	// Scroll: shift+up / shift+down / pgup / pgdown
	switch {
	case msg.Type == tea.KeyPgUp:
		m.scrollBack += m.Height
		maxBack := len(m.Lines) - m.Height
		if maxBack < 0 {
			maxBack = 0
		}
		if m.scrollBack > maxBack {
			m.scrollBack = maxBack
		}
		return m, nil
	case msg.Type == tea.KeyPgDown:
		m.scrollBack -= m.Height
		if m.scrollBack < 0 {
			m.scrollBack = 0
		}
		return m, nil
	case msg.Alt && msg.Type == tea.KeyUp:
		m.scrollBack++
		maxBack := len(m.Lines) - m.Height
		if maxBack < 0 {
			maxBack = 0
		}
		if m.scrollBack > maxBack {
			m.scrollBack = maxBack
		}
		return m, nil
	case msg.Alt && msg.Type == tea.KeyDown:
		m.scrollBack--
		if m.scrollBack < 0 {
			m.scrollBack = 0
		}
		return m, nil
	}

	// If no exec session, don't send keystrokes
	if m.exec == nil {
		return m, nil
	}

	// New output arrived — snap back to bottom
	m.scrollBack = 0

	// Raw/PTY mode: send each keypress as bytes to the exec session
	var data []byte

	switch msg.Type {
	case tea.KeyEnter:
		data = []byte{'\r'}
	case tea.KeyCtrlC:
		data = []byte{0x03}
	case tea.KeyCtrlD:
		data = []byte{0x04}
	case tea.KeyCtrlZ:
		data = []byte{0x1a}
	case tea.KeyCtrlL:
		data = []byte{0x0c}
	case tea.KeyCtrlA:
		data = []byte{0x01}
	case tea.KeyCtrlE:
		data = []byte{0x05}
	case tea.KeyCtrlU:
		data = []byte{0x15}
	case tea.KeyCtrlK:
		data = []byte{0x0b}
	case tea.KeyCtrlW:
		data = []byte{0x17}
	case tea.KeyBackspace:
		data = []byte{0x7f}
	case tea.KeyDelete:
		data = []byte{0x1b, '[', '3', '~'}
	case tea.KeyTab:
		data = []byte{'\t'}
	case tea.KeyUp:
		data = []byte{0x1b, '[', 'A'}
	case tea.KeyDown:
		data = []byte{0x1b, '[', 'B'}
	case tea.KeyRight:
		data = []byte{0x1b, '[', 'C'}
	case tea.KeyLeft:
		data = []byte{0x1b, '[', 'D'}
	case tea.KeyHome:
		data = []byte{0x1b, '[', 'H'}
	case tea.KeyEnd:
		data = []byte{0x1b, '[', 'F'}
	case tea.KeyRunes:
		data = []byte(string(msg.Runes))
	case tea.KeySpace:
		data = []byte{' '}
	default:
		// For any other key with a string representation, send it
		s := msg.String()
		if len(s) == 1 {
			data = []byte(s)
		}
	}

	if len(data) > 0 {
		// Write is non-blocking for a PTY connection, safe in Update
		_, _ = m.exec.Write(data)
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
	closeBtn := lipgloss.NewStyle().Foreground(t.Muted).Render("\u2715")
	pad := shellWidth - lipgloss.Width(tabContent) - lipgloss.Width(closeBtn)
	if pad < 0 {
		pad = 0
	}
	tabBar := lipgloss.NewStyle().
		Width(shellWidth).
		Background(t.ChromeBg).
		Render(tabContent + strings.Repeat(" ", pad) + closeBtn)

	// Shell content — show last N lines (offset by scrollBack), truncated to shell width
	var shellLines []string
	visibleLines := m.Height
	end := len(m.Lines) - m.scrollBack
	if end < 0 {
		end = 0
	}
	start := end - visibleLines
	if start < 0 {
		start = 0
	}
	leftPad := 2
	contentWidth := shellWidth - leftPad
	shellBg := lipgloss.NewStyle().Width(shellWidth).MaxWidth(shellWidth).Background(t.Background)
	padStr := strings.Repeat(" ", leftPad)
	for i := start; i < end; i++ {
		// Expand tabs and truncate to content area to prevent overflow
		line := expandTabs(m.Lines[i], 8)
		runes := []rune(line)
		if len(runes) > contentWidth {
			runes = runes[:contentWidth]
		}
		text := padStr + string(runes)
		// Show cursor on the last line when focused and at bottom
		if m.Focused && m.exec != nil && m.scrollBack == 0 && i == len(m.Lines)-1 {
			text += lipgloss.NewStyle().Foreground(t.Accent).Render("\u258c")
		}
		shellLines = append(shellLines, shellBg.Foreground(t.Foreground).Render(text))
	}

	// Cursor indicator when focused and no output yet
	if m.Focused && len(shellLines) == 0 {
		shellLines = append(shellLines, shellBg.Foreground(t.Accent).Render(padStr+"\u258c"))
	}

	for len(shellLines) < m.Height {
		shellLines = append(shellLines, shellBg.Render(""))
	}

	// Clip to exact height to prevent overflow
	joined := lipgloss.JoinVertical(lipgloss.Left, shellLines...)
	visualRows := strings.Split(joined, "\n")
	if len(visualRows) > m.Height {
		visualRows = visualRows[:m.Height]
	}
	for len(visualRows) < m.Height {
		visualRows = append(visualRows, shellBg.Render(""))
	}

	content := strings.Join(visualRows, "\n")
	return lipgloss.JoinVertical(lipgloss.Left, tabBar, content)
}

// Open opens a shell for the given container.
func (m *ShellModel) Open(c *model.Container) {
	m.Container = c
	m.Lines = []string{fmt.Sprintf("Connecting to %s...", c.Name)}
	m.exec = nil
}

// Close closes the shell panel.
func (m *ShellModel) Close() {
	if m.exec != nil {
		_ = m.exec.Close()
		m.exec = nil
	}
	m.Container = nil
	m.Lines = nil
}

// IsOpen returns whether the shell is open.
func (m ShellModel) IsOpen() bool {
	return m.Container != nil
}

// expandTabs replaces tab characters with spaces aligned to tabstop boundaries.
func expandTabs(s string, tabWidth int) string {
	var b strings.Builder
	col := 0
	for _, r := range s {
		if r == '\t' {
			spaces := tabWidth - (col % tabWidth)
			for j := 0; j < spaces; j++ {
				b.WriteByte(' ')
			}
			col += spaces
		} else {
			b.WriteRune(r)
			col++
		}
	}
	return b.String()
}
