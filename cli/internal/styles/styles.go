// Package styles defines colors and styles for the terminal application.
package styles

import "github.com/charmbracelet/lipgloss"

// Dark theme colors; InputBackground and MenuBackground match the menu screenshot.
const (
	Background              lipgloss.Color = "#000000"
	Foreground              lipgloss.Color = "#FFFFFF"
	LightBlue               lipgloss.Color = "#AFD7D7"
	AccentBlue              lipgloss.Color = "#87AFFF"
	AccentPurple            lipgloss.Color = "#D7AFFF"
	AccentCyan              lipgloss.Color = "#87D7D7"
	AccentGreen             lipgloss.Color = "#D7FFD7"
	AccentYellow            lipgloss.Color = "#FFFFAF"
	AccentRed               lipgloss.Color = "#FF87AF"
	DiffAdded               lipgloss.Color = "#005F00"
	DiffRemoved             lipgloss.Color = "#5F0000"
	Comment                 lipgloss.Color = "#AFAFAF"
	Gray                    lipgloss.Color = "#AFAFAF"
	DarkGray                lipgloss.Color = "#878787"
	MutedPurple             lipgloss.Color = "#5F5F87"
	MutedRed                lipgloss.Color = "#875F5F"
	MutedGreen              lipgloss.Color = "#5F875F"
	InputBackground         lipgloss.Color = "#454748"
	MessageBackground       lipgloss.Color = "#5F5F5F"
	FocusBackground         lipgloss.Color = "#005F00"
	MenuBackground          lipgloss.Color = "#47514A"
	InputBackgroundFallback lipgloss.Color = "#2A2A2A"
	FocusBackgroundFallback lipgloss.Color = "#2B332B"
	GradientStart           lipgloss.Color = "#4796E4"
	GradientMiddle          lipgloss.Color = "#847ACE"
	GradientEnd             lipgloss.Color = "#C3677F"
)

// Tokens groups shared Lip Gloss styles used by mch views.
type Tokens struct {
	Background          lipgloss.Style
	Surface             lipgloss.Style
	Foreground          lipgloss.Style
	Muted               lipgloss.Style
	InputBand           lipgloss.Style
	Selection           lipgloss.Style
	Error               lipgloss.Style
	Success             lipgloss.Style
	AccentCyan          lipgloss.Style
	AccentPurple        lipgloss.Style
	Border              lipgloss.Style
	Title               lipgloss.Style
	Footer              lipgloss.Style
	MenuPrompt          lipgloss.Style
	MenuPromptIndicator lipgloss.Style
	PromptEdge          lipgloss.Style
	MenuItem            lipgloss.Style
	MenuSelected        lipgloss.Style
	MenuCounter         lipgloss.Style
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
		Background(InputBackground).
		Foreground(Foreground),
	Selection: lipgloss.NewStyle().
		Background(MutedPurple).
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
	MenuPrompt: lipgloss.NewStyle().
		Background(InputBackground).
		Foreground(AccentGreen),
	MenuPromptIndicator: lipgloss.NewStyle().
		Background(InputBackground).
		Foreground(AccentPurple),
	PromptEdge: lipgloss.NewStyle().
		Foreground(InputBackground),
	MenuItem: lipgloss.NewStyle().
		Foreground(Gray),
	MenuSelected: lipgloss.NewStyle().
		Background(MenuBackground).
		Foreground(AccentGreen),
	MenuCounter: lipgloss.NewStyle().
		Foreground(DarkGray),
}
