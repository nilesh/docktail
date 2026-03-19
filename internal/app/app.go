package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nilesh/docktail/internal/docker"
	"github.com/nilesh/docktail/internal/model"
	"github.com/nilesh/docktail/internal/theme"
	"github.com/nilesh/docktail/internal/ui"
)

const (
	maxLogBuffer    = 5000
	sidebarWidth    = 22
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
	opts   Options
	keys   KeyMap
	width  int
	height int
	focus  FocusArea

	// Sub-models
	sidebar    ui.SidebarModel
	logView    ui.LogViewModel
	shell      ui.ShellModel
	search     ui.SearchModel
	actionMenu ui.ActionMenuModel
	help       ui.HelpModel

	// Filtering
	levelFilter int

	// Notifications
	notification    string
	notificationExp time.Time
	copied          bool
	copiedExp       time.Time

	// Docker client
	client *docker.Client
	logCh  chan docker.LogMessage
	cancel context.CancelFunc
}

// New creates a new application model.
func New(opts Options) Model {
	m := Model{
		opts:   opts,
		keys:   DefaultKeyMap(),
		focus:  FocusLogs,
		client: opts.Client,
		sidebar: ui.SidebarModel{
			Containers: opts.Containers,
			Width:      sidebarWidth,
		},
		logView: ui.LogViewModel{
			Logs:           make([]*model.LogEntry, 0, maxLogBuffer),
			SelectedLines:  make(map[int]bool),
			SelAnchor:      -1,
			ShowTimestamps: opts.Timestamps,
			WrapLines:      opts.Wrap,
		},
		shell: ui.NewShellModel(),
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

	for _, c := range m.sidebar.Containers {
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
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateDimensions()
		m.refilter()
		return m, nil

	case TickMsg:
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
			m.logView.Logs = append(m.logView.Logs, msg.Entry)
			if len(m.logView.Logs) > maxLogBuffer {
				m.logView.Logs = m.logView.Logs[len(m.logView.Logs)-maxLogBuffer:]
			}
			m.refilter()
		}
		return m, m.waitForLog()

	case ContainerActionMsg:
		if msg.Err != nil {
			m.notify(fmt.Sprintf("Error: %s %s failed", msg.Action, msg.ContainerID))
		}
		return m, nil

	case ui.RefilterMsg:
		m.refilter()
		return m, nil

	case ui.OpenShellMsg:
		m.shell.Open(msg.Container)
		m.focus = FocusShell
		m.updateDimensions()
		return m, nil

	case ui.OpenActionMenuMsg:
		m.actionMenu.OpenMenu(m.sidebar.SelectedContainer())
		return m, nil

	case ui.ShellFocusLogs:
		m.focus = FocusLogs
		return m, nil

	case ui.CopiedMsg:
		// Phase 3 will wire clipboard here
		_ = msg.Text
		m.copied = true
		m.copiedExp = time.Now().Add(1500 * time.Millisecond)
		return m, nil

	case ui.ExecuteActionMsg:
		m.actionMenu.Close()
		return m, m.executeContainerAction(msg.Container, msg.Action)

	case tea.MouseMsg:
		return m.handleMouse(msg)

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Quit always works (except in search/shell)
	if key.Matches(msg, m.keys.Quit) && !m.search.Active && m.focus != FocusShell {
		if m.cancel != nil {
			m.cancel()
		}
		return m, tea.Quit
	}

	// Help overlay intercepts
	if m.help.Visible {
		m.help.HandleKey(msg.String())
		return m, nil
	}

	// Action menu intercepts
	if m.actionMenu.Open {
		var cmd tea.Cmd
		m.actionMenu, cmd = m.actionMenu.Update(msg, m.sidebar.SelectedContainer())
		return m, cmd
	}

	// Shell focused
	if m.focus == FocusShell {
		var cmd tea.Cmd
		m.shell, cmd = m.shell.Update(msg)
		return m, cmd
	}

	// Search mode
	if m.search.Active {
		var cmd tea.Cmd
		m.search, cmd = m.search.Update(msg)
		return m, cmd
	}

	// Global keys
	if msg.String() == "?" {
		m.help.Toggle()
		return m, nil
	}

	if key.Matches(msg, m.keys.Search) {
		m.search.Activate()
		return m, nil
	}

	if key.Matches(msg, m.keys.Freeze) {
		m.logView.Freeze()
		return m, nil
	}

	if key.Matches(msg, m.keys.Timestamps) {
		m.logView.ShowTimestamps = !m.logView.ShowTimestamps
		return m, nil
	}

	if key.Matches(msg, m.keys.Wrap) {
		m.logView.WrapLines = !m.logView.WrapLines
		return m, nil
	}

	if key.Matches(msg, m.keys.LevelFilter) {
		m.levelFilter = (m.levelFilter + 1) % len(ui.LevelFilters)
		m.refilter()
		return m, nil
	}

	if key.Matches(msg, m.keys.CloseShell) {
		if m.shell.IsOpen() {
			m.shell.Close()
			if m.focus == FocusShell {
				m.focus = FocusLogs
			}
			m.updateDimensions()
		}
		return m, nil
	}

	if key.Matches(msg, m.keys.CycleFocus) {
		m.cycleFocus()
		return m, nil
	}

	// Sidebar keys
	if m.focus == FocusSidebar {
		sidebarKeys := ui.SidebarKeyMap{
			Up:     m.keys.SidebarUp,
			Down:   m.keys.SidebarDown,
			Toggle: m.keys.SidebarToggle,
			Action: m.keys.SidebarAction,
			All:    m.keys.SidebarAll,
			Shell:  m.keys.SidebarShell,
		}
		var cmd tea.Cmd
		m.sidebar, cmd = m.sidebar.Update(msg, sidebarKeys)
		return m, cmd
	}

	// Log keys (when frozen)
	if m.logView.Frozen {
		logKeys := ui.LogViewKeyMap{
			Up:       m.keys.Up,
			Down:     m.keys.Down,
			Top:      m.keys.Top,
			Bottom:   m.keys.Bottom,
			PageUp:   m.keys.PageUp,
			PageDown: m.keys.PageDown,
			Select:   m.keys.Select,
			Copy:     m.keys.Copy,
			ClearSel: m.keys.ClearSel,
		}
		var cmd tea.Cmd
		m.logView, cmd = m.logView.Update(msg, logKeys)
		return m, cmd
	}

	return m, nil
}

func (m *Model) cycleFocus() {
	if m.shell.IsOpen() {
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
	m.sidebar.Focused = m.focus == FocusSidebar
	m.shell.Focused = m.focus == FocusShell
}

func (m *Model) notify(msg string) {
	m.notification = msg
	m.notificationExp = time.Now().Add(2 * time.Second)
}

func (m *Model) refilter() {
	lf := ui.LevelFilters[m.levelFilter]
	m.logView.FilteredLogs = m.logView.FilteredLogs[:0]

	for _, entry := range m.logView.Logs {
		if !entry.Container.Visible {
			continue
		}
		if lf != "" && entry.Level != lf {
			continue
		}
		if !m.search.Matches(entry.Message) {
			continue
		}
		m.logView.FilteredLogs = append(m.logView.FilteredLogs, entry)
	}

	m.logView.ClampCursor()
}

func (m *Model) updateDimensions() {
	contentHeight := m.height - titleBarHeight - statusBarHeight
	m.sidebar.Height = contentHeight
	m.sidebar.Focused = m.focus == FocusSidebar

	logHeight := contentHeight
	if m.shell.IsOpen() {
		logHeight -= m.shell.Height + 1 + shellTabHeight
	}
	if logHeight < 5 {
		logHeight = 5
	}
	m.logView.Width = m.width - sidebarWidth
	m.logView.Height = logHeight

	m.shell.Focused = m.focus == FocusShell
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
			// Handled via OpenShellMsg
		}

		label := strings.ToUpper(action[:1]) + action[1:]
		return ContainerActionMsg{
			ContainerID: c.ID,
			Action:      label + " " + c.Name,
			Err:         err,
		}
	}
}

func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch msg.Button {
	case tea.MouseButtonLeft:
		if msg.Action != tea.MouseActionRelease {
			return m, nil
		}
		x, y := msg.X, msg.Y

		if y < titleBarHeight || y >= m.height-statusBarHeight {
			return m, nil
		}

		contentY := y - titleBarHeight

		// Sidebar click
		if x < sidebarWidth {
			m.focus = FocusSidebar
			m.sidebar.Focused = true
			m.shell.Focused = false
			m.sidebar.HandleClick(contentY)
			m.refilter()
			return m, nil
		}

		// Shell area click
		if m.shell.IsOpen() {
			logAreaHeight := m.logView.Height
			shellStart := titleBarHeight + logAreaHeight + 1 + shellTabHeight
			if y >= shellStart && y < m.height-statusBarHeight {
				m.focus = FocusShell
				m.shell.Focused = true
				m.sidebar.Focused = false
				return m, nil
			}
		}

		// Log area click
		m.focus = FocusLogs
		m.sidebar.Focused = false
		m.shell.Focused = false
		m.logView.ClickLine(contentY)
		return m, nil

	case tea.MouseButtonRight:
		if msg.Action != tea.MouseActionRelease {
			return m, nil
		}
		x, y := msg.X, msg.Y
		contentY := y - titleBarHeight

		if x < sidebarWidth {
			if m.sidebar.HandleRightClick(contentY) {
				m.actionMenu.OpenMenu(m.sidebar.SelectedContainer())
				m.focus = FocusSidebar
				m.sidebar.Focused = true
				m.shell.Focused = false
			}
		}
		return m, nil

	case tea.MouseButtonWheelUp:
		m.logView.ScrollUp(3)
		return m, nil

	case tea.MouseButtonWheelDown:
		m.logView.ScrollDown(3)
		return m, nil
	}

	return m, nil
}

// View renders the entire TUI.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	t := theme.Current

	// Title bar
	titleBar := ui.TitleBarView(m.width, m.opts.Project,
		m.sidebar.VisibleCount(), len(m.sidebar.Containers),
		m.notification, m.copied, m.search, m.logView.Frozen)

	// Sidebar
	sidebarView := m.sidebar.View()

	// Log view
	logView := m.logView.View()

	// Shell (if open)
	var shellView string
	if m.shell.IsOpen() {
		shellView = m.shell.View(m.width - sidebarWidth)
	}

	// Status bar
	statusBar := ui.StatusBarView(m.width, m.logView.Frozen,
		m.focus == FocusShell, len(m.logView.SelectedLines),
		len(m.logView.FilteredLogs), m.logView.CursorLine, m.levelFilter)

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
	mainArea := lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, rightPanel)

	// Full layout
	full := lipgloss.JoinVertical(lipgloss.Left, titleBar, mainArea, statusBar)

	// Help overlay
	if m.help.Visible {
		full = m.help.View(m.width, m.height)
	}

	return full
}
