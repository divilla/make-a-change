package integration_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func programEpic(id, projectID int, name string) map[string]any {
	return map[string]any{"id": id, "project_id": projectID, "name": name, "active": true, "done_tc": 2, "total_tc": 8, "completed": 63, "change_count": 4, "created_at": "2026-09-28T10:00:00Z", "updated_at": "2026-09-28T11:00:00Z"}
}

func TestCLIProgramEpicCRUDAndPartialSuccess(t *testing.T) {
	for _, mode := range []string{"success", "refresh failures", "write failures", "literal name"} {
		t.Run(mode, func(t *testing.T) {
			var mu sync.Mutex
			calls := map[string]int{}
			names := []string{}
			exists := false
			name := ""
			unrelatedName := "Unrelated epic"
			if mode == "success" {
				unrelatedName += "\n" + strings.Repeat("long name line\n", 40) + "last name line"
			}
			failDetail, failList := false, false
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				calls[r.URL.Path]++
				var body struct {
					ID        int    `json:"id"`
					ProjectID int    `json:"project_id"`
					Name      string `json:"name"`
				}
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				switch r.URL.Path {
				case "/api/v1/doc/comment-list":
					writeProgramJSON(w, []any{})
				case "/api/v1/project/details":
					writeProgramJSON(w, programProject(7, "Program Project"))
				case "/api/v1/project/config":
					writeProgramJSON(w, programProjectConfig())
				case "/api/v1/epic/list":
					assert.Equal(t, 7, body.ProjectID)
					if failList {
						failList = false
						http.Error(w, "list refresh unavailable", http.StatusServiceUnavailable)
						return
					}
					rows := []map[string]any{programEpic(10, 7, unrelatedName)}
					if exists {
						rows = append(rows, programEpic(9, 7, name))
					}
					writeProgramJSON(w, rows)
				case "/api/v1/epic/details":
					if failDetail {
						failDetail = false
						http.Error(w, "details refresh unavailable", http.StatusServiceUnavailable)
						return
					}
					if body.ID == 10 {
						writeProgramJSON(w, programEpic(10, 7, unrelatedName))
						return
					}
					assert.Equal(t, 9, body.ID)
					writeProgramJSON(w, programEpic(9, 7, name))
				case "/api/v1/epic/create", "/api/v1/epic/update":
					names = append(names, body.Name)
					if mode == "write failures" && calls[r.URL.Path] == 1 {
						http.Error(w, "write rejected", http.StatusInternalServerError)
						return
					}
					name = body.Name
					exists = true
					failDetail = mode == "refresh failures"
					if strings.HasSuffix(r.URL.Path, "create") {
						assert.Equal(t, 7, body.ProjectID)
						w.WriteHeader(201)
						writeProgramJSON(w, map[string]int{"id": 9})
					} else {
						assert.Equal(t, 9, body.ID)
						w.WriteHeader(204)
					}
				case "/api/v1/epic/delete":
					assert.Equal(t, 9, body.ID)
					if mode == "write failures" && calls[r.URL.Path] == 1 {
						http.Error(w, "delete rejected", http.StatusInternalServerError)
						return
					}
					exists = false
					failList = mode == "refresh failures"
					w.WriteHeader(204)
				case "/api/v1/change/list":
					writeProgramJSON(w, []any{})
				default:
					t.Errorf("unexpected route %s", r.URL.Path)
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			root := t.TempDir()
			writeProgramConfig(t, root, server.URL)
			raw := "/cancel\n\tEpic editor name\n"
			nameMarker := "Epic editor name"
			if mode == "literal name" {
				raw = "/save"
				nameMarker = raw
			}
			s := startProgram(t, root, raw)
			s.navigate(t, "/epics\r", "Unrelated epic")
			s.navigate(t, "/help\r", "/delete asks for confirmation")
			s.navigate(t, "/return\r", "loaded epics")
			s.navigate(t, "/new-epic\r", "EpicCreateScreen")
			s.navigate(t, "\r", "epic name is required")
			s.navigate(t, "New epic\r", map[bool]string{true: "save failed", false: "saved epic"}[mode == "write failures"])
			if mode == "write failures" {
				s.navigate(t, "\r", "saved epic")
			}
			if mode == "refresh failures" {
				s.waitFor(t, "saved epic; refresh failed")
				s.navigate(t, "/retry\r", "loaded epic")
			}
			for _, field := range []string{"ID: 9", "Project ID: 7", "Name: New epic", "Done TC: 2", "Total TC: 8", "Completed: 63", "Changes: 4", "Created:", "Modified:"} {
				s.waitFor(t, field)
			}
			s.navigate(t, "/edit\r", "EpicUpdateScreen")
			s.navigate(t, "\x05", map[bool]string{true: "save failed", false: "saved epic"}[mode == "write failures"])
			if mode == "write failures" {
				s.navigate(t, "\x05", "saved epic")
			}
			if mode == "refresh failures" {
				s.navigate(t, "/retry\r", "loaded epic")
			}
			s.waitFor(t, nameMarker)
			assert.Contains(t, s.output.String(), "\x1b[2J")
			s.navigate(t, "/edit\r", "EpicUpdateScreen")
			s.navigate(t, "\x05", "unchanged")
			s.navigate(t, "/edit\r", "EpicUpdateScreen")
			s.navigate(t, "\r", "unchanged")
			s.navigate(t, "/delete\r", "Are you sure?")
			s.navigate(t, "\x1b", "status cancel")
			s.navigate(t, "/delete\r", "Are you sure?")
			s.navigate(t, "\r", map[bool]string{true: "delete failed", false: map[bool]string{true: "epic delete committed", false: "deleted epic"}[mode == "refresh failures"]}[mode == "write failures"])
			if mode == "write failures" {
				s.navigate(t, "/delete\r", "Are you sure?")
				s.navigate(t, "\r", "deleted epic")
			}
			if mode == "refresh failures" {
				s.waitFor(t, "epic delete committed; refresh failed")
				s.navigate(t, "/retry\r", "loaded epics")
			}
			s.waitFor(t, "Unrelated epic")
			mu.Lock()
			failDetail = true
			mu.Unlock()
			s.navigate(t, "\r", "status load failed")
			s.navigate(t, "/retry\r", "status loaded epic")
			s.waitFor(t, "Name: Unrelated epic")
			if mode == "success" {
				// Scroll a multiline detail through the actual event loop, then
				// recover the identity fields at the top with page-up.
				s.navigate(t, strings.Repeat("\x1b[6~", 8), "Modified:")
				s.waitFor(t, "last name line")
				s.navigate(t, strings.Repeat("\x1b[5~", 8), "ID: 10")
				s.waitFor(t, "Project ID: 7")
			}
			s.navigate(t, "/return\r", "loaded epics")
			s.navigate(t, "/return\r", "MainScreen")
			s.navigate(t, "/changes\r", "no changes")
			s.navigate(t, "/epic-filter\r", "Unrelated epic")
			s.navigate(t, "\r", "status selected")
			s.finishFromChanges(t)
			mu.Lock()
			defer mu.Unlock()
			attempts := 1
			if mode == "write failures" {
				attempts = 2
			}
			for _, op := range []string{"create", "update", "delete"} {
				assert.Equal(t, attempts, calls["/api/v1/epic/"+op])
			}
			expected := []string{"New epic", raw}
			if attempts == 2 {
				expected = []string{"New epic", "New epic", raw, raw}
			}
			assert.Equal(t, expected, names)
		})
	}
}

func TestCLIProgramEpicDelayedScopeAndShutdown(t *testing.T) {
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
				case "/api/v1/epic/list":
					if mode == "leave detail" || mode == "empty find detail" || body.ProjectID == 8 || (mode == "empty find list" && readStarted.Swap(true)) {
						writeProgramJSON(w, []any{programEpic(3, body.ProjectID, "Current epic")})
						return
					}
					close(started)
					<-r.Context().Done()
					close(canceled)
				case "/api/v1/epic/details":
					if mode == "empty find detail" && readStarted.Swap(true) {
						writeProgramJSON(w, programEpic(body.ID, 7, "Current epic"))
						return
					}
					close(started)
					<-r.Context().Done()
					close(canceled)
				default:
					t.Errorf("unexpected route %s", r.URL.Path)
				}
			}))
			defer server.Close()
			root := t.TempDir()
			writeProgramConfig(t, root, server.URL)
			s := startProgram(t, root, "")
			if mode == "leave detail" || mode == "empty find detail" {
				s.navigate(t, "/epics\r", "Current epic")
				s.navigate(t, "\r", "Loading epic")
			} else {
				s.navigate(t, "/epics\r", "Epics: loading")
			}
			select {
			case <-started:
			case <-time.After(time.Second):
				t.Fatal("epic read did not start")
			}
			if mode == "shutdown" {
				controller := <-s.controller
				controller.Quit()
				require.NoError(t, s.waitDone(t))
			} else {
				if strings.HasPrefix(mode, "empty find") {
					s.navigate(t, "/find\r", "FindInputScreen")
					if mode == "empty find detail" {
						s.navigate(t, "\r", "Name: Current epic")
					} else {
						s.navigate(t, "\r", "Current epic")
					}
				}
				if mode == "leave detail" || mode == "empty find detail" {
					s.navigate(t, "/return\r", "loaded epics")
				}
				s.navigate(t, "/return\r", "MainScreen")
				if mode == "switch project" {
					s.navigate(t, "/select-project\r", "Second Project")
					s.send(t, "\x1b[B\r")
					s.waitFor(t, "project selection saved")
					s.navigate(t, "/epics\r", "Current epic")
					s.navigate(t, "/return\r", "MainScreen")
				}
				s.send(t, "/quit\r")
				require.NoError(t, s.waitDone(t))
			}
			select {
			case <-canceled:
			case <-time.After(time.Second):
				t.Fatal("obsolete epic HTTP remained active")
			}
		})
	}
}

func TestCLIProgramEpicReloadBlocksCachedRows(t *testing.T) {
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
		case "/api/v1/epic/list":
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
			writeProgramJSON(w, []any{programEpic(3, 7, "Loaded epic")})
		case "/api/v1/epic/details":
			mu.Lock()
			details++
			mu.Unlock()
			writeProgramJSON(w, programEpic(3, 7, "Loaded epic"))
		default:
			t.Errorf("unexpected route %s", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	t.Cleanup(func() { once.Do(func() { close(release) }) })
	root := t.TempDir()
	writeProgramConfig(t, root, server.URL)
	s := startProgram(t, root, "")
	s.navigate(t, "/epics\r", "Loaded epic")
	s.navigate(t, "\r", "loaded epic")
	s.navigate(t, "/return\r", "Epics: loading")
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("reload did not start")
	}
	s.navigate(t, "\r", "no epics selectable")
	mu.Lock()
	assert.Equal(t, 1, details)
	mu.Unlock()
	marker := s.output.count("loaded epics")
	once.Do(func() { close(release) })
	s.output.waitForCount(t, "loaded epics", marker+1)
	s.navigate(t, "\r", "Name: Loaded epic")
	mu.Lock()
	assert.Equal(t, 2, details)
	mu.Unlock()
	s.navigate(t, "/return\r", "loaded epics")
	s.navigate(t, "/return\r", "MainScreen")
	s.send(t, "/quit\r")
	require.NoError(t, s.waitDone(t))
}
