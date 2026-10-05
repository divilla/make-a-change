package documents

import (
	"cli/internal/styles"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"
)

func Test031HistoryFiltersTerminalControlsPreservingSGR(t *testing.T) {
	for _, tc := range []struct {
		name, input, want string
	}{
		{"clipboard BEL", "a\x1b]52;c;YXR0YWNr\ab", "ab"},
		{"clipboard ST", "a\x1b]52;c;YXR0YWNr\x1b\\b", "ab"},
		{"OSC hyperlink", "\x1b]8;;https://example.com\x1b\\link\x1b]8;;\x1b\\", "link"},
		{"CSI display cursor mode", "a\x1b[2J\x1b[H\x1b[?25lb", "ab"},
		{"ESC controls", "a\x1b7\x1b8\x1bcb", "ab"},
		{"DCS SOS PM APC", "a\x1bP1;2qpayload\x1b\\\x1bXpayload\x1b\\\x1b^payload\x1b\\\x1b_payload\x1b\\b", "ab"},
		{"raw C1 sequences", "a\x9d52;c;YXR0YWNr\x9c\x9b2J\x90qpayload\x9cb", "ab"},
		{"C0 DEL Unicode C1", "a\x00\a\b\r\v\f\x7f\u0085\u009bb", "ab"},
		{"SGR with private prefix", "a\x1b[?31mb", "ab"},
		{"SGR with intermediate", "a\x1b[31 mb", "ab"},
		{"incomplete OSC", "a\x1b]52;c;payload", "a"},
		{"incomplete CSI", "a\x1b[31", "a"},
		{"incomplete ESC", "a\x1b", "a"},
		{"malformed CSI", "a\x1b[\a\x1b[2Jb", "ab"},
		{"invalid UTF8", "a\xff\xc0b", "ab"},
		{"Unicode and tab", "\u0301漢字 e\u0301 👩‍💻\tend", "\u0301漢字 e\u0301 👩‍💻    end"},
		{"SGR colors and attributes", "\x1b[1;38;2;11;22;33mcolored\x1b[38:2::44:55:66mtext\x1b[m", "\x1b[1;38;2;11;22;33mcolored\x1b[38:2::44:55:66mtext\x1b[m"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := History{Output: tc.input}
			require.Equal(t, tc.want+ansi.ResetStyle, h.View(100, 1))
			require.Equal(t, tc.input, h.Output, "rendering leaves captured output intact")
		})
	}
}

func Test031HistoryExpandsTabsBeforeClipping(t *testing.T) {
	color := "\x1b[38;2;11;22;33m"
	for _, tc := range []struct {
		name, input, want string
		width             int
	}{
		{"reported overflow", "123456789012345\t12345", "123456789012345    1", 20},
		{"leading repeated tabs", "\t\tabc", "        ab", 10},
		{"clip inside tab", "abc\tdef", "abc  ", 5},
		{"wide and combining text", "漢e\u0301\tend", "漢e\u0301    e", 8},
		{"color across tab", "abc\t\x1b[1mdef", "abc    \x1b[1md", 8},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := History{Output: color + "first\n" + tc.input + "\nlast"}
			output := h.Output
			h = h.Scroll(1, 1)
			view := h.View(tc.width, 1)
			require.Equal(t, color+tc.want+ansi.ResetStyle, view)
			require.NotContains(t, view, "\t")
			require.LessOrEqual(t, ansi.StringWidth(view), tc.width)
			require.Equal(t, 1, lipgloss.Height(styles.Default.Surface.Width(tc.width).Render(view)))
			require.Equal(t, output, h.Output, "tab expansion only changes display text")
		})
	}
}

func Test031HistoryFiltersBeforeScrollingAndClipping(t *testing.T) {
	color := "\x1b[38;2;11;22;33m"
	h := History{Output: color + "first\x1b]52;c;hidden\nlines\ninside\x1b\\\nsecond\x1b[2J\nthird\x1b[0m\n"}
	h = h.Scroll(999, 1)
	require.Equal(t, 2, h.Offset, "OSC payload lines do not count as body lines")
	require.Equal(t, color+"thi\x1b[0m"+ansi.ResetStyle, h.View(3, 1))
	h = h.Scroll(-1, 1)
	require.Equal(t, color+"second"+ansi.ResetStyle, h.View(20, 1))
	h = h.Scroll(-999, 1)
	view := h.View(3, 3)
	require.NotContains(t, view, "\x1b]52")
	require.NotContains(t, view, "\x1b[2J")
	require.NotContains(t, view, "hidden")
	for _, line := range strings.Split(view, "\n") {
		require.Contains(t, line, color)
		require.LessOrEqual(t, ansi.StringWidth(line), 3)
	}
}
