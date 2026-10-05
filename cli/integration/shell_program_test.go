package integration_test

import (
	"bytes"
	"cli/internal/agent"
	"cli/internal/app"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCLIProgramStartupNavigationAndSelection(t *testing.T) {
	t.Run("quit immediately after selection", func(t *testing.T) {
		for _, exit := range []string{"\x1b", "/quit\r", "\x03"} {
			t.Run(fmt.Sprintf("%q", exit), func(t *testing.T) {
				backend := newShellBackend(t, false)
				root := t.TempDir()
				writeProgramConfig(t, root, backend.URL)
				s := startProgram(t, root, "")
				s.navigate(t, "/select-project\r", "Second Project")
				// Submit selection and exit together, without waiting for persistence.
				s.send(t, "\x1b[B\r"+exit)
				require.NoError(t, s.waitDone(t))
				assert.Contains(t, readFile(t, filepath.Join(root, ".mch/config.yaml")), "project_id: 8")
			})
		}
	})
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprint("save failure=", fail), func(t *testing.T) {
			backend := newShellBackend(t, false)
			root := t.TempDir()
			writeProgramConfig(t, root, backend.URL)
			require.NoError(t, os.Chmod(filepath.Join(root, ".mch/config.yaml"), 0o600))
			sentinel := filepath.Join(root, ".mch/tmp/existing.md")
			require.NoError(t, os.MkdirAll(filepath.Dir(sentinel), 0o755))
			require.NoError(t, os.WriteFile(sentinel, []byte("user draft"), 0o644))
			s := startProgram(t, root, "")
			if fail {
				require.NoError(t, os.Rename(filepath.Join(root, ".mch/config.yaml"), filepath.Join(root, ".mch/previous.yaml")))
				require.NoError(t, os.Mkdir(filepath.Join(root, ".mch/config.yaml"), 0o755))
			}
			s.navigate(t, "/select-project\r", "Second Project")
			s.send(t, "\x1b[B\r")
			if fail {
				s.waitFor(t, "config save failed")
				assert.Contains(t, readFile(t, filepath.Join(root, ".mch/previous.yaml")), "project_id: 7")
			} else {
				s.waitFor(t, "project selection saved")
				assert.Contains(t, readFile(t, filepath.Join(root, ".mch/config.yaml")), "project_id: 8")
				info, err := os.Stat(filepath.Join(root, ".mch/config.yaml"))
				require.NoError(t, err)
				assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
			}
			s.waitFor(t, "Second Project")
			s.navigate(t, "/config\r", "project_id: 8")
			s.send(t, "/return\r")
			s.navigate(t, "/help\r", "MainHelpScreen")
			s.send(t, "/return\r")
			s.navigate(t, "/changes\r", "no changes")
			s.finishFromChanges(t)
			assert.Equal(t, "user draft", readFile(t, sentinel))
			assert.NoDirExists(t, filepath.Join(root, ".mch/default"))
			assert.NotContains(t, s.output.String(), "AgentRunningScreen")
		})
	}
}

func TestCLIProgramEditorSaveAndFailure(t *testing.T) {
	t.Run("slash-prefixed testcase", testProgramSlashPrefixedTestCase)
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprint("save failure=", fail), func(t *testing.T) {
			backend := newShellBackend(t, fail)
			root := t.TempDir()
			writeProgramConfig(t, root, backend.URL)
			s := startProgram(t, root, "Edited project")
			s.navigate(t, "/projects\r", "ProjectsListScreen")
			s.navigate(t, "/new-project\r", "ProjectCreateScreen")
			s.send(t, "/editor\r")
			if fail {
				s.waitFor(t, "Error: /api/v1/project/create: 500")
				s.waitFor(t, "save rejected")
				s.navigate(t, "\x1b", "status prompt")
				s.navigate(t, "\x1b", "ProjectsListScreen")
			} else {
				s.waitFor(t, "ProjectDetailsScreen")
				s.navigate(t, "/return\r", "ProjectsListScreen")
			}
			s.navigate(t, "/return\r", "MainScreen")
			s.send(t, "/quit\r")
			require.NoError(t, s.waitDone(t))
			assert.Contains(t, s.output.String(), "\x1b[2J")
		})
	}
}

type programSession struct {
	finished   bool
	input      *os.File
	output     *synchronizedBuffer
	done       chan error
	controller chan app.ProgramController
}

func startProgram(t *testing.T, root, editorOutput string, runners ...agent.Runner) *programSession {
	t.Helper()
	reader, writer, err := os.Pipe()
	require.NoError(t, err)
	output := newSynchronizedBuffer()
	ctx, cancel := context.WithCancel(context.Background())
	editorSource := filepath.Join(t.TempDir(), "editor-output.md")
	require.NoError(t, os.WriteFile(editorSource, []byte(editorOutput), 0o644))
	t.Setenv("EDITOR", "cp "+editorSource)
	session := &programSession{
		input: writer, output: output, done: make(chan error, 1),
		controller: make(chan app.ProgramController, 1),
	}
	go func() {
		var runner agent.Runner
		if len(runners) > 0 {
			runner = runners[0]
		}
		session.done <- app.RunProgramWithIO(nil, reader, output, app.ProgramOptions{
			Context:        ctx,
			RepositoryRoot: root,
			BriefRunner:    runner,
			ProgramReady: func(controller app.ProgramController) {
				session.controller <- controller
			},
		})
	}()
	t.Cleanup(func() {
		if !session.finished {
			select {
			case controller := <-session.controller:
				controller.Quit()
			case <-time.After(time.Second):
				cancel()
			}
			select {
			case <-session.done:
				session.finished = true
			case <-time.After(5 * time.Second):
				cancel()
				t.Error("program cleanup timed out")
			}
		}
		cancel()
		_ = writer.Close()
		_ = reader.Close()
	})
	output.waitFor(t, "Program Project")
	return session
}

func (s *programSession) finishFromDetails(t *testing.T) {
	t.Helper()
	changesBefore := s.output.count("ChangesListScreen")
	s.send(t, "\x03")
	s.output.waitForCount(t, "ChangesListScreen", changesBefore+1)
	s.finishFromChanges(t)
}

func (s *programSession) finishFromChanges(t *testing.T) {
	t.Helper()
	mainBefore := s.output.count("MainScreen")
	s.send(t, "\x03")
	s.output.waitForCount(t, "MainScreen", mainBefore+1)
	s.send(t, "\x03")
	require.NoError(t, s.waitDone(t))
}

func (s *programSession) waitFor(t *testing.T, marker string) {
	t.Helper()
	s.output.waitFor(t, marker)
}

func (s *programSession) send(t *testing.T, keys string) {
	t.Helper()
	_, err := io.WriteString(s.input, keys)
	require.NoError(t, err)
}

func (s *programSession) waitDone(t *testing.T) error {
	t.Helper()
	select {
	case err := <-s.done:
		s.finished = true
		return err
	case <-time.After(5 * time.Second):
		require.Fail(t, "CLI program did not exit")
		return nil
	}
}

type synchronizedBuffer struct {
	mu      sync.Mutex
	buffer  bytes.Buffer
	changed chan struct{}
}

func newSynchronizedBuffer() *synchronizedBuffer {
	return &synchronizedBuffer{changed: make(chan struct{}, 1)}
}

func (b *synchronizedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	n, err := b.buffer.Write(p)
	b.mu.Unlock()
	select {
	case b.changed <- struct{}{}:
	default:
	}
	return n, err
}

func (b *synchronizedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.String()
}

func (b *synchronizedBuffer) count(marker string) int {
	return strings.Count(strings.Join(strings.Fields(ansi.Strip(b.String())), " "), strings.Join(strings.Fields(marker), " "))
}

func (b *synchronizedBuffer) waitFor(t *testing.T, marker string) {
	t.Helper()
	b.waitForCount(t, marker, 1)
}

func (b *synchronizedBuffer) waitForCount(t *testing.T, marker string, count int) {
	t.Helper()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	for b.count(marker) < count {
		select {
		case <-b.changed:
		case <-timer.C:
			require.FailNow(t, "timed out waiting for CLI output", "marker %q count %d; output:\n%s", marker, count, b.String())
		}
	}
}

func writeProgramConfig(t *testing.T, root, backendURL string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".mch"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".mch/config.yaml"), []byte("backend_url: "+backendURL+"\nproject_id: 7\n"), 0o644))
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("../..")
	require.NoError(t, err)
	return root
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(b)
}
func writeProgramJSON(w http.ResponseWriter, value any) { _ = json.NewEncoder(w).Encode(value) }

func newShellBackend(t *testing.T, failSave bool) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/doc/comment-list":
			writeProgramJSON(w, []any{})
		case "/api/v1/project/config":
			writeProgramJSON(w, programProjectConfig())
		case "/api/v1/project/list":
			writeProgramJSON(w, []map[string]any{programProject(7, "Program Project"), programProject(8, "Second Project")})
		case "/api/v1/project/details":
			var body struct {
				ID int `json:"id"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			name := "Program Project"
			if body.ID == 8 {
				name = "Second Project"
			}
			writeProgramJSON(w, programProject(body.ID, name))
		case "/api/v1/project/create":
			if failSave {
				http.Error(w, "save rejected", 500)
				return
			}
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			w.WriteHeader(http.StatusCreated)
			writeProgramJSON(w, map[string]any{"id": 7})
		case "/api/v1/change/list":
			writeProgramJSON(w, []any{})
		default:
			t.Errorf("unexpected request %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func waitForProgramOutput(t *testing.T, path, marker string, done <-chan error) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		content, _ := os.ReadFile(path)
		if strings.Contains(string(content), marker) {
			return
		}
		select {
		case err := <-done:
			t.Fatalf("CLI exited before %q: %v\n%s", marker, err, content)
		default:
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %q in %s", marker, path)
}

func (s *programSession) navigate(t *testing.T, keys, marker string) {
	t.Helper()
	before := s.output.count(marker)
	s.send(t, keys)
	s.output.waitForCount(t, marker, before+1)
}

func programProject(id int, name string) map[string]any {
	return map[string]any{"id": id, "name": name, "config_slug": "program", "active": true, "last_ref": 12, "created_at": "2026-09-28T10:00:00Z", "updated_at": "2026-09-28T11:00:00Z", "change_count": 3}
}

func programProjectConfig() map[string]any {
	return map[string]any{"slug": "program", "project_docs": []string{"readme"}, "epic_docs": []string{"brief"}, "change_docs": []string{"brief", "spec", "pr"}, "change_phases": []string{"backlog"}, "change_colors": []string{"12"}, "change_types": []string{"feature"}}
}

func programChange(id int, title string) map[string]any {
	return map[string]any{"id": id, "project_id": 7, "ref_uuid": "0198a86f-9b8a-7d89-ae5b-6f25b528b04c", "ref_slug": nil, "epic_id": nil, "epic_name": nil, "change_phase": "backlog", "change_types": []string{}, "title": title, "active": true, "done_tc": int64(2), "total_tc": int64(9), "completed": int64(73), "updated_at": "2026-09-28T11:00:00Z", "after_change_id": nil, "after_change_name": nil, "pr_url": "", "created_at": "2026-09-28T10:00:00Z"}
}

func programDocument(kind, body string) map[string]any {
	return map[string]any{"id": 91, "ref_id": 12, "ref_table": "change", "doc_type": kind, "body": body, "agent_edit": false, "deleted_at": nil, "created_at": "2026-09-28T10:00:00Z", "updated_at": "2026-09-28T10:00:00Z", "html": ""}
}
