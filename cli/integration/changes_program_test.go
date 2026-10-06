package integration_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func changePromptLabel(command string) string {
	switch command {
	case "/title":
		return "Title >"
	case "/pr-url":
		return "PR URL >"
	case "/after-change":
		return "After Change >"
	default:
		return command + " >"
	}
}

func TestCLIProgramChangeCRUDAndPartialSuccess(t *testing.T) {
	for _, partial := range []bool{false, true} {
		t.Run(fmt.Sprint(partial), func(t *testing.T) {
			brief := "# Inferred title\n\nplain brief\twith bytes\n"
			var mu sync.Mutex
			change := programChange(12, "Original")
			exists := true
			failRead := false
			writes := map[string][]map[string]any{}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				var body map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				switch r.URL.Path {
				case "/api/v1/doc/comment-list":
					writeProgramJSON(w, []any{})
				case "/api/v1/project/details":
					writeProgramJSON(w, programProject(7, "Program Project"))
				case "/api/v1/project/config":
					cfg := programProjectConfig()
					cfg["change_phases"] = []string{"review", "backlog"}
					cfg["change_types"] = []string{"fix", "feature"}
					writeProgramJSON(w, cfg)
				case "/api/v1/epic/list":
					writeProgramJSON(w, []any{programEpic(3, 7, "Parent")})
				case "/api/v1/change/list":
					require.Equal(t, float64(7), body["project_id"])
					if failRead {
						failRead = false
						http.Error(w, "refresh unavailable", 500)
						return
					}
					if exists && change["active"] == body["active"] {
						writeProgramJSON(w, []any{change})
					} else {
						writeProgramJSON(w, []any{})
					}
				case "/api/v1/change/details":
					require.Equal(t, float64(12), body["id"])
					if failRead {
						failRead = false
						http.Error(w, "refresh unavailable", 500)
						return
					}
					writeProgramJSON(w, change)
				case "/api/v1/doc/list-active", "/api/v1/test-case/list":
					writeProgramJSON(w, []any{})
				case "/api/v1/change/create":
					writes["create"] = append(writes["create"], body)
					require.Equal(t, float64(7), body["project_id"])
					require.Equal(t, "Inferred title", body["title"])
					require.Equal(t, brief, body["brief"])
					require.Regexp(t, `^[0-9a-f-]{36}$`, body["ref_uuid"])
					change["title"] = body["title"]
					exists = true
					failRead = partial
					w.WriteHeader(201)
					writeProgramJSON(w, map[string]int{"id": 12})
				default:
					op := strings.TrimPrefix(r.URL.Path, "/api/v1/change/update-")
					keys := map[string]string{"title": "title", "phase": "change_phase", "types": "change_types", "epic": "epic_id", "after-change": "after_change_id", "active": "active", "pr-url": "pr_url"}
					key, ok := keys[op]
					if !ok {
						t.Errorf("unexpected route %s", r.URL.Path)
						http.NotFound(w, r)
						return
					}
					require.Len(t, body, 2)
					require.Equal(t, float64(12), body["id"])
					value, present := body[key]
					require.True(t, present)
					writes[op] = append(writes[op], body)
					change[key] = value
					if op == "epic" {
						change["epic_name"] = nil
						if value != nil {
							change["epic_name"] = "Parent"
						}
					}
					failRead = partial
					w.WriteHeader(204)
				}
			}))
			defer server.Close()
			root := t.TempDir()
			writeProgramConfig(t, root, server.URL)
			s := startProgram(t, root, brief, programAgent(t, root, "unchanged"))
			editor := filepath.Join(root, "editor-value")
			require.NoError(t, os.WriteFile(editor, []byte(brief), 0o600))
			t.Setenv("EDITOR", "cp "+editor)
			edit := func(command, value, marker string) {
				t.Helper()
				require.NoError(t, os.WriteFile(editor, []byte(value), 0o600))
				s.navigate(t, command+"\r", changePromptLabel(command))
				s.navigate(t, "\x05", marker)
			}
			recoverRead := func() {
				t.Helper()
				if partial {
					s.waitFor(t, "/retry reads only")
					s.navigate(t, "/retry\r", "loaded change")
				}
			}
			clearAndReplace := func(command, value, marker string) {
				t.Helper()
				s.navigate(t, command+"\r", changePromptLabel(command))
				s.navigate(t, "\x03", "cleared")
				s.navigate(t, command+"\r", changePromptLabel(command))
				s.send(t, "\x01\x0b")
				s.navigate(t, value+"\r", marker)
			}
			s.navigate(t, "/changes\r", "Rows 1-1 of 1")
			s.navigate(t, "/help\r", "PR URL requires HTTP(S)")
			s.navigate(t, "/return\r", "Rows 1-1 of 1")
			t.Setenv("TMPDIR", t.TempDir())
			s.navigate(t, "/new-change\r", "created change #12")
			recoverRead()
			s.waitFor(t, "Inferred title")
			edit("/title", "/save", "saved title")
			recoverRead()
			s.navigate(t, "/title\r", "Title >")
			s.navigate(t, "\r", "unchanged")
			s.navigate(t, "/phase\r", "review")
			s.navigate(t, "\x1b[A\r", "saved phase")
			recoverRead()
			s.navigate(t, "/types\r", "press <space> to change")
			s.navigate(t, " \r", "saved types")
			recoverRead()
			s.navigate(t, "/types\r", "press <space> to change")
			s.navigate(t, " \r", "saved types")
			recoverRead()
			s.navigate(t, "/epic\r", "Parent")
			s.navigate(t, "\x1b[A\r", "saved epic")
			recoverRead()
			s.navigate(t, "/epic\r", "Parent")
			s.navigate(t, "\x1b[B\r", "saved epic")
			recoverRead()
			clearAndReplace("/after-change", "9", "saved after-change")
			recoverRead()
			edit("/after-change", "null", "saved after-change")
			recoverRead()
			s.navigate(t, "/active\r", "saved active")
			recoverRead()
			clearAndReplace("/pr-url", "https://example.test/pr/1", "saved pr-url")
			recoverRead()
			s.waitFor(t, "Created:")
			s.navigate(t, strings.Repeat("\x1b[6~", 8), "PR URL │")
			s.waitFor(t, "73%")
			s.waitFor(t, "https://example.test/pr/1")
			s.navigate(t, "/delete\r", "Are you sure?")
			s.navigate(t, "\r", "status deactivated change")
			if partial {
				s.navigate(t, "/return\r", "MainScreen")
				s.navigate(t, "/changes\r", "no changes")
			}
			s.waitFor(t, "No changes.")
			s.finishFromChanges(t)
			mu.Lock()
			defer mu.Unlock()
			for _, op := range []string{"create", "title", "phase", "pr-url"} {
				assert.Len(t, writes[op], 1, op)
			}
			for _, op := range []string{"types", "epic", "after-change", "active"} {
				assert.Len(t, writes[op], 2, op)
			}
			assert.Equal(t, "/save", writes["title"][0]["title"])
			assert.Equal(t, "review", writes["phase"][0]["change_phase"])
			assert.Equal(t, []any{"fix"}, writes["types"][0]["change_types"])
			assert.Equal(t, float64(3), writes["epic"][0]["epic_id"])
			assert.Equal(t, float64(9), writes["after-change"][0]["after_change_id"])
			assert.Equal(t, "https://example.test/pr/1", writes["pr-url"][0]["pr_url"])
			assert.Equal(t, false, writes["active"][0]["active"])
			assert.Empty(t, writes["types"][1]["change_types"])
			assert.Nil(t, writes["epic"][1]["epic_id"])
			assert.Nil(t, writes["after-change"][1]["after_change_id"])
			assert.Contains(t, s.output.String(), "\x1b[2J")
		})
	}
}

func TestCLIProgramChangeDelayedScopeAndShutdown(t *testing.T) {
	for _, mode := range []string{"leave detail", "switch project", "shutdown", "empty find list", "empty find detail"} {
		t.Run(mode, func(t *testing.T) {
			started, canceled := make(chan struct{}), make(chan struct{})
			var readStarted atomic.Bool
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					ID        int `json:"id"`
					ProjectID int `json:"project_id"`
				}
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				switch r.URL.Path {
				case "/api/v1/doc/comment-list":
					writeProgramJSON(w, []any{})
				case "/api/v1/project/details":
					name := "Program Project"
					if body.ID == 8 {
						name = "Second Project"
					}
					writeProgramJSON(w, programProject(body.ID, name))
				case "/api/v1/project/list":
					writeProgramJSON(w, []any{programProject(7, "Program Project"), programProject(8, "Second Project")})
				case "/api/v1/project/config":
					writeProgramJSON(w, programProjectConfig())
				case "/api/v1/change/list":
					if mode == "leave detail" || mode == "empty find detail" || body.ProjectID == 8 || (mode == "empty find list" && readStarted.Swap(true)) {
						writeProgramJSON(w, []any{scopedProgramChange(3, body.ProjectID, "Current change")})
						return
					}
					close(started)
					<-r.Context().Done()
					close(canceled)
				case "/api/v1/change/details":
					if mode == "empty find detail" && readStarted.Swap(true) {
						writeProgramJSON(w, scopedProgramChange(body.ID, 7, "Current change"))
						return
					}
					close(started)
					<-r.Context().Done()
					close(canceled)
				case "/api/v1/doc/list-active", "/api/v1/test-case/list":
					writeProgramJSON(w, []any{})
				default:
					t.Errorf("unexpected route %s", r.URL.Path)
				}
			}))
			defer server.Close()
			root := t.TempDir()
			writeProgramConfig(t, root, server.URL)
			s := startProgram(t, root, "")
			if mode == "leave detail" || mode == "empty find detail" {
				s.navigate(t, "/changes\r", "Current change")
				s.navigate(t, "\r", "loading change")
			} else {
				s.navigate(t, "/changes\r", "Changes: loading")
			}
			select {
			case <-started:
			case <-time.After(time.Second):
				t.Fatal("change read did not start")
			}
			if mode == "shutdown" {
				controller := <-s.controller
				controller.Quit()
				require.NoError(t, s.waitDone(t))
			} else {
				if strings.HasPrefix(mode, "empty find") {
					findCommand := "/find\r"
					if mode == "empty find list" {
						findCommand = "/find-filter\r"
					}
					s.navigate(t, findCommand, "FindInputScreen")
					if mode == "empty find detail" {
						s.navigate(t, "\r", "Current change")
					} else {
						s.navigate(t, "\r", "Current change")
					}
				}
				if mode == "leave detail" || mode == "empty find detail" {
					s.navigate(t, "/return\r", "changes loaded")
				}
				s.navigate(t, "/return\r", "MainScreen")
				if mode == "switch project" {
					s.navigate(t, "/select-project\r", "Second Project")
					s.send(t, "\x1b[B\r")
					s.waitFor(t, "project selection saved")
					s.navigate(t, "/changes\r", "Current change")
					s.navigate(t, "/return\r", "MainScreen")
				}
				s.send(t, "/quit\r")
				require.NoError(t, s.waitDone(t))
			}
			select {
			case <-canceled:
			case <-time.After(time.Second):
				t.Fatal("obsolete change HTTP remained active")
			}
		})
	}
}

func TestCLIProgramChangeReloadBlocksCachedRows(t *testing.T) {
	var mu sync.Mutex
	lists, details := 0, 0
	started, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		switch r.URL.Path {
		case "/api/v1/doc/comment-list":
			writeProgramJSON(w, []any{})
		case "/api/v1/project/details":
			writeProgramJSON(w, programProject(7, "Program Project"))
		case "/api/v1/project/config":
			writeProgramJSON(w, programProjectConfig())
		case "/api/v1/change/list":
			mu.Lock()
			lists++
			n := lists
			mu.Unlock()
			if n > 1 {
				if n == 2 {
					close(started)
				}
				select {
				case <-release:
				case <-r.Context().Done():
					return
				}
			}
			writeProgramJSON(w, []any{scopedProgramChange(3, 7, "Loaded change")})
		case "/api/v1/change/details":
			mu.Lock()
			details++
			mu.Unlock()
			writeProgramJSON(w, scopedProgramChange(3, 7, "Loaded change"))
		case "/api/v1/doc/list-active", "/api/v1/test-case/list":
			writeProgramJSON(w, []any{})
		default:
			t.Errorf("unexpected route %s", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	t.Cleanup(func() { once.Do(func() { close(release) }) })
	root := t.TempDir()
	writeProgramConfig(t, root, server.URL)
	s := startProgram(t, root, "")
	s.navigate(t, "/changes\r", "Loaded change")
	s.navigate(t, "\r", "loaded change")
	s.navigate(t, "/return\r", "Changes: loading")
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("reload did not start")
	}
	s.navigate(t, "\r", "no changes selectable")
	mu.Lock()
	assert.Equal(t, 1, details)
	mu.Unlock()
	marker := s.output.count("changes loaded")
	once.Do(func() { close(release) })
	s.output.waitForCount(t, "changes loaded", marker+1)
	s.navigate(t, "\r", "Loaded change")
	mu.Lock()
	assert.Equal(t, 2, details)
	mu.Unlock()
	s.navigate(t, "/return\r", "changes loaded")
	s.navigate(t, "/return\r", "MainScreen")
	s.send(t, "/quit\r")
	require.NoError(t, s.waitDone(t))
}

func scopedProgramChange(id, project int, title string) map[string]any {
	v := programChange(id, title)
	v["project_id"] = project
	return v
}

func TestCLIProgramChangeMalformedReadRecovery(t *testing.T) {
	var lists int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/doc/comment-list":
			writeProgramJSON(w, []any{})
		case "/api/v1/project/details":
			writeProgramJSON(w, programProject(7, "Program Project"))
		case "/api/v1/project/config":
			writeProgramJSON(w, programProjectConfig())
		case "/api/v1/change/list":
			lists++
			if lists == 1 {
				writeProgramJSON(w, []any{map[string]any{"id": "bad"}})
			} else {
				writeProgramJSON(w, []any{programChange(12, "Recovered change")})
			}
		default:
			t.Errorf("unexpected %s", r.URL.Path)
		}
	}))
	defer server.Close()
	root := t.TempDir()
	writeProgramConfig(t, root, server.URL)
	s := startProgram(t, root, "")
	s.navigate(t, "/changes\r", "changeWire.id of type int")
	s.navigate(t, "/return\r", "MainScreen")
	s.navigate(t, "/changes\r", "Recovered change")
	s.finishFromChanges(t)
	require.Equal(t, 2, lists)
}
