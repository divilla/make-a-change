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
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRewriteScreenUsesColoredBlackScrollableViewport(t *testing.T) {
	if _, err := exec.LookPath("socat"); err != nil {
		t.Skip("socat is required for PTY coverage")
	}

	cliRoot, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	testRoot := t.TempDir()
	binPath := filepath.Join(testRoot, "mch")
	build := exec.Command("go", "build", "-o", binPath, "./cmd/mch")
	build.Dir = cliRoot
	build.Env = append(os.Environ(), "GOCACHE=/tmp/project-manager-cli-go-build")
	require.NoError(t, build.Run())

	backend := newTerminalBackend(t)
	repoRoot := filepath.Join(testRoot, "repo")
	require.NoError(t, os.MkdirAll(filepath.Join(repoRoot, ".mch", "default"), 0o755))
	require.NoError(t, exec.Command("git", "init", repoRoot).Run())
	writeTerminalFile(t, filepath.Join(repoRoot, ".mch", "config.yaml"), "backend_url: "+backend.URL+"\nproject_id: 7\n", 0o644)
	writeTerminalFile(t, filepath.Join(repoRoot, ".mch", "default", "flow.yaml"), "version: 1\nslug: default\nhelp: help.yaml\nmakefile: Makefile\nsteps:\n  - slug: def-write\n    type: edit\n", 0o644)
	writeTerminalFile(t, filepath.Join(repoRoot, ".mch", "default", "help.yaml"), "version: 1\nstage_modes: []\ntask_statuses: []\ntask_steps: []\n", 0o644)

	stubDir := filepath.Join(testRoot, "bin")
	require.NoError(t, os.MkdirAll(stubDir, 0o755))
	writeTerminalFile(t, filepath.Join(stubDir, "editor"), "#!/bin/sh\nprintf '# PTY Change\\n\\nInitial definition\\n' > \"$1\"\n", 0o755)
	codexPIDPath := filepath.Join(testRoot, "codex.pid")
	codexReleasePath := filepath.Join(testRoot, "codex.release")
	writeTerminalFile(t, filepath.Join(stubDir, "codex"), terminalCodexStub(), 0o755)

	wrapper := filepath.Join(testRoot, "run-mch")
	writeTerminalFile(t, wrapper, "#!/bin/sh\nstty rows 20 cols 100\nexec \"$MCH_PTY_BINARY\"\n", 0o755)
	cmd := exec.Command("socat", "EXEC:"+wrapper+",pty,setsid,ctty,stderr", "STDIO")
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		"EDITOR="+filepath.Join(stubDir, "editor"),
		"PATH="+stubDir+":"+os.Getenv("PATH"),
		"MCH_PTY_BINARY="+binPath,
		"MCH_PTY_CODEX_PID="+codexPIDPath,
		"MCH_PTY_CODEX_RELEASE="+codexReleasePath,
	)
	stdin, err := cmd.StdinPipe()
	require.NoError(t, err)
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	require.NoError(t, cmd.Start())

	capture := newTerminalCapture(stdout)
	_, err = stdin.Write([]byte("\x1b]11;rgb:0000/0000/0000\x1b\\\x1b[1;1R"))
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = cmd.Process.Signal(syscall.SIGTERM)
		_, _ = cmd.Process.Wait()
		if content, readErr := os.ReadFile(codexPIDPath); readErr == nil {
			_ = exec.Command("kill", strings.TrimSpace(string(content))).Run()
		}
	})

	require.NoError(t, capture.waitFor("MainScreen", 5*time.Second))
	_, err = io.WriteString(stdin, "/changes\r")
	require.NoError(t, err)
	require.NoError(t, capture.waitFor("ChangesListScreen", 5*time.Second))
	_, err = stdin.Write([]byte{0x0e}) // Ctrl+N
	require.NoError(t, err)
	require.NoError(t, capture.waitFor("Create Change?", 5*time.Second))
	_, err = stdin.Write([]byte{'\r'})
	require.NoError(t, err)
	require.NoError(t, capture.waitFor("assistant: output line 75", 8*time.Second))
	runningOutput := capture.after(0)
	assert.Contains(t, runningOutput, "Type / for commands")
	assert.Contains(t, runningOutput, "def-write - 33333333-3333-3333-3333-333333333333 - started - 00:")

	_, err = stdin.Write([]byte{'/'})
	require.NoError(t, err)
	require.NoError(t, capture.waitFor("Commands: no options", 5*time.Second))
	beforeEscape := capture.len()
	_, err = stdin.Write([]byte{'\x1b'})
	require.NoError(t, err)
	require.NoError(t, capture.waitForAfter("Type / for commands", beforeEscape, 5*time.Second))
	beforeHome := capture.len()
	_, err = stdin.Write([]byte("\x1b[H"))
	require.NoError(t, err)
	require.NoError(t, capture.waitForAfter("Codex output:", beforeHome, 5*time.Second))
	homeFrame := capture.after(beforeHome)
	assert.Contains(t, homeFrame, "assistant: output line 01")
	assert.Contains(t, homeFrame, ";40m")
	assert.Contains(t, homeFrame, "\x1b[38;5;183m")

	beforePage := capture.len()
	_, err = stdin.Write([]byte("\x1b[6~"))
	require.NoError(t, err)
	require.NoError(t, capture.waitForAfter("assistant: output line 10", beforePage, 5*time.Second))
	assert.Contains(t, capture.after(0), "<pgup/pgdown> page")

	beforeCompletion := capture.len()
	writeTerminalFile(t, codexReleasePath, "release\n", 0o644)
	require.NoError(t, capture.waitForAfter("ChangeDetailsScreen", beforeCompletion, 5*time.Second))
	completionFrame := capture.after(beforeCompletion)
	assert.Contains(t, completionFrame, "\x1b[2J")
	assert.Contains(t, completionFrame, "status save")

	beforeCommand := capture.len()
	_, err = stdin.Write([]byte{'/'})
	require.NoError(t, err)
	require.NoError(t, capture.waitForAfter("Commands", beforeCommand, 5*time.Second))
}

func terminalCodexStub() string {
	return `#!/bin/sh
output=""
previous=""
for argument in "$@"; do
  if [ "$previous" = "-o" ]; then output="$argument"; fi
  previous="$argument"
done
printf '%s\n' "$$" > "$MCH_PTY_CODEX_PID"
printf 'Done.' > "$output"
printf '{"type":"thread.started","thread_id":"33333333-3333-3333-3333-333333333333"}\n'
index=1
while [ "$index" -le 75 ]; do
  printf '{"type":"item.completed","item":{"type":"agent_message","text":"output line %02d"}}\n' "$index"
  index=$((index + 1))
done
while [ ! -f "$MCH_PTY_CODEX_RELEASE" ]; do sleep 0.05; done
`
}

func newTerminalBackend(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)
		switch r.URL.Path {
		case "/api/v1/options/change-phases-list":
			_ = json.NewEncoder(w).Encode([]any{})
		case "/api/v1/options/change-types-list":
			_ = json.NewEncoder(w).Encode([]map[string]any{{"slug": "feature"}})
		case "/api/v1/project/get":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 7, "name": "PTY Project"})
		case "/api/v1/change/list", "/api/v1/epic/list":
			_ = json.NewEncoder(w).Encode([]any{})
		case "/api/v1/change/create":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": 12, "project_id": 7, "ref_uuid": payload["ref_uuid"],
				"title": payload["title"], "def": payload["def"],
			})
		case "/api/v1/change/update-def":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": 12, "project_id": 7, "ref_uuid": "0198a86f-9b8a-7d89-ae5b-6f25b528b04c",
				"title": "PTY Change", "def": payload["def"], "agent_edit": true,
			})
		case "/api/v1/change/get":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"change": map[string]any{
					"id": 12, "project_id": 7, "ref_uuid": "0198a86f-9b8a-7d89-ae5b-6f25b528b04c",
					"title": "PTY Change", "def": "# PTY Change\n\nInitial definition\n", "agent_edit": true,
				},
				"test_cases": []any{},
			})
		default:
			http.Error(w, fmt.Sprintf("unexpected path %s", r.URL.Path), http.StatusNotFound)
		}
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
