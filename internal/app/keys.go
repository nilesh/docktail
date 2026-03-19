package app

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all keybindings for the application.
type KeyMap struct {
	// Global
	Help      key.Binding
	Quit      key.Binding
	Freeze    key.Binding
	Timestamps key.Binding
	Wrap      key.Binding
	LevelFilter key.Binding
	Search    key.Binding
	CloseShell key.Binding
	CycleFocus  key.Binding
	ToggleTheme key.Binding

	// Sidebar
	SidebarUp     key.Binding
	SidebarDown   key.Binding
	SidebarToggle key.Binding
	SidebarAction key.Binding
	SidebarAll    key.Binding
	SidebarShell  key.Binding

	// Log navigation (frozen)
	Up       key.Binding
	Down     key.Binding
	Top      key.Binding
	Bottom   key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Select   key.Binding
	Copy     key.Binding
	ClearSel key.Binding

	// Search
	SearchNext key.Binding
	SearchPrev key.Binding
}

// DefaultKeyMap returns the default keybindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Help:      key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "toggle help")),
		Quit:      key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Freeze:    key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "freeze/unfreeze")),
		Timestamps: key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "toggle timestamps")),
		Wrap:      key.NewBinding(key.WithKeys("w"), key.WithHelp("w", "toggle wrap")),
		LevelFilter: key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "cycle log level")),
		Search:    key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
		CloseShell: key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "close shell")),
		CycleFocus:  key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "cycle focus")),
		ToggleTheme: key.NewBinding(key.WithKeys("T"), key.WithHelp("T", "toggle theme")),

		SidebarUp:     key.NewBinding(key.WithKeys("up", "k")),
		SidebarDown:   key.NewBinding(key.WithKeys("down", "j")),
		SidebarToggle: key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "toggle container")),
		SidebarAction: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "actions")),
		SidebarAll:    key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "select all")),
		SidebarShell:  key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "open shell")),

		Up:       key.NewBinding(key.WithKeys("up", "k")),
		Down:     key.NewBinding(key.WithKeys("down", "j")),
		Top:      key.NewBinding(key.WithKeys("g")),
		Bottom:   key.NewBinding(key.WithKeys("G")),
		PageUp:   key.NewBinding(key.WithKeys("pgup")),
		PageDown: key.NewBinding(key.WithKeys("pgdown")),
		Select:   key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "select line")),
		Copy:     key.NewBinding(key.WithKeys("y", "c"), key.WithHelp("y", "copy")),
		ClearSel: key.NewBinding(key.WithKeys("esc")),

		SearchNext: key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "next match")),
		SearchPrev: key.NewBinding(key.WithKeys("N"), key.WithHelp("N", "prev match")),
	}
}
