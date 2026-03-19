package ui

import (
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// SearchModel manages search state.
type SearchModel struct {
	Query      string
	Regex      *regexp.Regexp
	IsRegex    bool
	Active     bool
	RegexError string
}

func (m SearchModel) Update(msg tea.KeyMsg) (SearchModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.Active = false
		m.Query = ""
		m.Regex = nil
		m.RegexError = ""
		return m, func() tea.Msg { return RefilterMsg{} }
	case "enter":
		m.Active = false
	case "tab":
		m.IsRegex = !m.IsRegex
		m.updateRegex()
		return m, func() tea.Msg { return RefilterMsg{} }
	case "backspace":
		if len(m.Query) > 0 {
			m.Query = m.Query[:len(m.Query)-1]
			m.updateRegex()
			return m, func() tea.Msg { return RefilterMsg{} }
		}
	default:
		if len(msg.String()) == 1 {
			m.Query += msg.String()
			m.updateRegex()
			return m, func() tea.Msg { return RefilterMsg{} }
		}
	}
	return m, nil
}

func (m *SearchModel) updateRegex() {
	if m.IsRegex && m.Query != "" {
		re, err := regexp.Compile("(?i)" + m.Query)
		if err != nil {
			m.Regex = nil
			m.RegexError = "invalid regex"
		} else {
			m.Regex = re
			m.RegexError = ""
		}
	} else {
		m.Regex = nil
		m.RegexError = ""
	}
}

// Matches returns whether a message matches the current search.
func (m SearchModel) Matches(msg string) bool {
	if m.Query == "" {
		return true
	}
	if m.IsRegex && m.Regex != nil {
		return m.Regex.MatchString(msg)
	}
	return strings.Contains(strings.ToLower(msg), strings.ToLower(m.Query))
}

// Activate starts search mode.
func (m *SearchModel) Activate() {
	m.Active = true
	m.Query = ""
	m.RegexError = ""
}
