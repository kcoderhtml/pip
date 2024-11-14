package styles

import "github.com/charmbracelet/lipgloss"

var Error = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("1"))

var Warn = lipgloss.NewStyle().
	Bold(true).
	Italic(true).
	Foreground(lipgloss.Color("11"))

var Info = lipgloss.NewStyle().
	Italic(true).
	Foreground(lipgloss.Color("2"))
