package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/nilesh/docktail/internal/theme"
)

// TitleBarView renders the title bar.
func TitleBarView(width int, project string, visibleCount, totalCount int, notification string, copied bool, search SearchModel, frozen bool) string {
	t := theme.Current
	left := lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("◉ docktail")
	left += lipgloss.NewStyle().Foreground(t.Muted).Render(" │ ")
	left += lipgloss.NewStyle().Foreground(t.Muted).Render("project: ")
	left += lipgloss.NewStyle().Foreground(t.Foreground).Render(project)
	left += lipgloss.NewStyle().Foreground(t.Muted).Render(" │ ")
	left += fmt.Sprintf("containers: %d/%d", visibleCount, totalCount)

	var right string
	if notification != "" {
		right += lipgloss.NewStyle().Foreground(t.OrangeColor).Render(notification) + " "
	}
	if copied {
		right += lipgloss.NewStyle().Foreground(t.GreenColor).Render("✓ copied") + " "
	}
	if search.Active {
		mode := "/"
		if search.IsRegex {
			mode = "regex:"
		}
		right += lipgloss.NewStyle().Foreground(t.OrangeColor).Render(mode+search.Query+"▌") + " "
		if search.RegexError != "" {
			right += lipgloss.NewStyle().Foreground(t.ErrorColor).Render(search.RegexError) + " "
		}
	} else if search.Query != "" {
		label := "search"
		if search.IsRegex {
			label = "regex"
		}
		right += lipgloss.NewStyle().Foreground(t.Muted).Render(label+": "+search.Query) + " "
	}
	if frozen {
		right += lipgloss.NewStyle().Foreground(t.OrangeColor).Render("❄ FROZEN") + " "
	}

	padding := width - lipgloss.Width(left) - lipgloss.Width(right)
	if padding < 0 {
		padding = 0
	}

	return lipgloss.NewStyle().
		Width(width).
		Background(lipgloss.Color("#161b22")).
		Render(left + strings.Repeat(" ", padding) + right)
}
