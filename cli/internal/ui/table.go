package ui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// TruncateBlock trims each line in a block to the supplied terminal width.
func TruncateBlock(value string, width int) string {
	width = NormalizeWidth(width)
	lines := strings.Split(value, "\n")
	for i, line := range lines {
		lines[i] = ansi.Truncate(line, width, "")
	}
	return strings.Join(lines, "\n")
}
