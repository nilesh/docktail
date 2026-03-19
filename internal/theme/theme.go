package theme

import "github.com/charmbracelet/lipgloss"

// Theme holds all style definitions for the TUI.
type Theme struct {
	// Base colors
	Background  lipgloss.Color
	Foreground  lipgloss.Color
	Muted       lipgloss.Color
	Border      lipgloss.Color
	Accent      lipgloss.Color
	AccentDim   lipgloss.Color

	// Log level colors
	ErrorColor  lipgloss.Color
	WarnColor   lipgloss.Color
	InfoColor   lipgloss.Color
	DebugColor  lipgloss.Color

	// UI state colors
	SelectedBg  lipgloss.Color
	CursorBg    lipgloss.Color
	CursorBorder lipgloss.Color
	FrozenBg    lipgloss.Color
	GreenColor  lipgloss.Color
	OrangeColor lipgloss.Color
	SearchHighlight lipgloss.Color
}

// Dark is the default dark theme.
var Dark = Theme{
	Background:  lipgloss.Color("#0d1117"),
	Foreground:  lipgloss.Color("#c9d1d9"),
	Muted:       lipgloss.Color("#484f58"),
	Border:      lipgloss.Color("#30363d"),
	Accent:      lipgloss.Color("#58a6ff"),
	AccentDim:   lipgloss.Color("#1f6feb"),

	ErrorColor:  lipgloss.Color("#f85149"),
	WarnColor:   lipgloss.Color("#d29922"),
	InfoColor:   lipgloss.Color("#9ca3af"),
	DebugColor:  lipgloss.Color("#6b7280"),

	SelectedBg:  lipgloss.Color("#1f3a5f"),
	CursorBg:    lipgloss.Color("#1c2333"),
	CursorBorder: lipgloss.Color("#58a6ff"),
	FrozenBg:    lipgloss.Color("#1c1e2a"),
	GreenColor:  lipgloss.Color("#3fb950"),
	OrangeColor: lipgloss.Color("#f0883e"),
	SearchHighlight: lipgloss.Color("#f0883e"),
}

// Current is the active theme.
var Current = Dark

// Styles holds pre-built lipgloss styles.
type Styles struct {
	TitleBar      lipgloss.Style
	StatusBar     lipgloss.Style
	Sidebar       lipgloss.Style
	SidebarHeader lipgloss.Style
	LogLine       lipgloss.Style
	ShellTabBar   lipgloss.Style
	HelpOverlay   lipgloss.Style
	HelpContent   lipgloss.Style
}

// NewStyles creates styles from the current theme.
func NewStyles() Styles {
	t := Current
	return Styles{
		TitleBar: lipgloss.NewStyle().
			Background(lipgloss.Color("#161b22")).
			Foreground(t.Foreground).
			Padding(0, 1),
		StatusBar: lipgloss.NewStyle().
			Background(lipgloss.Color("#161b22")).
			Foreground(t.Muted).
			Padding(0, 1),
		Sidebar: lipgloss.NewStyle().
			Background(t.Background).
			BorderRight(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(t.Border),
		SidebarHeader: lipgloss.NewStyle().
			Foreground(t.Muted).
			Bold(true).
			Padding(0, 1),
		LogLine: lipgloss.NewStyle().
			Foreground(t.Foreground),
		ShellTabBar: lipgloss.NewStyle().
			Background(lipgloss.Color("#161b22")).
			Foreground(t.Foreground),
		HelpOverlay: lipgloss.NewStyle().
			Background(lipgloss.Color("#161b22")).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(t.Border).
			Padding(1, 2),
		HelpContent: lipgloss.NewStyle().
			Foreground(t.Foreground),
	}
}
