package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
)

func TestTruncateBlockTerminalCells(t *testing.T) {
	for _, tc := range []struct {
		name, input, want string
		width             int
	}{
		{"short", "unchanged\n", "unchanged\n", 20},
		{"ascii", strings.Repeat("a", 50000), strings.Repeat("a", 20), 20},
		{"wide", strings.Repeat("界", 20), strings.Repeat("界", 10), 21},
		{"combining", strings.Repeat("e\u0301", 30), strings.Repeat("e\u0301", 20), 20},
		{"emoji", strings.Repeat("👩‍💻", 30), strings.Repeat("👩‍💻", 10), 20},
		{"normalized", strings.Repeat("x", 120), strings.Repeat("x", 100), 0},
		{"ansi", "\x1b[31m" + strings.Repeat("x", 40) + "\x1b[0m", "\x1b[31m" + strings.Repeat("x", 20) + "\x1b[0m", 20},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := TruncateBlock(tc.input, tc.width)
			assert.Equal(t, tc.want, got)
			for _, line := range strings.Split(got, "\n") {
				assert.LessOrEqual(t, ansi.StringWidth(line), NormalizeWidth(tc.width))
			}
		})
	}
}
