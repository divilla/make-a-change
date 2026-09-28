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
	writeTerminalFile(t, filepath.Join(stubDir, "editor"), "#!/bin/sh\nprintf '# PTY Change\\n\\nInitial brief\\n' > \"$1\"\n", 0o755)

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
	send("/health\r", "HTTP 200")
	send("/health-legacy\r", "HTTP 503")
	send("/return\r", "MainScreen")
	send("/backend-configs\r", "BackendConfigListScreen")
	send("\r", "Configuration: pty")
	send("\x1b[6~", "change_types:")
	send("/delete\r", "Delete configuration pty?")
	send("\r", "No configurations loaded.")
	send("/return\r", "MainScreen")
	send("/projects\r", "ProjectsListScreen")
	send("/new-project\r", "ProjectCreateScreen")
	send("/editor\r", "ProjectDetailsScreen")
	send("/documents\r", "DocumentScreen")
	send("\x1b[6~", "#80")
	send("\x1b[5~", "#90")
	send("/new-document\r", "Document draft:")
	send("PTY document\r", "saved document #91")
	send("/return\r", "ProjectDetailsScreen")
	assert.Contains(t, capture.after(0), "\x1b[2J", "editor restoration redraws screen")
	send("/return\r", "ProjectsListScreen")
	send("/return\r", "MainScreen")
	send("/epics\r", "PTY Epic")
	send("\r", "Completed: 63")
	send("/edit\r", "EpicUpdateScreen")
	send("\x05", "saved epic")
	assert.Contains(t, capture.after(0), "Name: # PTY Change")
	send("/delete\r", "Are you sure?")
	send("\x1b", "status cancel")
	send("/return\r", "loaded epics")
	send("/new-epic\r", "EpicCreateScreen")
	send("New PTY epic\r", "saved epic")
	send("/return\r", "loaded epics")
	send("/return\r", "MainScreen")
	send("/changes\r", "Rows 1-7 of 30")
	assert.Contains(t, capture.after(0), "\x1b[", "terminal output retains styles")
	send("\x1b[6~", "Rows 2-8 of 30")
	send("\x1b[5~", "Rows 1-7 of 30")
	send("/", "Commands")
	send("\x1b", "Type / for commands")
	send("\r", "loaded change")
	send("/new-testcase\r", "TestCaseCreateScreen")
	send("PTY case\r", "saved test case")
	send("\x1b[6~", "Spec")
	send("\x1b[6~", "PTY case")
	send("/title\r", "ChangeUpdateScreen")
	send("\x05", "saved title")
	send(strings.Repeat("\x1b[6~", 5), "Complete")
	assert.Contains(t, capture.after(0), "73%")
	send("/return\r", "Rows")
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
	var mu sync.Mutex
	epicName := "PTY Epic"
	changeTitle := "PTY Change"
	backendConfigExists := true
	backendConfig := map[string]any{"slug": "pty", "project_docs": []string{strings.Repeat("very-long-document-name-", 12)}, "epic_docs": []string{}, "change_docs": []string{"brief", "spec"}, "change_phases": []string{"backlog"}, "change_colors": []string{"12"}, "change_types": []string{"feature"}}
	var testCases []map[string]any
	docs := make([]map[string]any, 0, 20)
	for id := 90; id >= 71; id-- {
		docs = append(docs, map[string]any{"id": id, "ref_id": 7, "ref_table": "project", "doc_type": "notes", "body": fmt.Sprintf("historical body %d", id), "agent_edit": false, "current": id == 90, "created_at": "2026-09-28T10:00:00Z", "updated_at": "2026-09-28T11:00:00Z", "html": "<p>rendered</p>"})
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		var value any
		switch r.URL.Path {
		case "/api/v1/health":
			require.Equal(t, http.MethodGet, r.Method)
			value = map[string]string{"status": "ok", "api": "ok", "database": "ok"}
		case "/api/health":
			require.Equal(t, http.MethodGet, r.Method)
			w.WriteHeader(503)
			value = map[string]string{"status": "degraded", "api": "ok", "database": "error", "error": "database unavailable"}
		case "/api/v1/config/list":
			value = []any{}
			if backendConfigExists {
				value = []any{backendConfig}
			}
		case "/api/v1/config/details":
			value = backendConfig
		case "/api/v1/config/delete":
			var body struct {
				Slug string `json:"slug"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.Equal(t, "pty", body.Slug)
			backendConfigExists = false
			w.WriteHeader(204)
			return
		case "/api/v1/project/config":
			value = map[string]any{"slug": "pty", "project_docs": []string{"notes"}, "epic_docs": []string{}, "change_docs": []string{"brief", "spec"}, "change_phases": []string{"backlog"}, "change_colors": []string{"12"}, "change_types": []string{}}
		case "/api/v1/project/create":
			w.WriteHeader(201)
			value = map[string]any{"id": 7}
		case "/api/v1/project/details":
			value = terminalProject()
		case "/api/v1/project/list":
			value = []any{terminalProject()}
		case "/api/v1/epic/list":
			value = []any{terminalEpic(epicName)}
		case "/api/v1/epic/details":
			value = terminalEpic(epicName)
		case "/api/v1/epic/create", "/api/v1/epic/update":
			var body struct {
				Name string `json:"name"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			epicName = body.Name
			if strings.HasSuffix(r.URL.Path, "create") {
				w.WriteHeader(201)
				value = map[string]int{"id": 3}
			} else {
				w.WriteHeader(204)
				return
			}
		case "/api/v1/change/list":
			rows := []map[string]any{}
			for i := 1; i <= 30; i++ {
				rows = append(rows, terminalChange(i, fmt.Sprintf("Row %02d", i)))
			}
			value = rows
		case "/api/v1/change/details":
			var body struct {
				ID int `json:"id"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			value = terminalChange(body.ID, changeTitle)
		case "/api/v1/doc/current":
			var request struct {
				RefID    int    `json:"ref_id"`
				RefTable string `json:"ref_table"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
			if request.RefTable == "project" {
				current := []map[string]any{}
				for _, d := range docs {
					if d["current"] == true {
						current = append(current, d)
					}
				}
				value = current
			} else {
				value = []any{}
			}
		case "/api/v1/doc/list":
			var request struct {
				RefID    int    `json:"ref_id"`
				RefTable string `json:"ref_table"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
			require.Equal(t, 7, request.RefID)
			require.Equal(t, "project", request.RefTable)
			value = docs
		case "/api/v1/doc/details":
			var request struct {
				ID int `json:"id"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
			for _, d := range docs {
				if d["id"] == request.ID {
					value = d
					break
				}
			}
			if value == nil {
				http.NotFound(w, r)
				return
			}
		case "/api/v1/doc/insert":
			var request struct {
				RefID     int    `json:"ref_id"`
				RefTable  string `json:"ref_table"`
				DocType   string `json:"doc_type"`
				Body      string `json:"body"`
				AgentEdit bool   `json:"agent_edit"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
			require.Equal(t, 7, request.RefID)
			require.Equal(t, "project", request.RefTable)
			require.Equal(t, "notes", request.DocType)
			require.Equal(t, "PTY document", request.Body)
			require.False(t, request.AgentEdit)
			for _, d := range docs {
				d["current"] = false
			}
			docs = append([]map[string]any{{"id": 91, "ref_id": 7, "ref_table": "project", "doc_type": "notes", "body": request.Body, "agent_edit": false, "current": true, "created_at": "2026-09-28T10:00:00Z", "updated_at": "2026-09-28T11:00:00Z", "html": "<p>rendered</p>"}}, docs...)
			w.WriteHeader(201)
			value = map[string]any{"id": 91}
		case "/api/v1/test-case/list":
			if testCases == nil {
				value = []any{}
			} else {
				value = testCases
			}
		case "/api/v1/test-case/create":
			var body struct {
				ChangeID int    `json:"change_id"`
				Scenario string `json:"scenario"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, 1, body.ChangeID)
			assert.Equal(t, "PTY case", body.Scenario)
			testCases = append(testCases, map[string]any{"id": 31, "change_id": body.ChangeID, "scenario": body.Scenario, "done": false, "created_at": "2026-09-28T10:00:00Z", "updated_at": "2026-09-28T10:00:00Z"})
			w.WriteHeader(201)
			value = map[string]any{"id": 31}
		case "/api/v1/change/update-title":
			var body struct {
				ID    int    `json:"id"`
				Title string `json:"title"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, "# PTY Change\n\nInitial brief\n", body.Title)
			changeTitle = body.Title
			w.WriteHeader(204)
			return
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

func terminalProject() map[string]any {
	return map[string]any{"id": 7, "name": "PTY Project", "config": "pty", "last_ref": 0, "created_at": "2026-09-28T10:00:00Z", "updated_at": "2026-09-28T10:00:00Z", "change_count": 0}
}

func terminalEpic(name string) map[string]any {
	return map[string]any{"id": 3, "project_id": 7, "name": name, "done_tc": 2, "total_tc": 8, "completed": 63, "change_count": 4, "created_at": "2026-09-28T10:00:00Z", "updated_at": "2026-09-28T11:00:00Z"}
}

func terminalChange(id int, title string) map[string]any {
	return map[string]any{"id": id, "project_id": 7, "ref_uuid": "0198a86f-9b8a-7d89-ae5b-6f25b528b04c", "ref": nil, "slug": nil, "epic_id": nil, "epic_name": nil, "change_phase": "backlog", "change_types": []string{}, "title": title, "open": true, "done_tc": 2, "total_tc": 9, "completed": 73, "updated_at": "2026-09-28T11:00:00Z", "after_change_id": nil, "pr_url": "", "created_at": "2026-09-28T10:00:00Z"}
}
