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
func StatusBarView(width int, frozen bool, shellFocused bool, shellOpen bool, selectedCount int, totalLines int, cursorLine int, levelFilter int) string {
	t := theme.Current

	var left string
	left += lipgloss.NewStyle().Foreground(t.Accent).Render("f") + lipgloss.NewStyle().Foreground(t.Muted).Render(" freeze ")
	left += lipgloss.NewStyle().Foreground(t.Accent).Render("t") + lipgloss.NewStyle().Foreground(t.Muted).Render(" timestamps ")
	left += lipgloss.NewStyle().Foreground(t.Accent).Render("w") + lipgloss.NewStyle().Foreground(t.Muted).Render(" wrap ")
	left += lipgloss.NewStyle().Foreground(t.Accent).Render("l") + lipgloss.NewStyle().Foreground(t.Muted).Render(fmt.Sprintf(" %s ", LevelFilterLabel(levelFilter)))
	left += lipgloss.NewStyle().Foreground(t.Accent).Render("/") + lipgloss.NewStyle().Foreground(t.Muted).Render(" search ")
	if shellOpen {
		left += lipgloss.NewStyle().Foreground(t.Accent).Render("x") + lipgloss.NewStyle().Foreground(t.Muted).Render(" close shell ")
	}
	if frozen {
		left += lipgloss.NewStyle().Foreground(t.Muted).Render("│ ")
		left += lipgloss.NewStyle().Foreground(t.Accent).Render("⎵") + lipgloss.NewStyle().Foreground(t.Muted).Render(" select ")
		left += lipgloss.NewStyle().Foreground(t.Accent).Render("⇧↑↓") + lipgloss.NewStyle().Foreground(t.Muted).Render(" range ")
		left += lipgloss.NewStyle().Foreground(t.Accent).Render("y") + lipgloss.NewStyle().Foreground(t.Muted).Render(" copy ")
	}

	var right string
	if shellFocused {
		right += lipgloss.NewStyle().Foreground(t.Accent).Render("SHELL ")
	}
	if selectedCount > 0 {
		right += lipgloss.NewStyle().Foreground(t.OrangeColor).Render(fmt.Sprintf("%d selected ", selectedCount))
	}
	right += lipgloss.NewStyle().Foreground(t.Muted).Render(fmt.Sprintf("%d lines", totalLines))
	if frozen && cursorLine >= 0 {
		right += lipgloss.NewStyle().Foreground(t.Muted).Render(fmt.Sprintf(" ln %d", cursorLine+1))
	}

	padding := width - lipgloss.Width(left) - lipgloss.Width(right)
	if padding < 0 {
		padding = 0
	}

	return lipgloss.NewStyle().
		Width(width).
		Background(func() lipgloss.Color {
			if frozen {
				return t.FrozenBg
			}
			return t.ChromeBg
		}()).
		Render(left + strings.Repeat(" ", padding) + right)
}

// LevelFilterLabel returns the display label for a filter index.
func LevelFilterLabel(idx int) string {
	if idx == 0 {
		return "all"
	}
	return strings.ToLower(string(LevelFilters[idx]))
}
