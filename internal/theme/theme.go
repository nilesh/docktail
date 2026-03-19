package theme

import "github.com/charmbracelet/lipgloss"

// Theme holds all style definitions for the TUI.
type Theme struct {
	// Base colors
	Background lipgloss.Color
	Foreground lipgloss.Color
	Muted      lipgloss.Color
	Border     lipgloss.Color
	Accent     lipgloss.Color
	AccentDim  lipgloss.Color

	// Chrome colors (title bar, status bar, shell tab, etc.)
	TitleBg   lipgloss.Color // title bar brand background
	TitleFg   lipgloss.Color // title bar text color
	TitleDim  lipgloss.Color // title bar muted text
	ChromeBg  lipgloss.Color
	SidebarBg lipgloss.Color // sidebar background
	FocusBg   lipgloss.Color // focused sidebar item bg
	OverlayBg lipgloss.Color // help overlay backdrop

	// Log level colors
	ErrorColor lipgloss.Color
	WarnColor  lipgloss.Color
	InfoColor  lipgloss.Color
	DebugColor lipgloss.Color

	// UI state colors
	SelectedBg      lipgloss.Color
	CursorBg        lipgloss.Color
	CursorBorder    lipgloss.Color
	FrozenBg        lipgloss.Color
	GreenColor      lipgloss.Color
	OrangeColor     lipgloss.Color
	SearchHighlight lipgloss.Color
}

// Dark is the default dark theme.
var Dark = Theme{
	Background: lipgloss.Color("#141921"),
	Foreground: lipgloss.Color("#c9d1d9"),
	Muted:      lipgloss.Color("#484f58"),
	Border:     lipgloss.Color("#30363d"),
	Accent:     lipgloss.Color("#58a6ff"),
	AccentDim:  lipgloss.Color("#1f6feb"),

	TitleBg:   lipgloss.Color("#3d2e00"),
	TitleFg:   lipgloss.Color("#f5d67b"),
	TitleDim:  lipgloss.Color("#9a8548"),
	ChromeBg:  lipgloss.Color("#1a2028"),
	SidebarBg: lipgloss.Color("#111820"),
	FocusBg:   lipgloss.Color("#1f2937"),
	OverlayBg: lipgloss.Color("#000000"),

	ErrorColor: lipgloss.Color("#f85149"),
	WarnColor:  lipgloss.Color("#d29922"),
	InfoColor:  lipgloss.Color("#9ca3af"),
	DebugColor: lipgloss.Color("#6b7280"),

	SelectedBg:      lipgloss.Color("#1f3a5f"),
	CursorBg:        lipgloss.Color("#1c2333"),
	CursorBorder:    lipgloss.Color("#58a6ff"),
	FrozenBg:        lipgloss.Color("#1c1e2a"),
	GreenColor:      lipgloss.Color("#3fb950"),
	OrangeColor:     lipgloss.Color("#f0883e"),
	SearchHighlight: lipgloss.Color("#f0883e"),
}

// Light is a theme for light terminal backgrounds.
var Light = Theme{
	Background: lipgloss.Color("#ffffff"),
	Foreground: lipgloss.Color("#1f2328"),
	Muted:      lipgloss.Color("#656d76"),
	Border:     lipgloss.Color("#d0d7de"),
	Accent:     lipgloss.Color("#0969da"),
	AccentDim:  lipgloss.Color("#218bff"),

	TitleBg:   lipgloss.Color("#fef3c7"),
	TitleFg:   lipgloss.Color("#713f12"),
	TitleDim:  lipgloss.Color("#a16207"),
	ChromeBg:  lipgloss.Color("#e8ecf0"),
	SidebarBg: lipgloss.Color("#f0f3f6"),
	FocusBg:   lipgloss.Color("#ddf4ff"),
	OverlayBg: lipgloss.Color("#e0e0e0"),

	ErrorColor: lipgloss.Color("#cf222e"),
	WarnColor:  lipgloss.Color("#9a6700"),
	InfoColor:  lipgloss.Color("#57606a"),
	DebugColor: lipgloss.Color("#8b949e"),

	SelectedBg:      lipgloss.Color("#b6d7ff"),
	CursorBg:        lipgloss.Color("#ddf4ff"),
	CursorBorder:    lipgloss.Color("#0969da"),
	FrozenBg:        lipgloss.Color("#eef6ff"),
	GreenColor:      lipgloss.Color("#1a7f37"),
	OrangeColor:     lipgloss.Color("#bc4c00"),
	SearchHighlight: lipgloss.Color("#bc4c00"),
}

// Current is the active theme.
var Current = Dark

// SetTheme activates a theme by name. Valid values: "dark", "light", "auto".
// "auto" attempts to detect the terminal background.
func SetTheme(name string) {
	switch name {
	case "light":
		Current = Light
	case "dark":
		Current = Dark
	case "auto":
		Current = detectTheme()
	default:
		Current = Dark
	}
}

// detectTheme uses lipgloss's HasDarkBackground to pick the right theme.
func detectTheme() Theme {
	if lipgloss.HasDarkBackground() {
		return Dark
	}
	return Light
}

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
			Background(t.TitleBg).
			Foreground(t.TitleFg).
			Padding(0, 1),
		StatusBar: lipgloss.NewStyle().
			Background(t.ChromeBg).
			Foreground(t.Muted).
			Padding(0, 1),
		Sidebar: lipgloss.NewStyle().
			Background(t.SidebarBg).
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
			Background(t.ChromeBg).
			Foreground(t.Foreground),
		HelpOverlay: lipgloss.NewStyle().
			Background(t.ChromeBg).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(t.Border).
			Padding(1, 2),
		HelpContent: lipgloss.NewStyle().
			Foreground(t.Foreground),
	}
}
