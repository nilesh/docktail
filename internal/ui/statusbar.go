package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/nilesh/docktail/internal/model"
	"github.com/nilesh/docktail/internal/theme"
)

// LevelFilters defines the cycle order for log level filtering.
var LevelFilters = []model.LogLevel{"", model.LevelError, model.LevelWarn, model.LevelInfo, model.LevelDebug}

// StatusBarView renders the status bar.
func StatusBarView(width int, frozen bool, shellFocused bool, shellOpen bool, sidebarFocused bool, hideStopped bool, selectedCount int, totalLines int, cursorLine int, levelFilter int) string {
	t := theme.Current
	bg := t.ChromeBg
	if frozen {
		bg = t.FrozenBg
	}
	s := func(fg lipgloss.Color, text string) string {
		return lipgloss.NewStyle().Foreground(fg).Background(bg).Render(text)
	}

	var left string
	if sidebarFocused {
		left += s(t.Accent, "⎵") + s(t.Muted, " toggle ")
		left += s(t.Accent, "↵") + s(t.Muted, " actions ")
		left += s(t.Accent, "s") + s(t.Muted, " shell ")
		left += s(t.Accent, "a") + s(t.Muted, " all ")
		if hideStopped {
			left += s(t.Accent, "h") + s(t.Muted, " show stopped ")
		} else {
			left += s(t.Accent, "h") + s(t.Muted, " hide stopped ")
		}
	} else {
		left += s(t.Accent, "f") + s(t.Muted, " freeze ")
		left += s(t.Accent, "t") + s(t.Muted, " timestamps ")
		left += s(t.Accent, "w") + s(t.Muted, " wrap ")
		left += s(t.Accent, "l") + s(t.Muted, fmt.Sprintf(" %s ", LevelFilterLabel(levelFilter)))
		left += s(t.Accent, "/") + s(t.Muted, " search ")
		if shellOpen {
			left += s(t.Accent, "x") + s(t.Muted, " close shell ")
		}
		if frozen {
			left += s(t.Muted, "│ ")
			left += s(t.Accent, "⎵") + s(t.Muted, " select ")
			left += s(t.Accent, "⇧↑↓") + s(t.Muted, " range ")
			left += s(t.Accent, "y") + s(t.Muted, " copy ")
		}
	}

	var right string
	if shellFocused {
		right += s(t.Accent, "SHELL ")
	}
	if selectedCount > 0 {
		right += s(t.OrangeColor, fmt.Sprintf("%d selected ", selectedCount))
	}
	right += s(t.Muted, fmt.Sprintf("%d lines", totalLines))
	if frozen && cursorLine >= 0 {
		right += s(t.Muted, fmt.Sprintf(" ln %d", cursorLine+1))
	}

	padding := width - lipgloss.Width(left) - lipgloss.Width(right)
	if padding < 0 {
		padding = 0
	}
	pad := lipgloss.NewStyle().Background(bg).Render(strings.Repeat(" ", padding))

	return lipgloss.NewStyle().
		Width(width).
		Background(bg).
		Render(left + pad + right)
}

// LevelFilterLabel returns the display label for a filter index.
func LevelFilterLabel(idx int) string {
	if idx == 0 {
		return "all"
	}
	return strings.ToLower(string(LevelFilters[idx]))
}
