package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nilesh/docktail/internal/model"
	"github.com/nilesh/docktail/internal/theme"
)

// stripANSI removes ANSI escape sequences from a string.
func stripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

// LogViewModel manages the main log viewport.
type LogViewModel struct {
	Logs           []*model.LogEntry
	FilteredLogs   []*model.LogEntry
	Frozen         bool
	CursorLine     int
	SelectedLines  map[int]bool
	SelAnchor      int
	ShowTimestamps bool
	WrapLines      bool
	Width          int
	Height         int
	NameWidth      int // max container name width, set by app
}

// LogViewKeyMap holds log view key bindings.
type LogViewKeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Top      key.Binding
	Bottom   key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Select   key.Binding
	Copy     key.Binding
	ClearSel key.Binding
}

// CopiedMsg indicates lines were copied.
type CopiedMsg struct {
	Text string
}

func NewLogViewModel() LogViewModel {
	return LogViewModel{
		Logs:          make([]*model.LogEntry, 0, 5000),
		SelectedLines: make(map[int]bool),
		SelAnchor:     -1,
	}
}

func (m LogViewModel) Update(msg tea.KeyMsg, keys LogViewKeyMap) (LogViewModel, tea.Cmd) {
	maxIdx := len(m.FilteredLogs) - 1

	switch {
	case msg.Type == tea.KeyShiftUp:
		if m.SelAnchor < 0 {
			m.SelAnchor = m.CursorLine
		}
		if m.CursorLine > 0 {
			m.CursorLine--
		}
		m.selectRange(m.SelAnchor, m.CursorLine)
	case msg.Type == tea.KeyShiftDown:
		if m.SelAnchor < 0 {
			m.SelAnchor = m.CursorLine
		}
		if m.CursorLine < maxIdx {
			m.CursorLine++
		}
		m.selectRange(m.SelAnchor, m.CursorLine)
	case key.Matches(msg, keys.Up):
		if m.CursorLine > 0 {
			m.CursorLine--
		}
	case key.Matches(msg, keys.Down):
		if m.CursorLine < maxIdx {
			m.CursorLine++
		}
	case key.Matches(msg, keys.Top):
		m.CursorLine = 0
	case key.Matches(msg, keys.Bottom):
		m.CursorLine = maxIdx
	case key.Matches(msg, keys.PageUp):
		m.CursorLine -= 20
		if m.CursorLine < 0 {
			m.CursorLine = 0
		}
	case key.Matches(msg, keys.PageDown):
		m.CursorLine += 20
		if m.CursorLine > maxIdx {
			m.CursorLine = maxIdx
		}
	case key.Matches(msg, keys.Select):
		if m.SelectedLines[m.CursorLine] {
			delete(m.SelectedLines, m.CursorLine)
		} else {
			m.SelectedLines[m.CursorLine] = true
		}
		m.SelAnchor = m.CursorLine
	case key.Matches(msg, keys.Copy):
		text := m.buildCopyText()
		if text != "" {
			return m, func() tea.Msg { return CopiedMsg{Text: text} }
		}
	case key.Matches(msg, keys.ClearSel):
		m.SelectedLines = make(map[int]bool)
		m.SelAnchor = -1
	}
	return m, nil
}

// selectRange replaces the selection with all lines between a and b (inclusive).
func (m *LogViewModel) selectRange(a, b int) {
	m.SelectedLines = make(map[int]bool)
	lo, hi := a, b
	if lo > hi {
		lo, hi = hi, lo
	}
	for i := lo; i <= hi; i++ {
		m.SelectedLines[i] = true
	}
}

func (m LogViewModel) buildCopyText() string {
	if len(m.SelectedLines) == 0 {
		return ""
	}

	var lines []string
	for i := 0; i < len(m.FilteredLogs); i++ {
		if !m.SelectedLines[i] {
			continue
		}
		entry := m.FilteredLogs[i]
		var parts []string
		if m.ShowTimestamps {
			parts = append(parts, entry.Timestamp.Format("15:04:05.000"))
		}
		parts = append(parts, fmt.Sprintf("[%s]", entry.Container.Name))
		parts = append(parts, stripANSI(entry.Message))
		lines = append(lines, strings.Join(parts, " "))
	}
	return strings.Join(lines, "\n")
}

func (m LogViewModel) View() string {
	t := theme.Current
	logWidth := m.Width

	var lines []string

	// Show helpful empty state when there are no logs
	if len(m.FilteredLogs) == 0 {
		emptyMsg := "No logs yet. Waiting for container output..."
		if len(m.Logs) > 0 {
			emptyMsg = "No logs match the current filters."
		}
		for i := 0; i < m.Height; i++ {
			if i == m.Height/2 {
				centered := lipgloss.NewStyle().
					Width(logWidth).
					Align(lipgloss.Center).
					Foreground(t.Muted).
					Background(t.Background).
					Render(emptyMsg)
				lines = append(lines, centered)
			} else {
				lines = append(lines, lipgloss.NewStyle().Width(logWidth).Background(t.Background).Render(""))
			}
		}
		return lipgloss.JoinVertical(lipgloss.Left, lines...)
	}

	startIdx := 0
	if !m.Frozen {
		startIdx = len(m.FilteredLogs) - m.Height
	} else if m.CursorLine >= 0 {
		startIdx = m.CursorLine - m.Height/2
	}
	if startIdx < 0 {
		startIdx = 0
	}
	endIdx := startIdx + m.Height
	if endIdx > len(m.FilteredLogs) {
		endIdx = len(m.FilteredLogs)
	}

	nameW := m.NameWidth
	if nameW < 8 {
		nameW = 8
	}

	for i := startIdx; i < endIdx; i++ {
		entry := m.FilteredLogs[i]
		isCursor := m.Frozen && m.CursorLine == i
		isSelected := m.SelectedLines[i]

		// Left border indicator
		border := " "
		if isCursor {
			border = lipgloss.NewStyle().Foreground(t.Accent).Render("│")
		} else if isSelected {
			border = lipgloss.NewStyle().Foreground(t.AccentDim).Render("│")
		}

		// Determine background for this line
		bg := t.Background
		if isSelected {
			bg = t.SelectedBg
		} else if isCursor {
			bg = t.CursorBg
		}

		// Helper to style text with the correct background
		styled := func(fg lipgloss.Color, text string) string {
			return lipgloss.NewStyle().Foreground(fg).Background(bg).Render(text)
		}
		styledBold := func(fg lipgloss.Color, text string) string {
			return lipgloss.NewStyle().Foreground(fg).Background(bg).Bold(true).Render(text)
		}
		gap := lipgloss.NewStyle().Background(bg).Render(" ")

		// Build the line as a plain string, then truncate to width
		var parts []string
		usedWidth := 1 // left border

		if m.Frozen {
			parts = append(parts, styled(t.Border, fmt.Sprintf("%4d", i+1)))
			parts = append(parts, gap)
			usedWidth += 5
		}

		if m.ShowTimestamps {
			ts := entry.Timestamp.Format("15:04:05.000")
			parts = append(parts, styled(t.Muted, ts))
			parts = append(parts, gap)
			usedWidth += 13
		}

		// Container name — pad or truncate to fixed width
		name := entry.Container.Name
		if len(name) > nameW {
			name = name[:nameW-1] + "…"
		}
		parts = append(parts, styledBold(lipgloss.Color(entry.Container.Color), fmt.Sprintf("%-*s", nameW, name)))
		parts = append(parts, gap)
		usedWidth += nameW + 1

		// Level
		levelColor := t.InfoColor
		switch entry.Level {
		case model.LevelError:
			levelColor = t.ErrorColor
		case model.LevelWarn:
			levelColor = t.WarnColor
		case model.LevelDebug:
			levelColor = t.DebugColor
		}
		levelStr := string(entry.Level)
		if levelStr == "" {
			levelStr = "     "
		}
		parts = append(parts, styled(levelColor, fmt.Sprintf("%-5s", levelStr)))
		parts = append(parts, gap)
		usedWidth += 6

		// Message — truncate or wrap
		msgColor := t.Foreground
		if entry.Level == model.LevelError {
			msgColor = t.ErrorColor
		} else if entry.Level == model.LevelWarn {
			msgColor = t.WarnColor
		}
		msg := entry.Message
		msgWidth := logWidth - usedWidth
		if msgWidth < 1 {
			msgWidth = 1
		}

		if !m.WrapLines {
			if len(msg) > msgWidth {
				if msgWidth > 3 {
					msg = msg[:msgWidth-3] + "..."
				} else {
					msg = msg[:msgWidth]
				}
			}
			parts = append(parts, styled(msgColor, msg))
		} else {
			// Wrap message manually with indent on continuation lines
			indent := lipgloss.NewStyle().Background(bg).Render(strings.Repeat(" ", usedWidth-1)) // -1 for border
			msgRunes := []rune(msg)
			first := true
			for len(msgRunes) > 0 {
				chunkLen := msgWidth
				if chunkLen > len(msgRunes) {
					chunkLen = len(msgRunes)
				}
				chunk := string(msgRunes[:chunkLen])
				msgRunes = msgRunes[chunkLen:]
				if first {
					parts = append(parts, styled(msgColor, chunk))
					first = false
				} else {
					parts = append(parts, "\n"+indent+styled(msgColor, chunk))
				}
			}
		}

		line := strings.Join(parts, "")

		// Render with fixed width and background to fill remaining space
		rendered := border + lipgloss.NewStyle().
			Width(logWidth - 1).
			MaxWidth(logWidth - 1).
			Background(bg).
			Render(line)

		lines = append(lines, rendered)
	}

	// Join all rendered lines, then split by actual visual rows
	// (wrapped lines produce multiple \n-separated rows)
	joined := lipgloss.JoinVertical(lipgloss.Left, lines...)
	visualRows := strings.Split(joined, "\n")

	// Clip to exactly m.Height visual rows so wrapping doesn't overflow the panel
	bgFill := lipgloss.NewStyle().Width(logWidth).Background(t.Background)
	if len(visualRows) > m.Height {
		visualRows = visualRows[:m.Height]
	}
	for len(visualRows) < m.Height {
		visualRows = append(visualRows, bgFill.Render(""))
	}

	return strings.Join(visualRows, "\n")
}

// Freeze toggles freeze state and adjusts cursor.
func (m *LogViewModel) Freeze() {
	m.Frozen = !m.Frozen
	if m.Frozen {
		m.CursorLine = len(m.FilteredLogs) - 1
	} else {
		m.CursorLine = -1
		m.SelectedLines = make(map[int]bool)
		m.SelAnchor = -1
	}
}

// ClickLine auto-freezes and moves cursor to the given line index.
func (m *LogViewModel) ClickLine(lineIdx int) {
	if lineIdx >= 0 && lineIdx < len(m.FilteredLogs) {
		if !m.Frozen {
			m.Frozen = true
		}
		m.CursorLine = lineIdx
		m.SelectedLines = make(map[int]bool)
		m.SelAnchor = -1
	}
}

// ShiftClickLine selects a range from the current cursor (or anchor) to the
// clicked line index, auto-freezing if necessary.
func (m *LogViewModel) ShiftClickLine(lineIdx int) {
	if lineIdx < 0 || lineIdx >= len(m.FilteredLogs) {
		return
	}
	if !m.Frozen {
		m.Frozen = true
	}

	anchor := m.SelAnchor
	if anchor < 0 {
		anchor = m.CursorLine
	}
	if anchor < 0 {
		anchor = 0
	}

	m.SelectedLines = make(map[int]bool)
	lo, hi := anchor, lineIdx
	if lo > hi {
		lo, hi = hi, lo
	}
	for i := lo; i <= hi; i++ {
		m.SelectedLines[i] = true
	}
	m.CursorLine = lineIdx
}

// CtrlClickLine toggles the clicked line in the selection without clearing
// existing selections. Auto-freezes if needed.
func (m *LogViewModel) CtrlClickLine(lineIdx int) {
	if lineIdx < 0 || lineIdx >= len(m.FilteredLogs) {
		return
	}
	if !m.Frozen {
		m.Frozen = true
	}

	if m.SelectedLines[lineIdx] {
		delete(m.SelectedLines, lineIdx)
	} else {
		m.SelectedLines[lineIdx] = true
	}
	m.CursorLine = lineIdx
	m.SelAnchor = lineIdx
}

// CopyLine returns the text of a single log line for clipboard copy.
func (m *LogViewModel) CopyLine(lineIdx int) string {
	if lineIdx < 0 || lineIdx >= len(m.FilteredLogs) {
		return ""
	}
	entry := m.FilteredLogs[lineIdx]
	var parts []string
	if m.ShowTimestamps {
		parts = append(parts, entry.Timestamp.Format("15:04:05.000"))
	}
	parts = append(parts, fmt.Sprintf("[%s]", entry.Container.Name))
	parts = append(parts, stripANSI(entry.Message))
	return strings.Join(parts, " ")
}

// VisibleStartIndex returns the index of the first visible log line in the
// current viewport, matching the logic in View().
func (m *LogViewModel) VisibleStartIndex() int {
	startIdx := 0
	if !m.Frozen {
		startIdx = len(m.FilteredLogs) - m.Height
	} else if m.CursorLine >= 0 {
		startIdx = m.CursorLine - m.Height/2
	}
	if startIdx < 0 {
		startIdx = 0
	}
	return startIdx
}

// ScrollUp scrolls the view up by n lines.
func (m *LogViewModel) ScrollUp(n int) {
	if m.Frozen {
		m.CursorLine -= n
		if m.CursorLine < 0 {
			m.CursorLine = 0
		}
	}
}

// ScrollDown scrolls the view down by n lines.
func (m *LogViewModel) ScrollDown(n int) {
	if m.Frozen {
		m.CursorLine += n
		if m.CursorLine >= len(m.FilteredLogs) {
			m.CursorLine = len(m.FilteredLogs) - 1
		}
	}
}

// ClampCursor ensures cursor is within bounds after refilter.
func (m *LogViewModel) ClampCursor() {
	if m.CursorLine >= len(m.FilteredLogs) {
		m.CursorLine = len(m.FilteredLogs) - 1
	}
}
