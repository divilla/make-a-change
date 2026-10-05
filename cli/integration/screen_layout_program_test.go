package integration_test

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"
)

func TestCLIProgramSelectedItemViewer(t *testing.T) {
	b, server := docs031Server(t)
	root := t.TempDir()
	writeProgramConfig(t, root, server.URL)
	s := startProgram(t, root, "unused")
	s.navigate(t, "/changes\r", "Rows")
	s.navigate(t, "\r", "loaded change")
	s.navigate(t, strings.Repeat("\x1b[B", 6)+" ", "ItemViewScreen")
	s.waitFor(t, "status view brief")
	s.navigate(t, "\x1b[6~", "full brief tail") // bat receives the full body, not a detail preview.
	s.navigate(t, "\x1b", "returned from view")
	s.navigate(t, "\x1b[B ", "Selected active") // current spec, not newest retained version.
	s.waitFor(t, "ItemViewScreen")
	s.navigate(t, "\x03", "returned from view")
	s.navigate(t, "h", "Newest retained")
	s.navigate(t, "\x1b", "returned from history")
	s.navigate(t, "\x1b[B\x1b[B ", "full comment tail")
	s.navigate(t, "\x03", "returned from view")
	s.finishFromDetails(t)
	output := s.output.String()
	updated := time.Date(2026, 9, 28, 11, 0, 0, 0, time.UTC).Local().Format("2006-01-02 15:04")
	require.Contains(t, ansi.Strip(output), updated+" - one two three full comment tail")
	require.Contains(t, ansi.Strip(output), updated+" - [✓] brief")
	for _, mode := range []string{"\x1b[?1000h", "\x1b[?1002h", "\x1b[?1003h", "\x1b[?1006h"} {
		require.NotContains(t, output, mode)
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	require.Empty(t, b.writes, "view and history browsing must not mutate data")
}
