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
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
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
	require.NoError(t, os.MkdirAll(filepath.Join(repoRoot, ".mch", "default", "prompts"), 0o755))
	writeTerminalFile(t, filepath.Join(repoRoot, ".mch", "default", "prompts", "brief-rewrite.md"), "Clarify the brief and write structured output.\n", 0o644)

	stubDir := filepath.Join(testRoot, "bin")
	require.NoError(t, os.MkdirAll(stubDir, 0o755))
	writeTerminalFile(t, filepath.Join(stubDir, "editor"), "#!/bin/sh\nprintf '# PTY Change\\n\\nInitial brief\\n' > \"$1\"\n", 0o755)
	writeTerminalFile(t, filepath.Join(stubDir, "codex"), "#!/bin/sh\nprintf '%s\\n' \"$$\" > \"$MCH_PTY_AGENT_PID\"\nprintf 'PTY agent started\\n'\nsleep 30\n", 0o755)

	childPIDPath := filepath.Join(testRoot, "mch.pid")
	childExitPath := filepath.Join(testRoot, "mch.exit")
	agentPIDPath := filepath.Join(testRoot, "agent.pid")
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
		"MCH_PTY_AGENT_PID="+agentPIDPath,
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
	send("x", "x")
	send("\x1b", "status prompt cleared")
	send("\x03", "MainScreen")
	send("/backend-configs\r", "BackendConfigListScreen")
	send("\r", "Configuration: pty")
	send("\x1b[6~", "change_types:")
	send("/delete\r", "Delete configuration pty?")
	send("\r", "No configurations loaded.")
	send("/new-config\r", "BackendConfigFormScreen")
	send("temporary", "temporary")
	send("\x03", "status prompt cleared")
	send("\x1b", "BackendConfigListScreen")
	send("/return\r", "MainScreen")
	send("/projects\r", "ProjectsListScreen")
	send("/new-project\r", "ProjectCreateScreen")
	send("/editor\r", "ProjectDetailsScreen")
	send("/documents\r", "DocumentScreen")
	send("\x1b[6~", "#80")
	send("\x1b[5~", "#90")
	send("/new-document\r", "Document draft:")
	send("Temporary", "Temporary")
	send("\x03", "status prompt cleared")
	send("PTY document\r", "saved document #91")
	send("\x03", "ProjectDetailsScreen")
	assert.Contains(t, capture.after(0), "\x1b[2J", "editor restoration redraws screen")
	send("/return\r", "ProjectsListScreen")
	send("/new-project\r", "ProjectCreateScreen")
	send("Temporary", "Temporary")
	send("\x03", "status prompt cleared")
	send("\x1b", "ProjectsListScreen")
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
	send("/new-epic\r", "EpicCreateScreen")
	send("Temporary", "Temporary")
	send("\x1b", "status prompt cleared")
	send("\x03", "EpicsListScreen")
	send("/return\r", "MainScreen")
	send("/changes\r", "Rows 1-7 of 30")
	assert.Contains(t, capture.after(0), "\x1b[", "terminal output retains styles")
	send("\x1b[6~", "Rows 2-8 of 30")
	send("\x1b[5~", "Rows 1-7 of 30")
	send("/", "Create a change")
	assert.Contains(t, capture.after(0), "▄")
	assert.Contains(t, capture.after(0), "▀")
	send("/", "Commands: no options")
	assert.Contains(t, capture.after(0), "//")
	send("\x7f", "Create a change")
	send("\x7f", "Type / for commands")
	send("/", "Create a change")
	send("\x1b[3~", "Type / for commands")
	send("/", "Create a change")
	send("\x03", "Type / for commands")
	send("\r", "loaded change")
	send("/phase\r", "Phase >")
	send("\x1b", "status cancel")
	send("/epic\r", "New PTY epic #3")
	send("\x03", "status cancel")
	send("/types\r", "Types >")
	send("\x1b", "status cancel")
	// Capture real bat stdout inside the PTY, including older/deleted versions,
	// scrolling syntax colors, same-ID activation and confirmed deletion.
	send(strings.Repeat("\x1b[B", 7)+"\x08", "Version #65")
	historyOutput := capture.after(0)
	for _, mode := range []string{"\x1b[?1000h", "\x1b[?1002h", "\x1b[?1003h", "\x1b[?1006h"} {
		require.NotContains(t, historyOutput, mode, "native terminal selection must remain enabled")
	}
	assert.Contains(t, historyOutput, "created_at:")
	assert.Contains(t, historyOutput, "updated_at:")
	assert.Contains(t, historyOutput, "\x1b[38;5;211mdeleted_at:")
	syntaxColor := regexp.MustCompile(`\x1b\[38;2;[0-9;]+m`)
	require.NotEmpty(t, syntaxColor.FindString(historyOutput), "captured bat syntax colors")
	offset := capture.len()
	send("\x1b[6~\x1b[6~", "colored_line_20")
	require.NotEmpty(t, syntaxColor.FindString(capture.after(offset)), "bat colors survive scrolling")
	offset = capture.len()
	send("\x1b[C", "Version #64")
	_, err = io.WriteString(stdin, "\x1b[C") // oldest boundary does not wrap or redraw
	require.NoError(t, err)
	require.NotEmpty(t, syntaxColor.FindString(capture.after(offset)), "bat colors survive version change")
	send("\x1b[D", "Version #65")
	send(" ", "committed active selection document #65")
	send("\x1b", "returned from history")
	send("\x1b[3~", "Are you sure?")
	assert.Contains(t, capture.after(0), "\x1b[38;5;183mAre you sure?")
	send("\x03", "cancel")
	send("\x1b[3~", "Are you sure?")
	send("\r", "committed delete document #65")
	send("\x08", "Version #65")
	send(" ", "committed active selection document #65")
	send("\x03", "returned from history")
	send("/title\r", "Title > ")
	send("\x03", "status prompt cleared")
	send("/pr-url\r", "PR URL >")
	send("\x1b", "Type / for commands")
	send("\x1b", "ChangesListScreen")
	send("\r", "loaded change")
	send(strings.Repeat("\x1b[B", 11)+"\r", "old-slug")
	_, err = io.WriteString(stdin, "A [")
	require.NoError(t, err)
	send("\x1b", "status prompt cleared")
	send("\x03", "ChangesListScreen")
	send("\r", "loaded change")
	send(strings.Repeat("\x1b[B", 11)+"\r", "old-slug")
	send(strings.Repeat("\x7f", 8)+"new-slug\r", "saved slug")
	send("/brief-clarify\r", "brief ready for editing")
	send("Temporary", "Temporary")
	send("\x03", "status prompt cleared")
	send("\x1b[6~", "Backend current brief")
	send("\x1b[5~", "Original user brief")
	send("\x05", "Original user brief: # PTY Change")
	assert.Contains(t, capture.after(0), "\x1b[2J", "workflow editor restores terminal redraw")
	send("/confirm\r", "agent stdout: PTY agent started")
	send("\x1b", "loaded change")
	pidBytes, err := os.ReadFile(agentPIDPath)
	require.NoError(t, err)
	pid, err := strconv.Atoi(strings.TrimSpace(string(pidBytes)))
	require.NoError(t, err)
	require.Eventually(t, func() bool { return syscall.Kill(pid, 0) == syscall.ESRCH }, 3*time.Second, 20*time.Millisecond, "canceled agent must be reaped")
	send("/new-testcase\r", "TestCaseCreateScreen")
	send("Temporary", "Temporary")
	send("\x03", "status prompt cleared")
	send("\x1b", "ChangeDetailsScreen")
	send("/new-testcase\r", "TestCaseCreateScreen")
	send("PTY case\r", "saved test case")
	send(strings.Repeat("\x1b[A", 30)+strings.Repeat("\x1b[B", 8)+" ", "[✓] PTY case")
	send(" ", "[ ] PTY case")
	send("\x1b[A\x1b[A ", "Initial brief")
	send("\x1b", "returned from view")
	send("\x1b[6~", "Complete")
	send("/title\r", "Title > ")
	send("\x05", "saved title")
	send(strings.Repeat("\x1b[6~", 5), "Complete")
	assert.Contains(t, capture.after(0), "73%")
	send("/return\r", "Rows")
	send("\x08", "Space activate")
	send(" ", "activated change #31")
	send("\x03", "of 31")
	send("/return\r", "MainScreen")
	require.Eventually(t, func() bool {
		entries, readErr := os.ReadDir(filepath.Join(repoRoot, ".mch", "tmp"))
		return readErr == nil && len(entries) == 0
	}, 3*time.Second, 20*time.Millisecond, "canceled clarification leaves no operation files")
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
	changeSlug := "111-old-slug"
	changeBrief := strings.Repeat("Precise workflow context and constraints. ", 12)
	changeBriefID := 41
	backendConfigExists := true
	backendConfig := map[string]any{"slug": "pty", "project_docs": []string{strings.Repeat("very-long-document-name-", 12)}, "epic_docs": []string{}, "change_docs": []string{"brief", "spec"}, "change_phases": []string{"backlog"}, "change_colors": []string{"12"}, "change_types": []string{"feature"}}
	var testCases []map[string]any
	inactive := true
	specActive := 64
	specRows := []map[string]any{terminalSpec(65, true), terminalSpec(64, false)}
	activeDocumentID := 90
	docs := make([]map[string]any, 0, 20)
	for id := 90; id >= 71; id-- {
		docs = append(docs, map[string]any{"id": id, "ref_id": 7, "ref_table": "project", "doc_type": "notes", "body": fmt.Sprintf("historical body %d", id), "agent_edit": false, "deleted_at": nil, "created_at": "2026-09-28T10:00:00Z", "updated_at": "2026-09-28T11:00:00Z", "html": "<p>rendered</p>"})
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		var value any
		switch r.URL.Path {
		case "/api/v1/doc/comment-list":
			value = []any{}
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
		case "/api/v1/change/list-inactive":
			value = []any{}
			if inactive {
				value = []any{terminalChange(31, "Inactive PTY Change")}
			}
		case "/api/v1/change/update-active":
			var payload map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
			require.Equal(t, map[string]any{"id": float64(31), "active": true}, payload)
			inactive = false
			w.WriteHeader(204)
			return
		case "/api/v1/doc/delete", "/api/v1/doc/active-set":
			var payload map[string]int
			require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
			require.Equal(t, 65, payload["id"])
			if strings.HasSuffix(r.URL.Path, "delete") {
				specActive = 0
				specRows[0]["deleted_at"] = "2026-09-28T12:00:00Z"
			} else {
				specActive = 65
				specRows[0]["deleted_at"] = nil
			}
			w.WriteHeader(204)
			return
		case "/api/v1/change/list":
			rows := []map[string]any{}
			for i := 1; i <= 30; i++ {
				rows = append(rows, terminalChange(i, fmt.Sprintf("Row %02d", i)))
			}
			if !inactive {
				rows = append(rows, terminalChange(31, "Activated PTY Change"))
			}
			value = rows
		case "/api/v1/change/details":
			var body struct {
				ID int `json:"id"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			value = terminalChange(body.ID, changeTitle)
			if body.ID == 1 {
				value.(map[string]any)["ref_slug"] = changeSlug
			}
		case "/api/v1/doc/list-active":
			var request struct {
				RefID    int    `json:"ref_id"`
				RefTable string `json:"ref_table"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
			if request.RefTable == "project" {
				current := []map[string]any{}
				for _, d := range docs {
					if int(d["id"].(int)) == activeDocumentID {
						current = append(current, d)
					}
				}
				value = current
			} else {
				value = []any{map[string]any{"id": changeBriefID, "ref_id": request.RefID, "ref_table": "change", "doc_type": "brief", "body": changeBrief, "agent_edit": false, "deleted_at": nil, "created_at": "2026-09-28T10:00:00Z", "updated_at": "2026-09-28T11:00:00Z", "html": "<p>rendered</p>"}}
			}
			if request.RefTable == "change" && specActive > 0 {
				for _, d := range specRows {
					if d["id"] == specActive {
						value = append([]any{d}, value.([]any)...)
					}
				}
			}
		case "/api/v1/doc/list":
			var request struct {
				RefID    int    `json:"ref_id"`
				RefTable string `json:"ref_table"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
			if request.RefTable == "change" {
				require.Equal(t, 1, request.RefID)
				value = specRows
			} else {
				require.Equal(t, 7, request.RefID)
				require.Equal(t, "project", request.RefTable)
				value = docs
			}
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
			if request.RefTable == "change" {
				require.Equal(t, 1, request.RefID)
				require.Equal(t, "brief", request.DocType)
				require.Equal(t, "# PTY Change\n\nInitial brief\n", request.Body)
				require.False(t, request.AgentEdit)
				changeBrief = strings.TrimSpace(request.Body)
				changeBriefID++
				w.WriteHeader(201)
				value = map[string]int{"id": changeBriefID}
				break
			}
			require.Equal(t, 7, request.RefID)
			require.Equal(t, "project", request.RefTable)
			require.Equal(t, "notes", request.DocType)
			require.Equal(t, "PTY document", request.Body)
			require.False(t, request.AgentEdit)
			activeDocumentID = 91
			docs = append([]map[string]any{{"id": 91, "ref_id": 7, "ref_table": "project", "doc_type": "notes", "body": request.Body, "agent_edit": false, "deleted_at": nil, "created_at": "2026-09-28T10:00:00Z", "updated_at": "2026-09-28T11:00:00Z", "html": "<p>rendered</p>"}}, docs...)
			w.WriteHeader(201)
			value = map[string]any{"id": 91}
		case "/api/v1/test-case/list":
			if testCases == nil {
				value = []any{}
			} else {
				value = testCases
			}
		case "/api/v1/test-case/update-done":
			var body struct {
				ID   int  `json:"id"`
				Done bool `json:"done"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.Equal(t, 31, body.ID)
			require.Len(t, testCases, 1)
			testCases[0]["done"] = body.Done
			w.WriteHeader(http.StatusNoContent)
			return
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
		case "/api/v1/change/update-slug":
			var body struct {
				ID   int    `json:"id"`
				Slug string `json:"slug"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, 1, body.ID)
			assert.Equal(t, "new-slug", body.Slug)
			changeSlug = "111-" + body.Slug
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
		if strings.Contains(strings.Join(strings.Fields(ansi.Strip(c.after(offset))), " "), strings.Join(strings.Fields(marker), " ")) {
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
	return map[string]any{"id": 7, "name": "PTY Project", "config_slug": "pty", "active": true, "last_ref": 0, "created_at": "2026-09-28T10:00:00Z", "updated_at": "2026-09-28T10:00:00Z", "change_count": 0}
}

func terminalEpic(name string) map[string]any {
	return map[string]any{"id": 3, "project_id": 7, "name": name, "active": true, "done_tc": 2, "total_tc": 8, "completed": 63, "change_count": 4, "created_at": "2026-09-28T10:00:00Z", "updated_at": "2026-09-28T11:00:00Z"}
}

func terminalChange(id int, title string) map[string]any {
	return map[string]any{"id": id, "project_id": 7, "ref_uuid": "0198a86f-9b8a-7d89-ae5b-6f25b528b04c", "ref_slug": fmt.Sprintf("%03d-old-slug", id+110), "epic_id": nil, "epic_name": nil, "change_phase": "backlog", "change_types": []string{}, "title": title, "active": true, "done_tc": 2, "total_tc": 9, "completed": 73, "updated_at": "2026-09-28T11:00:00Z", "after_change_id": nil, "after_change_name": nil, "pr_url": "", "created_at": "2026-09-28T10:00:00Z"}
}

func terminalSpec(id int, deleted bool) map[string]any {
	body := fmt.Sprintf("# Highlighted version %d\n\n```go\n", id)
	for i := 0; i < 45; i++ {
		body += fmt.Sprintf("var colored_line_%02d = \"colored value\"\n", i)
	}
	body += "```\n"
	var deletedAt any
	if deleted {
		deletedAt = "2026-09-28T12:00:00Z"
	}
	return map[string]any{"id": id, "ref_id": 1, "ref_table": "change", "doc_type": "spec", "body": body, "agent_edit": false, "deleted_at": deletedAt, "created_at": "2026-09-28T10:00:00Z", "updated_at": "2026-09-28T11:00:00Z", "html": "<p>rendered</p>"}
}
