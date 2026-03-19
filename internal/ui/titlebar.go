package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/nilesh/docktail/internal/theme"
)

// TitleBarView renders the title bar.
func TitleBarView(width int, project string, visibleCount, totalCount int, notification string, copied bool, search SearchModel, frozen bool, wrapLines bool, showTimestamps bool, filterLevel int) string {
	t := theme.Current
	bg := t.TitleBg
	s := func(fg lipgloss.Color, text string) string {
		return lipgloss.NewStyle().Foreground(fg).Background(bg).Render(text)
	}
	sb := func(fg lipgloss.Color, text string) string {
		return lipgloss.NewStyle().Foreground(fg).Background(bg).Bold(true).Render(text)
	}

	left := sb(t.TitleFg, "◉ docktail")
	left += s(t.TitleDim, " │ ")
	left += s(t.TitleDim, "project: ")
	left += s(t.TitleFg, project)
	left += s(t.TitleDim, " │ ")
	left += s(t.TitleFg, fmt.Sprintf("containers: %d/%d", visibleCount, totalCount))

	var right string
	if notification != "" {
		right += s(t.OrangeColor, notification) + s(t.TitleDim, " ")
	}
	if copied {
		right += s(t.GreenColor, "✓ copied") + s(t.TitleDim, " ")
	}
	if search.Active {
		mode := "/"
		if search.IsRegex {
			mode = "regex:"
		}
		right += s(t.TitleFg, mode+search.Query+"▌") + s(t.TitleDim, " ")
		if search.RegexError != "" {
			right += s(t.ErrorColor, search.RegexError) + s(t.TitleDim, " ")
		}
	} else if search.Query != "" {
		label := "search"
		if search.IsRegex {
			label = "regex"
		}
		right += s(t.TitleDim, label+": "+search.Query) + s(t.TitleDim, " ")
	}
	// Filter level badge
	if filterLevel > 0 {
		levelName := strings.ToUpper(string(LevelFilters[filterLevel]))
		right += s(t.TitleFg, levelName) + s(t.TitleDim, " ")
	}
	if frozen {
		right += sb(t.TitleFg, "❄ FROZEN") + s(t.TitleDim, " ")
	}
	if wrapLines {
		right += s(t.TitleDim, "wrap:on ")
	}
	if showTimestamps {
		right += s(t.TitleDim, "ts:on ")
	}
	right += s(t.TitleDim, "? help")

	padding := width - lipgloss.Width(left) - lipgloss.Width(right)
	if padding < 0 {
		padding = 0
	}
	pad := lipgloss.NewStyle().Background(bg).Render(strings.Repeat(" ", padding))

	return lipgloss.NewStyle().
		Width(width).
		Background(t.TitleBg).
		Render(left + pad + right)
}
