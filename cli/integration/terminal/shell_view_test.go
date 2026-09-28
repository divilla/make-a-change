package terminal_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShellNavigationEditorAndScrolling(t *testing.T) {
	if _, err := exec.LookPath("socat"); err != nil {
		t.Fatal("socat is required for PTY execution")
	}

	cliRoot, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	testRoot := t.TempDir()
	binPath, err := terminalBinary(cliRoot, testRoot, os.Getenv("MCH_COVER_BINARY"), os.Getenv("MCH_COVER_DIR"))
	require.NoError(t, err)

	backend := newTerminalBackend(t)
	repoRoot := filepath.Join(testRoot, "repo")
	require.NoError(t, os.MkdirAll(filepath.Join(repoRoot, ".mch", "default"), 0o755))
	require.NoError(t, exec.Command("git", "init", repoRoot).Run())
	writeTerminalFile(t, filepath.Join(repoRoot, ".mch", "config.yaml"), "backend_url: "+backend.URL+"\nproject_id: 7\n", 0o644)

	stubDir := filepath.Join(testRoot, "bin")
	require.NoError(t, os.MkdirAll(stubDir, 0o755))
	writeTerminalFile(t, filepath.Join(stubDir, "editor"), "#!/bin/sh\nprintf '# PTY Change\\n\\nInitial definition\\n' > \"$1\"\n", 0o755)

	childPIDPath := filepath.Join(testRoot, "mch.pid")
	childExitPath := filepath.Join(testRoot, "mch.exit")
	wrapper := filepath.Join(testRoot, "run-mch")
	writeTerminalFile(t, wrapper, "#!/bin/sh\nstty rows 20 cols 100\nprintf '%s\\n' \"$$\" > \"$MCH_PTY_CHILD_PID\"\n\"$MCH_PTY_BINARY\"\nresult=$?\nprintf '%s\\n' \"$result\" > \"$MCH_PTY_CHILD_EXIT\"\nexit \"$result\"\n", 0o755)
	cmd := exec.Command("socat", "EXEC:"+wrapper+",pty,setsid,ctty,stderr", "STDIO")
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		"EDITOR="+filepath.Join(stubDir, "editor"),
		"PATH="+stubDir+":"+os.Getenv("PATH"),
		"MCH_PTY_BINARY="+binPath,
		"MCH_PTY_CHILD_PID="+childPIDPath,
		"MCH_PTY_CHILD_EXIT="+childExitPath,
		"GOCOVERDIR="+os.Getenv("MCH_COVER_DIR"),
	)
	stdin, err := cmd.StdinPipe()
	require.NoError(t, err)
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	require.NoError(t, cmd.Start())

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	finished := false
	t.Cleanup(func() {
		_ = stdin.Close()
		if !finished {
			if err := cleanupTerminal(cmd, done, childPIDPath, 3*time.Second); err != nil {
				t.Error(err)
			}
		}
	})
	capture := newTerminalCapture(stdout)
	_, err = stdin.Write([]byte("\x1b]11;rgb:0000/0000/0000\x1b\\\x1b[1;1R"))
	require.NoError(t, err)

	require.NoError(t, capture.waitFor("MainScreen", 5*time.Second))

	send := func(keys, marker string) {
		t.Helper()
		offset := capture.len()
		_, err := io.WriteString(stdin, keys)
		require.NoError(t, err)
		require.NoError(t, capture.waitForAfter(marker, offset, 5*time.Second))
	}
	send("/projects\r", "ProjectsListScreen")
	send("/new-project\r", "ProjectCreateScreen")
	send("/editor\r", "ProjectDetailsScreen")
	assert.Contains(t, capture.after(0), "\x1b[2J", "editor restoration redraws screen")
	send("/return\r", "ProjectsListScreen")
	send("/return\r", "MainScreen")
	send("/changes\r", "Rows 1-8 of 30")
	assert.Contains(t, capture.after(0), "\x1b[", "terminal output retains styles")
	send("\x1b[6~", "Rows 2-9 of 30")
	send("\x1b[5~", "Rows 1-8 of 30")
	send("/", "Commands")
	send("\x1b", "Type / for commands")
	send("/return\r", "MainScreen")
	assert.NoDirExists(t, filepath.Join(repoRoot, ".mch/tmp"))
	_, err = io.WriteString(stdin, "/quit\r")
	require.NoError(t, err)
	require.NoError(t, waitTerminal(done, 5*time.Second))
	finished = true
	childExit, err := os.ReadFile(childExitPath)
	require.NoError(t, err)
	require.Equal(t, "0", strings.TrimSpace(string(childExit)), "application child exit")
	if directory := os.Getenv("MCH_COVER_DIR"); directory != "" {
		counters, err := filepath.Glob(filepath.Join(directory, "covcounters.*"))
		require.NoError(t, err)
		require.NotEmpty(t, counters, "orderly child exit must flush coverage")
	}
}

func newTerminalBackend(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var value any
		switch r.URL.Path {
		case "/api/v1/options/change-phases-list":
			value = []map[string]any{{"slug": "backlog", "color": "12"}}
		case "/api/v1/options/change-types-list":
			value = []any{}
		case "/api/v1/project/get", "/api/v1/project/create":
			value = map[string]any{"id": 7, "name": "PTY Project"}
		case "/api/v1/project/list":
			value = []map[string]any{{"id": 7, "name": "PTY Project"}}
		case "/api/v1/change/list":
			rows := []map[string]any{}
			for i := 1; i <= 30; i++ {
				rows = append(rows, map[string]any{"id": i, "title": fmt.Sprintf("Row %02d", i), "change_phase": "backlog"})
			}
			value = rows
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(value)
	}))
	t.Cleanup(server.Close)
	return server
}

func writeTerminalFile(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(content), mode))
}

type terminalCapture struct {
	mu      sync.Mutex
	content bytes.Buffer
	updated chan struct{}
}

func newTerminalCapture(reader io.Reader) *terminalCapture {
	capture := &terminalCapture{updated: make(chan struct{}, 1)}
	go func() {
		buffer := make([]byte, 4096)
		for {
			n, err := reader.Read(buffer)
			if n > 0 {
				capture.mu.Lock()
				_, _ = capture.content.Write(buffer[:n])
				capture.mu.Unlock()
				select {
				case capture.updated <- struct{}{}:
				default:
				}
			}
			if err != nil {
				return
			}
		}
	}()
	return capture
}

func (c *terminalCapture) len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.content.Len()
}

func (c *terminalCapture) after(offset int) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	content := c.content.String()
	if offset > len(content) {
		return ""
	}
	return content[offset:]
}

func (c *terminalCapture) waitFor(marker string, timeout time.Duration) error {
	return c.waitForAfter(marker, 0, timeout)
}

func (c *terminalCapture) waitForAfter(marker string, offset int, timeout time.Duration) error {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		if strings.Contains(c.after(offset), marker) {
			return nil
		}
		select {
		case <-c.updated:
		case <-timer.C:
			return fmt.Errorf("timed out waiting for %q; terminal output: %q", marker, c.after(offset))
		}
	}
}
