package integration_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCLIProgramHealthRoutesAndDegradedStatus(t *testing.T) {
	var mu sync.Mutex
	calls := []string{}
	v1Checks := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls = append(calls, r.Method+" "+r.URL.Path)
		mu.Unlock()
		switch r.URL.Path {
		case "/api/v1/project/config":
			writeProgramJSON(w, programProjectConfig())
		case "/api/v1/project/details":
			writeProgramJSON(w, programProject(7, "Program Project"))
		case "/api/v1/health":
			v1Checks++
			if v1Checks == 2 {
				http.Error(w, "health transport failed", 500)
				return
			}
			writeProgramJSON(w, map[string]string{"status": "ok", "api": "ok", "database": "ok"})
		case "/api/health":
			w.WriteHeader(503)
			writeProgramJSON(w, map[string]string{"status": "degraded", "api": "ok", "database": "error", "error": "database unavailable"})
		default:
			t.Errorf("unexpected request %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	root := t.TempDir()
	writeProgramConfig(t, root, server.URL)
	s := startProgram(t, root, "")
	s.navigate(t, "/health\r", "HealthScreen")
	s.waitFor(t, "HTTP 200")
	s.navigate(t, "/retry\r", "Refresh failed: /api/v1/health: 500")
	assert.Contains(t, s.output.String(), "HTTP 200")
	s.navigate(t, "/", "Check the v1 health route")
	s.navigate(t, "\x1b[B\r", "HTTP 503")
	s.waitFor(t, "database unavailable")
	s.send(t, "/retry\r")
	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		n := 0
		for _, call := range calls {
			if call == "GET /api/health" {
				n++
			}
		}
		return n == 2
	}, 2*time.Second, 10*time.Millisecond)
	s.navigate(t, "/", "Check the v1 health route")
	s.navigate(t, "\r", "HTTP 200")
	s.navigate(t, "/return\r", "MainScreen")
	s.send(t, "/quit\r")
	require.NoError(t, s.waitDone(t))
	mu.Lock()
	defer mu.Unlock()
	assert.Contains(t, calls, "GET /api/v1/health")
	assert.Contains(t, calls, "GET /api/health")
}

func TestCLIProgramConfigurationCRUDAndCatalogRefresh(t *testing.T) {
	var mu sync.Mutex
	rows := map[string]map[string]any{"program": programProjectConfig()}
	calls := map[string]int{}
	failDetails := 0
	insertPayload := map[string]any{
		"slug": "new", "project_docs": []any{"guide", "readme"}, "epic_docs": []any{},
		"change_docs": []any{"brief", "spec"}, "change_phases": []any{"todo"},
		"change_colors": []any{}, "change_types": []any{},
	}
	updatePayload := map[string]any{
		"slug": "new", "project_docs": []any{}, "epic_docs": []any{"epic-v2"},
		"change_docs": []any{}, "change_phases": []any{"review"},
		"change_colors": []any{"red"}, "change_types": []any{"fix"},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		calls[r.URL.Path]++
		switch r.URL.Path {
		case "/api/v1/project/config":
			writeProgramJSON(w, rows["program"])
		case "/api/v1/project/details":
			writeProgramJSON(w, programProject(7, "Program Project"))
		case "/api/v1/config/list":
			list := []map[string]any{}
			if row, ok := rows["new"]; ok {
				list = append(list, row)
			}
			if row, ok := rows["program"]; ok {
				list = append(list, row)
			}
			writeProgramJSON(w, list)
		case "/api/v1/config/details":
			var body struct {
				Slug string `json:"slug"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			if failDetails > 0 {
				failDetails--
				http.Error(w, "read unavailable", http.StatusServiceUnavailable)
				return
			}
			writeProgramJSON(w, rows[body.Slug])
		case "/api/v1/config/insert":
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, insertPayload, body)
			slug := body["slug"].(string)
			rows[slug] = body
			w.WriteHeader(201)
			writeProgramJSON(w, map[string]string{"slug": slug})
		case "/api/v1/config/update":
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, updatePayload, body)
			rows[body["slug"].(string)] = body
			failDetails = 2
			w.WriteHeader(204)
		case "/api/v1/config/delete":
			var body struct {
				Slug string `json:"slug"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			if body.Slug == "program" {
				http.Error(w, "config is in use", http.StatusConflict)
				return
			}
			delete(rows, body.Slug)
			w.WriteHeader(204)
		default:
			t.Errorf("unexpected route %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	root := t.TempDir()
	writeProgramConfig(t, root, server.URL)
	s := startProgram(t, root, "")
	s.navigate(t, "/backend-configs\r", "loaded configurations")
	s.navigate(t, "/new-config\r", "BackendConfigFormScreen")
	s.send(t, "new\t\x15[\"guide\",\"readme\"]\t\t\x15[\"brief\",\"spec\"]\t\x15[\"todo\"]\t\t\x13")
	s.waitFor(t, "created configuration new")
	s.navigate(t, "/", "Edit this item")
	s.navigate(t, "\r", "BackendConfigFormScreen")
	s.waitFor(t, "slug: new (fixed)")
	// Backward navigation from the first editable field wraps to change_types,
	// so a would-be replacement slug cannot be entered or submitted there.
	s.send(t, "\x1b[Z\x15renamed\x13")
	s.waitFor(t, "change_types must be a JSON string array")
	mu.Lock()
	assert.Zero(t, calls["/api/v1/config/update"])
	mu.Unlock()
	s.send(t, "\x15[]\t\x15[]\t\x15[\"epic-v2\"]\t\x15[]\t\x15[\"review\"]\t\x15[\"red\"]\t\x15[\"fix\"]\x13")
	s.waitFor(t, "/api/v1/config/details: 503")
	s.send(t, "/retry\r")
	require.Eventually(t, func() bool { mu.Lock(); defer mu.Unlock(); return calls["/api/v1/config/details"] >= 3 }, 2*time.Second, 10*time.Millisecond)
	s.send(t, "/retry\r")
	s.waitFor(t, "change_types: [\"fix\"]")
	s.navigate(t, "/delete\r", "Delete configuration new?")
	s.send(t, "\r")
	s.waitFor(t, "deleted configuration new")
	s.navigate(t, "\r", "Configuration: program")
	s.navigate(t, "/delete\r", "Delete configuration program?")
	s.send(t, "\r")
	s.waitFor(t, "/api/v1/config/delete: 409")
	s.navigate(t, "\x1b", "BackendConfigDetailsScreen")
	s.navigate(t, "/return\r", "BackendConfigListScreen")
	s.navigate(t, "/return\r", "MainScreen")
	s.send(t, "/quit\r")
	require.NoError(t, s.waitDone(t))
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 1, calls["/api/v1/config/insert"])
	assert.Equal(t, 1, calls["/api/v1/config/update"])
	assert.Equal(t, 2, calls["/api/v1/config/delete"])
	assert.GreaterOrEqual(t, calls["/api/v1/config/details"], 4)
}

func TestCLIProgramConfigurationSelectedProjectCatalogRefresh(t *testing.T) {
	var mu sync.Mutex
	cfg := programProjectConfig()
	configReads := 0
	updates := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch r.URL.Path {
		case "/api/v1/project/details":
			writeProgramJSON(w, programProject(7, "Program Project"))
		case "/api/v1/project/config":
			configReads++
			writeProgramJSON(w, cfg)
		case "/api/v1/config/list":
			writeProgramJSON(w, []map[string]any{cfg})
		case "/api/v1/config/details":
			writeProgramJSON(w, cfg)
		case "/api/v1/config/update":
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, "program", body["slug"])
			assert.Equal(t, []any{"newphase"}, body["change_phases"])
			cfg = body
			updates++
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
	s := startProgram(t, root, "")
	s.navigate(t, "/backend-configs\r", "loaded configurations")
	s.navigate(t, "\r", "Configuration: program")
	s.navigate(t, "/edit\r", "BackendConfigFormScreen")
	s.send(t, "\t\t\t\x15[\"newphase\"]\x13")
	s.waitFor(t, "change_phases: [\"newphase\"]")
	require.Eventually(t, func() bool { mu.Lock(); defer mu.Unlock(); return configReads >= 2 }, 2*time.Second, 10*time.Millisecond)
	s.navigate(t, "/return\r", "BackendConfigListScreen")
	s.navigate(t, "/return\r", "MainScreen")
	s.navigate(t, "/changes\r", "ChangesListScreen")
	s.navigate(t, "/phase-filter\r", "newphase")
	(<-s.controller).Quit()
	require.NoError(t, s.waitDone(t))
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 1, updates)
}

func TestCLIProgramConfigurationStaleResponseIsolation(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	finished := make(chan struct{})
	row := programProjectConfig()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/project/config":
			writeProgramJSON(w, row)
		case "/api/v1/project/details":
			writeProgramJSON(w, programProject(7, "Program Project"))
		case "/api/v1/config/list":
			writeProgramJSON(w, []map[string]any{row})
		case "/api/v1/config/details":
			close(started)
			<-release
			writeProgramJSON(w, row)
			close(finished)
		case "/api/v1/health":
			writeProgramJSON(w, map[string]string{"status": "ok", "api": "ok", "database": "ok"})
		default:
			t.Errorf("unexpected route %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	root := t.TempDir()
	writeProgramConfig(t, root, server.URL)
	s := startProgram(t, root, "")
	s.navigate(t, "/backend-configs\r", "loaded configurations")
	s.navigate(t, "\r", "BackendConfigDetailsScreen")
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("details did not start")
	}
	s.navigate(t, "\x1b", "MainScreen")
	close(release)
	select {
	case <-finished:
	case <-time.After(2 * time.Second):
		t.Fatal("details did not finish")
	}
	s.navigate(t, "/health\r", "HTTP 200")
	s.navigate(t, "/return\r", "MainScreen")
	s.send(t, "/quit\r")
	require.NoError(t, s.waitDone(t))
	assert.NotContains(t, s.output.String(), "Configuration: program")
}
