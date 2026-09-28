package integration_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCLIProgramProjectCRUDAndPartialSuccess(t *testing.T) {
	for _, partial := range []bool{false, true} {
		t.Run(map[bool]string{false: "selected deletion config failure", true: "selected deletion config and refresh failures"}[partial], func(t *testing.T) {
			testSelectedProjectDeletionConfigFailure(t, partial)
		})
		t.Run(map[bool]string{false: "success", true: "refresh failures"}[partial], func(t *testing.T) {
			var mu sync.Mutex
			name := "Program Project"
			exists := false
			failDetail, failList := false, false
			calls := map[string]int{}
			names := []string{}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				calls[r.URL.Path]++
				assert.Equal(t, http.MethodPost, r.Method)
				var body struct {
					ID   int    `json:"id"`
					Name string `json:"name"`
				}
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				switch r.URL.Path {
				case "/api/v1/project/list":
					if failList {
						failList = false
						http.Error(w, "list refresh unavailable", http.StatusServiceUnavailable)
						return
					}
					rows := []map[string]any{programProject(7, "Program Project")}
					if exists {
						rows = append(rows, programProject(9, name))
					}
					writeProgramJSON(w, rows)
				case "/api/v1/project/details":
					if body.ID == 9 && failDetail {
						failDetail = false
						http.Error(w, "detail refresh unavailable", http.StatusServiceUnavailable)
						return
					}
					n := "Program Project"
					if body.ID == 9 {
						n = name
					}
					writeProgramJSON(w, programProject(body.ID, n))
				case "/api/v1/project/config":
					writeProgramJSON(w, programProjectConfig())
				case "/api/v1/project/create":
					names = append(names, body.Name)
					name = body.Name
					exists = true
					failDetail = partial
					w.WriteHeader(201)
					writeProgramJSON(w, map[string]int{"id": 9})
				case "/api/v1/project/update":
					assert.Equal(t, 9, body.ID)
					names = append(names, body.Name)
					name = body.Name
					failDetail = partial
					w.WriteHeader(204)
				case "/api/v1/project/delete":
					assert.Equal(t, 9, body.ID)
					exists = false
					failList = partial
					w.WriteHeader(204)
				default:
					t.Errorf("unexpected route %s", r.URL.Path)
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			root := t.TempDir()
			writeProgramConfig(t, root, server.URL)
			s := startProgram(t, root, "Renamed project")
			s.navigate(t, "/projects\r", "ProjectsListScreen")
			s.navigate(t, "/new-project\r", "ProjectCreateScreen")
			s.send(t, "New project\r")
			if partial {
				s.waitFor(t, "saved project; refresh failed")
				s.navigate(t, "/retry\r", "loaded project")
			} else {
				s.waitFor(t, "saved project")
			}
			s.waitFor(t, "New project")
			s.navigate(t, "/project-config\r", "Change types: feature")
			for _, field := range []string{"Config", "Last ref", "Created", "Modified", "Changes", "Project docs: readme", "Epic docs: brief", "Change docs: brief, spec", "Change phases: backlog", "Change colors: 12"} {
				assert.Contains(t, s.output.String(), field)
			}
			s.navigate(t, "/edit\r", "ProjectUpdateScreen")
			marker := s.output.count("saved project")
			s.send(t, "\x05")
			s.output.waitForCount(t, "saved project", marker+1)
			if partial {
				s.navigate(t, "/retry\r", "loaded project")
			}
			s.waitFor(t, "Renamed project")
			s.navigate(t, "/delete\r", "Are you sure?")
			s.send(t, "\r")
			s.waitFor(t, "deleted project")
			if partial {
				s.waitFor(t, "deleted project; refresh failed")
				s.navigate(t, "/retry\r", "loaded projects")
			}
			s.navigate(t, "/return\r", "MainScreen")
			s.send(t, "/quit\r")
			require.NoError(t, s.waitDone(t))
			mu.Lock()
			defer mu.Unlock()
			assert.Equal(t, []string{"New project", "Renamed project"}, names)
			for _, op := range []string{"create", "update", "delete"} {
				assert.Equal(t, 1, calls["/api/v1/project/"+op])
			}
			for _, op := range []string{"list", "details", "config"} {
				assert.Positive(t, calls["/api/v1/project/"+op])
			}
		})
	}
}

func testSelectedProjectDeletionConfigFailure(t *testing.T, refreshFails bool) {
	t.Helper()
	var mu sync.Mutex
	deletes, lists := 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch r.URL.Path {
		case "/api/v1/project/details":
			writeProgramJSON(w, programProject(7, "Program Project"))
		case "/api/v1/project/config":
			writeProgramJSON(w, programProjectConfig())
		case "/api/v1/project/list":
			lists++
			if deletes == 0 {
				writeProgramJSON(w, []map[string]any{programProject(7, "Program Project")})
			} else if refreshFails && lists == 2 {
				http.Error(w, "list refresh unavailable", http.StatusServiceUnavailable)
			} else {
				writeProgramJSON(w, []any{})
			}
		case "/api/v1/project/delete":
			var body struct {
				ID int `json:"id"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, 7, body.ID)
			deletes++
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected route %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	root := t.TempDir()
	writeProgramConfig(t, root, server.URL)
	s := startProgram(t, root, "")
	s.navigate(t, "/projects\r", "loaded projects")
	s.navigate(t, "\r", "loaded project")
	s.navigate(t, "/delete\r", "Are you sure?")
	path := filepath.Join(root, ".mch/config.yaml")
	require.NoError(t, os.Rename(path, path+".original"))
	require.NoError(t, os.Mkdir(path, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(path, "keep"), []byte("keep"), 0o600))
	s.send(t, "\r")
	s.waitFor(t, "config save failed")
	s.waitFor(t, "project selection cleared in memory; failed to save project_id:")
	if refreshFails {
		s.waitFor(t, "deleted project; refresh failed")
		s.waitFor(t, "/retry reads only; config save failed")
		s.waitFor(t, "refresh unavailable")
	} else {
		s.waitFor(t, "deleted project; config save failed")
	}
	assert.NotContains(t, s.output.String(), "project selected in memory")
	s.navigate(t, "/retry\r", "no projects")
	s.navigate(t, "/return\r", "MainScreen")
	s.send(t, "/quit\r")
	require.NoError(t, s.waitDone(t))
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 1, deletes)
	assert.Equal(t, 3, lists)
	original, err := os.ReadFile(path + ".original")
	require.NoError(t, err)
	assert.Contains(t, string(original), "project_id: 7")
}

func TestCLIProgramProjectSwitchWithPendingConfig(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	finished := make(chan struct{})
	var once sync.Once
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			ID int `json:"id"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		switch r.URL.Path {
		case "/api/v1/project/details":
			name := "Program Project"
			if body.ID == 8 {
				name = "Second Project"
			}
			writeProgramJSON(w, programProject(body.ID, name))
		case "/api/v1/project/list":
			writeProgramJSON(w, []map[string]any{programProject(7, "Program Project"), programProject(8, "Second Project")})
		case "/api/v1/project/config":
			cfg := programProjectConfig()
			if body.ID == 7 {
				once.Do(func() { close(started) })
				select {
				case <-release:
				case <-r.Context().Done():
					return
				}
				cfg["change_phases"] = []string{"stale-seven"}
				defer close(finished)
			} else {
				cfg["change_phases"] = []string{"eight-phase"}
			}
			writeProgramJSON(w, cfg)
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
	s := startProgram(t, root, "")
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("config did not start")
	}
	s.navigate(t, "/select-project\r", "Second Project")
	s.send(t, "\x1b[B\r")
	s.waitFor(t, "project selection saved")
	// Observe project 8's catalog through a feature consumer before releasing 7.
	s.navigate(t, "/changes\r", "no changes")
	s.navigate(t, "/phase-filter\r", "eight-phase")
	close(release)
	<-finished
	s.navigate(t, "\x1b", "status cancel")
	s.navigate(t, "/phase-filter\r", "eight-phase")
	assert.NotContains(t, s.output.String(), "stale-seven")
	s.navigate(t, "\x1b", "status cancel")
	s.finishFromChanges(t)
}

func TestCLIProgramShutdownCancelsProjectHTTP(t *testing.T) {
	started := make(chan struct{})
	canceled := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/project/details":
			writeProgramJSON(w, programProject(7, "Program Project"))
		case "/api/v1/project/config":
			_, _ = io.Copy(io.Discard, r.Body)
			close(started)
			<-r.Context().Done()
			assert.ErrorIs(t, r.Context().Err(), context.Canceled)
			close(canceled)
		default:
			t.Errorf("unexpected route %s", r.URL.Path)
		}
	}))
	defer server.Close()
	root := t.TempDir()
	writeProgramConfig(t, root, server.URL)
	s := startProgram(t, root, "")
	<-started
	s.send(t, "/quit\r")
	require.NoError(t, s.waitDone(t))
	select {
	case <-canceled:
	case <-time.After(time.Second):
		t.Fatal("shutdown left HTTP work active")
	}
	assert.False(t, strings.Contains(s.output.String(), "AgentRunningScreen"))
}

func TestCLIProgramProjectReloadBlocksCachedSelection(t *testing.T) {
	var mu sync.Mutex
	lists := 0
	details := map[int]int{}
	started, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			ID int `json:"id"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		switch r.URL.Path {
		case "/api/v1/project/config":
			writeProgramJSON(w, programProjectConfig())
		case "/api/v1/project/details":
			mu.Lock()
			details[body.ID]++
			mu.Unlock()
			writeProgramJSON(w, programProject(body.ID, "Program Project"))
		case "/api/v1/project/list":
			mu.Lock()
			lists++
			request := lists
			mu.Unlock()
			if request == 1 {
				writeProgramJSON(w, []map[string]any{programProject(9, "Cached project")})
				return
			}
			if request == 2 {
				close(started)
			}
			select {
			case <-release:
				writeProgramJSON(w, []map[string]any{programProject(8, "Refreshed project")})
			case <-r.Context().Done():
				t.Error("list refresh canceled before release")
			}
		default:
			t.Errorf("unexpected route %s", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	t.Cleanup(func() { once.Do(func() { close(release) }) })
	root := t.TempDir()
	writeProgramConfig(t, root, server.URL)
	s := startProgram(t, root, "")
	s.navigate(t, "/projects\r", "Cached project")
	s.navigate(t, "\r", "loaded project")
	s.navigate(t, "/return\r", "Projects: loading")
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("list refresh did not start")
	}
	s.navigate(t, "\r", "no projects selectable")
	mu.Lock()
	assert.Equal(t, 1, details[9])
	mu.Unlock()
	once.Do(func() { close(release) })
	s.waitFor(t, "Refreshed project")
	s.navigate(t, "\r", "loaded project")
	mu.Lock()
	assert.Equal(t, 1, details[8])
	assert.Equal(t, 1, details[9])
	mu.Unlock()
	s.navigate(t, "/return\r", "loaded projects")
	s.navigate(t, "/return\r", "MainScreen")
	s.send(t, "/quit\r")
	require.NoError(t, s.waitDone(t))
}
