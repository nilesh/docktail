package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/nilesh/docktail/internal/theme"
)

// HelpModel manages the help overlay.
type HelpModel struct {
	Visible bool
}

// Toggle toggles help visibility.
func (m *HelpModel) Toggle() {
	m.Visible = !m.Visible
}

// HandleKey processes a key in help mode. Returns true if the key was consumed.
func (m *HelpModel) HandleKey(keyStr string) bool {
	if keyStr == "?" || keyStr == "esc" {
		m.Visible = false
		return true
	}
	return true // consume all keys when help is open
}

// View renders the help overlay.
func (m HelpModel) View(width, height int) string {
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
			{"T", "Toggle theme"},
			{"b", "Toggle sidebar"},
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
		{"Mouse", [][2]string{
			{"Click", "Freeze & move cursor"},
			{"Drag", "Select line range"},
			{"Shift+click", "Range select"},
			{"Ctrl+click", "Toggle select"},
			{"Double-click", "Copy line"},
			{"Scroll", "Scroll logs"},
			{"Right-click", "Container actions"},
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
		Background(t.ChromeBg).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Border).
		Padding(1, 2).
		Width(50).
		Render(content)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box,
		lipgloss.WithWhitespaceBackground(t.OverlayBg))
}
