package styles

import "github.com/charmbracelet/lipgloss"

// Tokens groups shared Lip Gloss styles used by mch views.
type Tokens struct {
	Background   lipgloss.Style
	Surface      lipgloss.Style
	Foreground   lipgloss.Style
	Muted        lipgloss.Style
	InputBand    lipgloss.Style
	Selection    lipgloss.Style
	Error        lipgloss.Style
	Success      lipgloss.Style
	AccentCyan   lipgloss.Style
	AccentPurple lipgloss.Style
	Border       lipgloss.Style
	Title        lipgloss.Style
	Footer       lipgloss.Style
}

// Default contains the standard mch style tokens.
var Default = Tokens{
	Background: lipgloss.NewStyle().
		Background(lipgloss.Color("235")),
	Surface: lipgloss.NewStyle(),
	Foreground: lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")),
	Muted: lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")),
	InputBand: lipgloss.NewStyle().
		Background(lipgloss.Color("240")).
		Foreground(lipgloss.Color("252")),
	Selection: lipgloss.NewStyle().
		Background(lipgloss.Color("60")).
		Foreground(lipgloss.Color("15")),
	Error: lipgloss.NewStyle().
		Foreground(lipgloss.Color("203")),
	Success: lipgloss.NewStyle().
		Foreground(lipgloss.Color("114")),
	AccentCyan: lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")),
	AccentPurple: lipgloss.NewStyle().
		Foreground(lipgloss.Color("183")),
	Border: lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")),
	Title: lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")),
	Footer: lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")),
}
