package integration_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCLIProgram034EpicAndChangeActivity(t *testing.T) {
	var mu sync.Mutex
	epics := []map[string]any{programEpic(3, 7, "Unreferenced"), programEpic(4, 7, "Active change epic"), programEpic(5, 7, "Inactive change epic")}
	epics[0]["change_count"] = 0
	changes := []map[string]any{programChange(12, "Active change"), programChange(13, "Inactive change")}
	changes[1]["id"] = 13
	changes[1]["active"] = false
	calls := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		calls[r.URL.Path]++
		var in map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&in))
		switch r.URL.Path {
		case "/api/v1/project/details":
			writeProgramJSON(w, programProject(7, "Program Project"))
		case "/api/v1/project/config":
			writeProgramJSON(w, programProjectConfig())
		case "/api/v1/epic/list":
			require.Equal(t, map[string]any{"project_id": float64(7)}, in)
			writeProgramJSON(w, epics)
		case "/api/v1/epic/delete":
			id := int(in["id"].(float64))
			require.Len(t, in, 1)
			epics = slices.DeleteFunc(epics, func(e map[string]any) bool { return e["id"] == id && e["change_count"] == 0 })
			for _, e := range epics {
				if e["id"] == id {
					e["active"] = false
				}
			}
			w.WriteHeader(204)
		case "/api/v1/epic/update-active":
			require.Len(t, in, 2)
			require.IsType(t, true, in["active"])
			for _, e := range epics {
				if e["id"] == int(in["id"].(float64)) {
					e["active"] = in["active"]
				}
			}
			w.WriteHeader(204)
		case "/api/v1/change/list":
			require.Len(t, in, 2)
			require.Equal(t, float64(7), in["project_id"])
			require.IsType(t, true, in["active"])
			rows := []map[string]any{}
			for _, c := range changes {
				if c["active"] == in["active"] {
					row := map[string]any{}
					for k, v := range c {
						if k != "active" {
							row[k] = v
						}
					}
					rows = append(rows, row)
				}
			}
			writeProgramJSON(w, rows)
		case "/api/v1/change/update-active":
			require.Len(t, in, 2)
			require.IsType(t, true, in["active"])
			for _, c := range changes {
				if c["id"] == int(in["id"].(float64)) {
					c["active"] = in["active"]
				}
			}
			w.WriteHeader(204)
		case "/api/v1/change/details":
			for _, c := range changes {
				if c["id"] == int(in["id"].(float64)) {
					writeProgramJSON(w, c)
					return
				}
			}
			http.NotFound(w, r)
		case "/api/v1/doc/list-active", "/api/v1/doc/comment-list", "/api/v1/test-case/list":
			writeProgramJSON(w, []any{})
		default:
			t.Errorf("unexpected route %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	root := t.TempDir()
	writeProgramConfig(t, root, server.URL)
	s := startProgram(t, root, "unused")
	s.navigate(t, "/epics\r", "Unreferenced")
	s.waitFor(t, "DoneTC Compl Chngs Active")
	s.navigate(t, "\x1b[3~", "status deleted epic")
	mu.Lock()
	require.Len(t, epics, 2)
	require.Equal(t, 1, calls["/api/v1/epic/delete"])
	mu.Unlock()
	s.navigate(t, "\x1b[3~", "epic deactivated")
	s.waitFor(t, "inactive")
	s.navigate(t, " ", "epic activity saved")
	s.navigate(t, "\x1b[B\x1b[3~", "epic deactivated")
	s.navigate(t, "/return\r", "MainScreen")
	s.navigate(t, "/changes\r", "Rows 1-1 of 1")
	s.navigate(t, "\x1b[3~", "Are you sure?")
	s.navigate(t, "\x03", "status cancel")
	s.navigate(t, "/del-change\r", "Are you sure?")
	s.navigate(t, "\r", "deactivated change #12")
	s.navigate(t, "/inactive-filter\r", "Rows 1-2 of 2")
	s.navigate(t, "\r", "loaded change")
	s.navigate(t, "/return\r", "Rows 1-2 of 2")
	s.navigate(t, "/return\r", "MainScreen")
	s.navigate(t, "/changes\r", "Rows 1-2 of 2")
	s.navigate(t, "/undel-change\r", "activated change #12")
	s.navigate(t, " ", "activated change #13")
	s.navigate(t, "/clear-filters\r", "Rows 1-2 of 2")
	s.navigate(t, "\r", "loaded change")
	s.navigate(t, "/delete\r", "Are you sure?")
	s.navigate(t, "\r", "deactivated change #12")
	s.finishFromChanges(t)
	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, 3, calls["/api/v1/epic/delete"])
	require.Equal(t, 1, calls["/api/v1/epic/update-active"])
	require.Equal(t, 4, calls["/api/v1/change/update-active"])
	require.Zero(t, calls["/api/v1/change/delete"])
	require.False(t, changes[0]["active"].(bool))
	require.True(t, changes[1]["active"].(bool))
	require.True(t, epics[0]["active"].(bool))
	require.False(t, epics[1]["active"].(bool))
}
