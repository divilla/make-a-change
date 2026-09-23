package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// TruncateBlock trims each line in a block to the supplied terminal width.
func TruncateBlock(value string, width int) string {
	width = NormalizeWidth(width)
	lines := strings.Split(value, "\n")
	for i, line := range lines {
		if lipgloss.Width(line) <= width {
			continue
		}
		runes := []rune(line)
		for len(runes) > 0 && lipgloss.Width(string(runes)) > width {
			runes = runes[:len(runes)-1]
		}
		lines[i] = string(runes)
	}
	return strings.Join(lines, "\n")
}
