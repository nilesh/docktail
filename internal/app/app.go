package app

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nilesh/docktail/internal/docker"
	"github.com/nilesh/docktail/internal/model"
	"github.com/nilesh/docktail/internal/theme"
)

const (
	maxLogBuffer  = 5000
	sidebarWidth  = 22
	statusBarHeight = 1
	titleBarHeight  = 1
	shellTabHeight  = 1
)

// FocusArea identifies which panel has focus.
type FocusArea int

const (
	FocusLogs FocusArea = iota
	FocusSidebar
	FocusShell
)

// LogLevelFilter cycles through: ALL, ERROR, WARN, INFO, DEBUG
var levelFilters = []model.LogLevel{"", model.LevelError, model.LevelWarn, model.LevelInfo, model.LevelDebug}

// Options holds startup configuration.
type Options struct {
	Project    string
	Containers []*model.Container
	Client     *docker.Client
	Timestamps bool
	Wrap       bool
	Since      string
}

// Model is the root Bubbletea model for the application.
type Model struct {
	opts       Options
	keys       KeyMap
	styles     theme.Styles
	width      int
	height     int

	// State
	containers    []*model.Container
	logs          []*model.LogEntry
	filteredLogs  []*model.LogEntry
	frozen        bool
	cursorLine    int
	selectedLines map[int]bool
	selAnchor     int
	focus         FocusArea
	sidebarCursor int

	// Timestamps and display
	showTimestamps bool
	wrapLines      bool

	// Filtering
	levelFilter    int // index into levelFilters
	searchQuery    string
	searchRegex    *regexp.Regexp
	isRegexMode    bool
	searchActive   bool
	regexError     string

	// Shell
	shellContainer *model.Container
	shellHeight    int
	shellLines     []string
	shellInput     string
	shellCmdHist   []string
	shellCmdIdx    int

	// Action menu
	actionMenuOpen bool
	actionMenuIdx  int

	// Help
	showHelp bool

	// Notifications
	notification    string
	notificationExp time.Time

	// Docker client
	client  *docker.Client
	logCh   chan docker.LogMessage
	cancel  context.CancelFunc

	// Copied indicator
	copied    bool
	copiedExp time.Time
}

// New creates a new application model.
func New(opts Options) Model {
	m := Model{
		opts:           opts,
		keys:           DefaultKeyMap(),
		styles:         theme.NewStyles(),
		containers:     opts.Containers,
		logs:           make([]*model.LogEntry, 0, maxLogBuffer),
		selectedLines:  make(map[int]bool),
		selAnchor:      -1,
		focus:          FocusLogs,
		showTimestamps: opts.Timestamps,
		wrapLines:      opts.Wrap,
		shellHeight:    10,
		shellCmdIdx:    -1,
		client:         opts.Client,
	}

	return m
}

// LogMsg is a Bubbletea message carrying a new log entry.
type LogMsg struct {
	Entry *model.LogEntry
	Err   error
}

// TickMsg triggers periodic UI updates.
type TickMsg time.Time

// NotifClearMsg clears a notification.
type NotifClearMsg struct{}

// ContainerActionMsg is sent after a container action completes.
type ContainerActionMsg struct {
	ContainerID string
	Action      string
	Err         error
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.startLogStreams(),
		tickCmd(),
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func (m *Model) startLogStreams() tea.Cmd {
	if m.cancel != nil {
		m.cancel()
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.logCh = make(chan docker.LogMessage, 256)

	for _, c := range m.containers {
		if c.Status == model.StatusRunning {
			ch := m.client.StreamLogs(ctx, c, m.opts.Since)
			go func(ch <-chan docker.LogMessage) {
				for msg := range ch {
					m.logCh <- msg
				}
			}(ch)
		}
	}

	return m.waitForLog()
}

func (m *Model) waitForLog() tea.Cmd {
	ch := m.logCh
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return LogMsg(msg)
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.refilter()
		return m, nil

	case TickMsg:
		// Clear expired notifications
		now := time.Now()
		if m.notification != "" && now.After(m.notificationExp) {
			m.notification = ""
		}
		if m.copied && now.After(m.copiedExp) {
			m.copied = false
		}
		return m, tickCmd()

	case LogMsg:
		if msg.Err == nil && msg.Entry != nil {
			m.logs = append(m.logs, msg.Entry)
			if len(m.logs) > maxLogBuffer {
				m.logs = m.logs[len(m.logs)-maxLogBuffer:]
			}
			m.refilter()
		}
		cmds = append(cmds, m.waitForLog())
		return m, tea.Batch(cmds...)

	case ContainerActionMsg:
		if msg.Err != nil {
			m.notify(fmt.Sprintf("Error: %s %s failed", msg.Action, msg.ContainerID))
		}
		return m, nil

	case tea.MouseMsg:
		return m.handleMouse(msg)

	case tea.KeyMsg:
		// Quit always works
		if key.Matches(msg, m.keys.Quit) && !m.searchActive && m.focus != FocusShell {
			if m.cancel != nil {
				m.cancel()
			}
			return m, tea.Quit
		}

		// Help overlay intercepts Esc
		if m.showHelp {
			if msg.String() == "?" || msg.String() == "esc" {
				m.showHelp = false
			}
			return m, nil
		}

		// Action menu intercepts
		if m.actionMenuOpen {
			return m.handleActionMenuKey(msg)
		}

		// Shell focused — only Esc escapes
		if m.focus == FocusShell {
			return m.handleShellKey(msg)
		}

		// Search mode
		if m.searchActive {
			return m.handleSearchKey(msg)
		}

		// Global keys
		if msg.String() == "?" {
			m.showHelp = true
			return m, nil
		}

		if key.Matches(msg, m.keys.Search) {
			m.searchActive = true
			m.searchQuery = ""
			m.regexError = ""
			return m, nil
		}

		if key.Matches(msg, m.keys.Freeze) {
			m.frozen = !m.frozen
			if m.frozen {
				m.cursorLine = len(m.filteredLogs) - 1
			} else {
				m.cursorLine = -1
				m.selectedLines = make(map[int]bool)
				m.selAnchor = -1
			}
			return m, nil
		}

		if key.Matches(msg, m.keys.Timestamps) {
			m.showTimestamps = !m.showTimestamps
			return m, nil
		}

		if key.Matches(msg, m.keys.Wrap) {
			m.wrapLines = !m.wrapLines
			return m, nil
		}

		if key.Matches(msg, m.keys.LevelFilter) {
			m.levelFilter = (m.levelFilter + 1) % len(levelFilters)
			m.refilter()
			return m, nil
		}

		if key.Matches(msg, m.keys.CloseShell) {
			if m.shellContainer != nil {
				m.shellContainer = nil
				m.shellLines = nil
				m.shellInput = ""
				if m.focus == FocusShell {
					m.focus = FocusLogs
				}
			}
			return m, nil
		}

		if key.Matches(msg, m.keys.CycleFocus) {
			m.cycleFocus()
			return m, nil
		}

		// Sidebar keys
		if m.focus == FocusSidebar {
			return m.handleSidebarKey(msg)
		}

		// Log keys (when frozen)
		if m.frozen {
			return m.handleLogKey(msg)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) cycleFocus() {
	if m.shellContainer != nil {
		switch m.focus {
		case FocusSidebar:
			m.focus = FocusLogs
		case FocusLogs:
			m.focus = FocusShell
		case FocusShell:
			m.focus = FocusSidebar
		}
	} else {
		if m.focus == FocusSidebar {
			m.focus = FocusLogs
		} else {
			m.focus = FocusSidebar
		}
	}
}

func (m *Model) notify(msg string) {
	m.notification = msg
	m.notificationExp = time.Now().Add(2 * time.Second)
}

func (m *Model) refilter() {
	lf := levelFilters[m.levelFilter]
	m.filteredLogs = m.filteredLogs[:0]

	for _, entry := range m.logs {
		if !entry.Container.Visible {
			continue
		}
		if lf != "" && entry.Level != lf {
			continue
		}
		if m.searchQuery != "" {
			if m.isRegexMode && m.searchRegex != nil {
				if !m.searchRegex.MatchString(entry.Message) {
					continue
				}
			} else if m.searchQuery != "" {
				if !strings.Contains(strings.ToLower(entry.Message), strings.ToLower(m.searchQuery)) {
					continue
				}
			}
		}
		m.filteredLogs = append(m.filteredLogs, entry)
	}

	// Clamp cursor
	if m.cursorLine >= len(m.filteredLogs) {
		m.cursorLine = len(m.filteredLogs) - 1
	}
}

func (m Model) handleSidebarKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.SidebarUp):
		if m.sidebarCursor > 0 {
			m.sidebarCursor--
		}
	case key.Matches(msg, m.keys.SidebarDown):
		if m.sidebarCursor < len(m.containers)-1 {
			m.sidebarCursor++
		}
	case key.Matches(msg, m.keys.SidebarToggle):
		c := m.containers[m.sidebarCursor]
		c.Visible = !c.Visible
		m.refilter()
	case key.Matches(msg, m.keys.SidebarAction):
		m.actionMenuOpen = true
		m.actionMenuIdx = 0
	case key.Matches(msg, m.keys.SidebarAll):
		allVisible := true
		for _, c := range m.containers {
			if !c.Visible {
				allVisible = false
				break
			}
		}
		for _, c := range m.containers {
			c.Visible = !allVisible
		}
		m.refilter()
	case key.Matches(msg, m.keys.SidebarShell):
		c := m.containers[m.sidebarCursor]
		if c.Status == model.StatusRunning {
			m.shellContainer = c
			m.shellLines = []string{fmt.Sprintf("Connecting to %s...", c.Name)}
			m.shellInput = ""
			m.focus = FocusShell
		}
	}
	return m, nil
}

func (m Model) handleLogKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	maxIdx := len(m.filteredLogs) - 1

	switch {
	case key.Matches(msg, m.keys.Up):
		if m.cursorLine > 0 {
			m.cursorLine--
		}
		if !msg.Alt { // not holding shift
			m.selectedLines = make(map[int]bool)
			m.selAnchor = -1
		}
	case key.Matches(msg, m.keys.Down):
		if m.cursorLine < maxIdx {
			m.cursorLine++
		}
		if !msg.Alt {
			m.selectedLines = make(map[int]bool)
			m.selAnchor = -1
		}
	case key.Matches(msg, m.keys.Top):
		m.cursorLine = 0
	case key.Matches(msg, m.keys.Bottom):
		m.cursorLine = maxIdx
	case key.Matches(msg, m.keys.PageUp):
		m.cursorLine -= 20
		if m.cursorLine < 0 {
			m.cursorLine = 0
		}
	case key.Matches(msg, m.keys.PageDown):
		m.cursorLine += 20
		if m.cursorLine > maxIdx {
			m.cursorLine = maxIdx
		}
	case key.Matches(msg, m.keys.Select):
		if m.selectedLines[m.cursorLine] {
			delete(m.selectedLines, m.cursorLine)
		} else {
			m.selectedLines[m.cursorLine] = true
		}
		m.selAnchor = m.cursorLine
	case key.Matches(msg, m.keys.Copy):
		m.copySelected()
	case key.Matches(msg, m.keys.ClearSel):
		m.selectedLines = make(map[int]bool)
		m.selAnchor = -1
	}
	return m, nil
}

func (m *Model) copySelected() {
	if len(m.selectedLines) == 0 {
		return
	}

	var lines []string
	for i := 0; i < len(m.filteredLogs); i++ {
		if !m.selectedLines[i] {
			continue
		}
		entry := m.filteredLogs[i]
		var parts []string
		if m.showTimestamps {
			parts = append(parts, entry.Timestamp.Format("15:04:05.000"))
		}
		parts = append(parts, fmt.Sprintf("[%s]", entry.Container.Name))
		parts = append(parts, entry.Message)
		lines = append(lines, strings.Join(parts, " "))
	}

	// Note: clipboard integration requires OS-specific handling.
	// In a real TUI this would use atotto/clipboard or similar.
	_ = lines

	m.copied = true
	m.copiedExp = time.Now().Add(1500 * time.Millisecond)
}

func (m Model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.searchActive = false
		m.searchQuery = ""
		m.searchRegex = nil
		m.regexError = ""
		m.refilter()
	case "enter":
		m.searchActive = false
	case "tab":
		m.isRegexMode = !m.isRegexMode
		m.updateSearchRegex()
		m.refilter()
	case "backspace":
		if len(m.searchQuery) > 0 {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
			m.updateSearchRegex()
			m.refilter()
		}
	default:
		if len(msg.String()) == 1 {
			m.searchQuery += msg.String()
			m.updateSearchRegex()
			m.refilter()
		}
	}
	return m, nil
}

func (m *Model) updateSearchRegex() {
	if m.isRegexMode && m.searchQuery != "" {
		re, err := regexp.Compile("(?i)" + m.searchQuery)
		if err != nil {
			m.searchRegex = nil
			m.regexError = "invalid regex"
		} else {
			m.searchRegex = re
			m.regexError = ""
		}
	} else {
		m.searchRegex = nil
		m.regexError = ""
	}
}

func (m Model) handleActionMenuKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	actions := m.getContainerActions(m.containers[m.sidebarCursor])

	switch msg.String() {
	case "esc":
		m.actionMenuOpen = false
	case "up", "k":
		if m.actionMenuIdx > 0 {
			m.actionMenuIdx--
		}
	case "down", "j":
		if m.actionMenuIdx < len(actions)-1 {
			m.actionMenuIdx++
		}
	case "enter":
		if m.actionMenuIdx < len(actions) {
			action := actions[m.actionMenuIdx]
			c := m.containers[m.sidebarCursor]
			m.actionMenuOpen = false
			return m, m.executeContainerAction(c, action.key)
		}
	}
	return m, nil
}

type containerAction struct {
	key   string
	label string
}

func (m *Model) getContainerActions(c *model.Container) []containerAction {
	switch c.Status {
	case model.StatusRunning:
		return []containerAction{
			{"stop", "Stop"},
			{"restart", "Restart"},
			{"pause", "Pause"},
			{"shell", "Shell"},
		}
	case model.StatusPaused:
		return []containerAction{
			{"unpause", "Unpause"},
			{"stop", "Stop"},
		}
	default:
		return []containerAction{
			{"start", "Start"},
		}
	}
}

func (m *Model) executeContainerAction(c *model.Container, action string) tea.Cmd {
	return func() tea.Msg {
		var err error
		switch action {
		case "stop":
			err = m.client.StopContainer(c.ID)
			if err == nil {
				c.Status = model.StatusStopped
			}
		case "start":
			err = m.client.StartContainer(c.ID)
			if err == nil {
				c.Status = model.StatusRunning
			}
		case "restart":
			err = m.client.RestartContainer(c.ID)
			if err == nil {
				c.Status = model.StatusRunning
			}
		case "pause":
			err = m.client.PauseContainer(c.ID)
			if err == nil {
				c.Status = model.StatusPaused
			}
		case "unpause":
			err = m.client.UnpauseContainer(c.ID)
			if err == nil {
				c.Status = model.StatusRunning
			}
		case "shell":
			m.shellContainer = c
			m.shellLines = []string{fmt.Sprintf("Connecting to %s...", c.Name)}
			m.shellInput = ""
			m.focus = FocusShell
		}

		m.notify(fmt.Sprintf("%s %s", strings.Title(action), c.Name))

		return ContainerActionMsg{
			ContainerID: c.ID,
			Action:      action,
			Err:         err,
		}
	}
}

func (m Model) handleShellKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.focus = FocusLogs
	case "enter":
		if m.shellInput != "" {
			m.shellLines = append(m.shellLines, "$ "+m.shellInput)
			// In real implementation, this sends to docker exec
			m.shellLines = append(m.shellLines, "(shell output would appear here)")
			if len(m.shellCmdHist) == 0 || m.shellCmdHist[0] != m.shellInput {
				m.shellCmdHist = append([]string{m.shellInput}, m.shellCmdHist...)
			}
			m.shellInput = ""
			m.shellCmdIdx = -1
		}
	case "up":
		if len(m.shellCmdHist) > 0 {
			m.shellCmdIdx++
			if m.shellCmdIdx >= len(m.shellCmdHist) {
				m.shellCmdIdx = len(m.shellCmdHist) - 1
			}
			m.shellInput = m.shellCmdHist[m.shellCmdIdx]
		}
	case "down":
		if m.shellCmdIdx > 0 {
			m.shellCmdIdx--
			m.shellInput = m.shellCmdHist[m.shellCmdIdx]
		} else {
			m.shellCmdIdx = -1
			m.shellInput = ""
		}
	case "backspace":
		if len(m.shellInput) > 0 {
			m.shellInput = m.shellInput[:len(m.shellInput)-1]
		}
	default:
		if len(msg.String()) == 1 {
			m.shellInput += msg.String()
		}
	}
	return m, nil
}

func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch msg.Button {
	case tea.MouseButtonLeft:
		if msg.Action != tea.MouseActionRelease {
			return m, nil
		}
		// Determine click area
		x, y := msg.X, msg.Y

		// Title bar area
		if y < titleBarHeight {
			return m, nil
		}

		// Status bar area
		if y >= m.height-statusBarHeight {
			return m, nil
		}

		contentY := y - titleBarHeight

		// Sidebar click
		if x < sidebarWidth {
			m.focus = FocusSidebar
			// Header is line 0, containers start at line 1
			containerIdx := contentY - 1
			if containerIdx >= 0 && containerIdx < len(m.containers) {
				m.sidebarCursor = containerIdx
				// Left click toggles visibility
				m.containers[containerIdx].Visible = !m.containers[containerIdx].Visible
				m.refilter()
			}
			return m, nil
		}

		// Shell area click
		if m.shellContainer != nil {
			logAreaHeight := m.logAreaHeight()
			shellStart := titleBarHeight + logAreaHeight + 1 + shellTabHeight // +1 for resize handle
			if y >= shellStart && y < m.height-statusBarHeight {
				m.focus = FocusShell
				return m, nil
			}
		}

		// Log area click
		m.focus = FocusLogs
		logLineIdx := contentY // relative to log area start
		if logLineIdx >= 0 && logLineIdx < len(m.filteredLogs) {
			// Auto-freeze on click
			if !m.frozen {
				m.frozen = true
			}
			m.cursorLine = logLineIdx
			// Shift-click for range selection would be handled via msg modifiers
			// For now, single click moves cursor
			m.selectedLines = make(map[int]bool)
			m.selAnchor = -1
		}
		return m, nil

	case tea.MouseButtonRight:
		if msg.Action != tea.MouseActionRelease {
			return m, nil
		}
		x, y := msg.X, msg.Y
		contentY := y - titleBarHeight

		// Right-click on sidebar opens action menu
		if x < sidebarWidth {
			containerIdx := contentY - 1
			if containerIdx >= 0 && containerIdx < len(m.containers) {
				m.sidebarCursor = containerIdx
				m.actionMenuOpen = true
				m.actionMenuIdx = 0
				m.focus = FocusSidebar
			}
		}
		return m, nil

	case tea.MouseButtonWheelUp:
		if m.frozen {
			m.cursorLine -= 3
			if m.cursorLine < 0 {
				m.cursorLine = 0
			}
		}
		return m, nil

	case tea.MouseButtonWheelDown:
		if m.frozen {
			m.cursorLine += 3
			if m.cursorLine >= len(m.filteredLogs) {
				m.cursorLine = len(m.filteredLogs) - 1
			}
		}
		return m, nil
	}

	return m, nil
}

func (m Model) logAreaHeight() int {
	h := m.height - titleBarHeight - statusBarHeight
	if m.shellContainer != nil {
		h -= m.shellHeight + 1 + shellTabHeight // 1 for resize handle
	}
	if h < 5 {
		h = 5
	}
	return h
}

// View renders the entire TUI.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	t := theme.Current

	// Title bar
	titleBar := m.renderTitleBar()

	// Sidebar
	sidebar := m.renderSidebar()

	// Log view
	logView := m.renderLogView()

	// Shell (if open)
	var shellView string
	if m.shellContainer != nil {
		shellView = m.renderShellPanel()
	}

	// Status bar
	statusBar := m.renderStatusBar()

	// Compose right panel
	rightPanel := logView
	if shellView != "" {
		resizeHandle := lipgloss.NewStyle().
			Width(m.width - sidebarWidth).
			Background(func() lipgloss.Color {
				if m.focus == FocusShell {
					return t.Accent
				}
				return t.Border
			}()).
			Render("─")
		rightPanel = lipgloss.JoinVertical(lipgloss.Left, logView, resizeHandle, shellView)
	}

	// Main area: sidebar + right panel
	mainArea := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, rightPanel)

	// Full layout
	full := lipgloss.JoinVertical(lipgloss.Left, titleBar, mainArea, statusBar)

	// Help overlay
	if m.showHelp {
		full = m.renderHelp()
	}

	// Action menu overlay
	if m.actionMenuOpen {
		// Menu renders on top; for simplicity we append it
		// In a real implementation we'd use an overlay
	}

	return full
}

func (m Model) renderTitleBar() string {
	t := theme.Current
	left := lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("◉ docktail")
	left += lipgloss.NewStyle().Foreground(t.Muted).Render(" │ ")
	left += lipgloss.NewStyle().Foreground(t.Muted).Render("project: ")
	left += lipgloss.NewStyle().Foreground(t.Foreground).Render(m.opts.Project)
	left += lipgloss.NewStyle().Foreground(t.Muted).Render(" │ ")

	visCount := 0
	for _, c := range m.containers {
		if c.Visible {
			visCount++
		}
	}
	left += fmt.Sprintf("containers: %d/%d", visCount, len(m.containers))

	var right string
	if m.notification != "" {
		right += lipgloss.NewStyle().Foreground(t.OrangeColor).Render(m.notification) + " "
	}
	if m.copied {
		right += lipgloss.NewStyle().Foreground(t.GreenColor).Render("✓ copied") + " "
	}
	if m.searchActive {
		mode := "/"
		if m.isRegexMode {
			mode = "regex:"
		}
		right += lipgloss.NewStyle().Foreground(t.OrangeColor).Render(mode+m.searchQuery+"▌") + " "
		if m.regexError != "" {
			right += lipgloss.NewStyle().Foreground(t.ErrorColor).Render(m.regexError) + " "
		}
	} else if m.searchQuery != "" {
		label := "search"
		if m.isRegexMode {
			label = "regex"
		}
		right += lipgloss.NewStyle().Foreground(t.Muted).Render(label+": "+m.searchQuery) + " "
	}
	if m.frozen {
		right += lipgloss.NewStyle().Foreground(t.OrangeColor).Render("❄ FROZEN") + " "
	}

	padding := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if padding < 0 {
		padding = 0
	}

	return lipgloss.NewStyle().
		Width(m.width).
		Background(lipgloss.Color("#161b22")).
		Render(left + strings.Repeat(" ", padding) + right)
}

func (m Model) renderSidebar() string {
	t := theme.Current
	lines := make([]string, 0, len(m.containers)+3)

	// Header
	headerColor := t.Muted
	if m.focus == FocusSidebar {
		headerColor = t.Accent
	}
	header := lipgloss.NewStyle().
		Foreground(headerColor).
		Bold(true).
		Width(sidebarWidth).
		Render("CONTAINERS")
	lines = append(lines, header)

	// Container list
	for i, c := range m.containers {
		focused := m.focus == FocusSidebar && m.sidebarCursor == i
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

		style := lipgloss.NewStyle().Width(sidebarWidth)
		if focused {
			style = style.Background(lipgloss.Color("#1f2937"))
		}

		line := lipgloss.NewStyle().Foreground(visColor).Render(vis) + " "
		line += lipgloss.NewStyle().Foreground(lipgloss.Color(c.Color)).Bold(true).Render(c.Name) + " "
		line += lipgloss.NewStyle().Foreground(t.Muted).Render(statusIcon)

		lines = append(lines, style.Render(line))
	}

	// Fill remaining height
	contentHeight := m.height - titleBarHeight - statusBarHeight
	for len(lines) < contentHeight {
		lines = append(lines, lipgloss.NewStyle().Width(sidebarWidth).Render(""))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m Model) renderLogView() string {
	t := theme.Current
	logHeight := m.logAreaHeight()
	logWidth := m.width - sidebarWidth

	var lines []string

	// Determine visible range
	startIdx := 0
	if !m.frozen {
		startIdx = len(m.filteredLogs) - logHeight
	} else if m.cursorLine >= 0 {
		// Keep cursor in view
		startIdx = m.cursorLine - logHeight/2
	}
	if startIdx < 0 {
		startIdx = 0
	}
	endIdx := startIdx + logHeight
	if endIdx > len(m.filteredLogs) {
		endIdx = len(m.filteredLogs)
	}

	for i := startIdx; i < endIdx; i++ {
		entry := m.filteredLogs[i]
		isCursor := m.frozen && m.cursorLine == i
		isSelected := m.selectedLines[i]

		var line string

		if m.frozen {
			lineNum := lipgloss.NewStyle().Foreground(t.Border).Width(4).Align(lipgloss.Right).Render(fmt.Sprintf("%d", i+1))
			line += lineNum + " "
		}

		if m.showTimestamps {
			ts := entry.Timestamp.Format("15:04:05.000")
			line += lipgloss.NewStyle().Foreground(t.Muted).Render(ts) + " "
		}

		line += lipgloss.NewStyle().
			Foreground(lipgloss.Color(entry.Container.Color)).
			Bold(true).
			Width(10).
			Render(entry.Container.Name) + " "

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
		line += lipgloss.NewStyle().Foreground(levelColor).Width(5).Render(levelStr) + " "

		msgColor := t.Foreground
		if entry.Level == model.LevelError {
			msgColor = t.ErrorColor
		} else if entry.Level == model.LevelWarn {
			msgColor = t.WarnColor
		}
		line += lipgloss.NewStyle().Foreground(msgColor).Render(entry.Message)

		style := lipgloss.NewStyle().Width(logWidth)
		if isSelected {
			style = style.Background(t.SelectedBg)
		} else if isCursor {
			style = style.Background(t.CursorBg)
		}

		lines = append(lines, style.Render(line))
	}

	// Fill remaining height
	for len(lines) < logHeight {
		lines = append(lines, lipgloss.NewStyle().Width(logWidth).Render(""))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m Model) renderShellPanel() string {
	t := theme.Current
	shellWidth := m.width - sidebarWidth

	// Tab bar
	tabBar := lipgloss.NewStyle().
		Width(shellWidth).
		Background(lipgloss.Color("#161b22")).
		Render(
			lipgloss.NewStyle().Foreground(t.Accent).Render(">_ ") +
				lipgloss.NewStyle().Foreground(lipgloss.Color(m.shellContainer.Color)).Render(m.shellContainer.Name) +
				lipgloss.NewStyle().Foreground(t.Muted).Render(" shell") +
				strings.Repeat(" ", shellWidth-30) +
				lipgloss.NewStyle().Foreground(t.Muted).Render("✕"),
		)

	// Shell content
	var shellLines []string
	visibleLines := m.shellHeight - 1 // -1 for input line
	start := 0
	if len(m.shellLines) > visibleLines {
		start = len(m.shellLines) - visibleLines
	}
	for i := start; i < len(m.shellLines); i++ {
		shellLines = append(shellLines, lipgloss.NewStyle().Width(shellWidth).Foreground(t.Foreground).Render(m.shellLines[i]))
	}

	// Input line
	prompt := "$ "
	inputLine := lipgloss.NewStyle().Foreground(t.GreenColor).Render(prompt) +
		lipgloss.NewStyle().Foreground(t.Foreground).Render(m.shellInput)
	if m.focus == FocusShell {
		inputLine += lipgloss.NewStyle().Foreground(t.Accent).Render("▌")
	}
	shellLines = append(shellLines, lipgloss.NewStyle().Width(shellWidth).Render(inputLine))

	// Fill
	for len(shellLines) < m.shellHeight {
		shellLines = append(shellLines, lipgloss.NewStyle().Width(shellWidth).Render(""))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, shellLines...)
	return lipgloss.JoinVertical(lipgloss.Left, tabBar, content)
}

func (m Model) renderStatusBar() string {
	t := theme.Current

	var left string
	left += lipgloss.NewStyle().Foreground(t.Accent).Render("f") + lipgloss.NewStyle().Foreground(t.Muted).Render(" freeze ")
	left += lipgloss.NewStyle().Foreground(t.Accent).Render("t") + lipgloss.NewStyle().Foreground(t.Muted).Render(" timestamps ")
	left += lipgloss.NewStyle().Foreground(t.Accent).Render("w") + lipgloss.NewStyle().Foreground(t.Muted).Render(" wrap ")
	left += lipgloss.NewStyle().Foreground(t.Accent).Render("l") + lipgloss.NewStyle().Foreground(t.Muted).Render(fmt.Sprintf(" %s ", levelFilterLabel(m.levelFilter)))
	left += lipgloss.NewStyle().Foreground(t.Accent).Render("/") + lipgloss.NewStyle().Foreground(t.Muted).Render(" search ")
	if m.frozen {
		left += lipgloss.NewStyle().Foreground(t.Muted).Render("│ ")
		left += lipgloss.NewStyle().Foreground(t.Accent).Render("⎵") + lipgloss.NewStyle().Foreground(t.Muted).Render(" select ")
		left += lipgloss.NewStyle().Foreground(t.Accent).Render("y") + lipgloss.NewStyle().Foreground(t.Muted).Render(" copy ")
	}

	var right string
	if m.focus == FocusShell {
		right += lipgloss.NewStyle().Foreground(t.Accent).Render("SHELL ")
	}
	if len(m.selectedLines) > 0 {
		right += lipgloss.NewStyle().Foreground(t.OrangeColor).Render(fmt.Sprintf("%d selected ", len(m.selectedLines)))
	}
	right += lipgloss.NewStyle().Foreground(t.Muted).Render(fmt.Sprintf("%d lines", len(m.filteredLogs)))
	if m.frozen && m.cursorLine >= 0 {
		right += lipgloss.NewStyle().Foreground(t.Muted).Render(fmt.Sprintf(" ln %d", m.cursorLine+1))
	}

	padding := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if padding < 0 {
		padding = 0
	}

	return lipgloss.NewStyle().
		Width(m.width).
		Background(func() lipgloss.Color {
			if m.frozen {
				return t.FrozenBg
			}
			return lipgloss.Color("#161b22")
		}()).
		Render(left + strings.Repeat(" ", padding) + right)
}

func levelFilterLabel(idx int) string {
	if idx == 0 {
		return "all"
	}
	return strings.ToLower(string(levelFilters[idx]))
}

func (m Model) renderHelp() string {
	t := theme.Current
	sections := []struct {
		title string
		keys  [][2]string
	}{
		{"General", [][2]string{
			{"?", "Toggle help"},
			{"f", "Freeze/unfreeze"},
			{"t", "Toggle timestamps"},
			{"w", "Toggle wrap"},
			{"l", "Cycle log levels"},
			{"/", "Search (Tab: regex)"},
			{"x", "Close shell"},
			{"Tab", "Cycle focus"},
			{"q", "Quit"},
		}},
		{"Sidebar", [][2]string{
			{"↑/↓ j/k", "Navigate"},
			{"Space", "Toggle container"},
			{"Enter", "Actions menu"},
			{"s", "Open shell"},
			{"a", "Select all"},
		}},
		{"Logs (frozen)", [][2]string{
			{"↑/↓ j/k", "Move cursor"},
			{"g/G", "Top/bottom"},
			{"PgUp/Dn", "Page scroll"},
			{"Space", "Select line"},
			{"Shift+↑↓", "Range select"},
			{"y/c", "Copy selected"},
			{"Esc", "Clear selection"},
		}},
		{"Shell", [][2]string{
			{"↑/↓", "Command history"},
			{"Enter", "Execute"},
			{"Esc", "Back to logs"},
		}},
	}

	var content string
	content += lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("Keyboard Shortcuts") + "\n\n"

	for _, sec := range sections {
		content += lipgloss.NewStyle().Foreground(t.Muted).Bold(true).Render(strings.ToUpper(sec.title)) + "\n"
		for _, kv := range sec.keys {
			k := lipgloss.NewStyle().Foreground(t.Accent).Width(14).Align(lipgloss.Right).Render(kv[0])
			v := lipgloss.NewStyle().Foreground(t.Foreground).Render(kv[1])
			content += k + "  " + v + "\n"
		}
		content += "\n"
	}

	content += lipgloss.NewStyle().Foreground(t.Muted).Render("Press ? or Esc to close")

	box := lipgloss.NewStyle().
		Background(lipgloss.Color("#161b22")).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Border).
		Padding(1, 2).
		Width(50).
		Render(content)

	// Center the help box
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box,
		lipgloss.WithWhitespaceBackground(lipgloss.Color("#000000")))
}
