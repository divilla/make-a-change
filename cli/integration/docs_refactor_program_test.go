package integration_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type docs031Backend struct {
	mu       sync.Mutex
	rows     []map[string]any
	active   map[string]int
	calls    map[string]int
	writes   []map[string]any
	inactive bool
}

func docs031Server(t *testing.T) (*docs031Backend, *httptest.Server) {
	t.Helper()
	b := &docs031Backend{active: map[string]int{"brief": 8, "spec": 9}, calls: map[string]int{}, inactive: true}
	b.rows = []map[string]any{programDocVersion(10, "change", 12, "spec", "# Newest retained\n\n```go\nfunc newest() {}\n```\n"+strings.Repeat("newest body line\n", 35), false), programDocVersion(9, "change", 12, "spec", "# Selected active\n\n```go\nfunc active() {}\n```", true), programDocVersion(8, "change", 12, "brief", strings.Repeat("brief preview line\n", 20)+"full brief tail", true), programDocVersion(7, "change", 12, "comment", "one\ntwo\nthree\nfull comment tail", false), programDocVersion(6, "change", 12, "comment", "deleted retained comment", false)}
	b.rows[0]["deleted_at"] = "2026-09-28T12:00:00Z"
	b.rows[4]["deleted_at"] = "2026-09-28T12:00:00Z"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b.mu.Lock()
		defer b.mu.Unlock()
		b.calls[r.URL.Path]++
		var in map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&in))
		switch r.URL.Path {
		case "/api/v1/project/details":
			writeProgramJSON(w, programProject(7, "Program Project"))
		case "/api/v1/project/config":
			cfg := programProjectConfig()
			cfg["change_docs"] = []string{"brief", "spec", "notes"}
			writeProgramJSON(w, cfg)
		case "/api/v1/change/list":
			writeProgramJSON(w, []any{programChange(12, "Program Change")})
		case "/api/v1/change/list-inactive":
			require.Equal(t, float64(7), in["project_id"])
			if b.inactive {
				row := programChange(20, "Inactive change")
				delete(row, "active")
				writeProgramJSON(w, []any{row})
			} else {
				writeProgramJSON(w, []any{})
			}
		case "/api/v1/change/update-active":
			require.Equal(t, float64(20), in["id"])
			require.Equal(t, true, in["active"])
			b.inactive = false
			b.writes = append(b.writes, in)
			w.WriteHeader(204)
		case "/api/v1/change/details":
			writeProgramJSON(w, programChange(12, "Program Change"))
		case "/api/v1/test-case/list":
			writeProgramJSON(w, []any{})
		case "/api/v1/epic/list":
			active := programEpic(3, 7, "Active epic")
			inactive := programEpic(4, 7, "Hidden inactive epic")
			inactive["active"] = false
			writeProgramJSON(w, []any{active, inactive})
		case "/api/v1/doc/list", "/api/v1/doc/list-active", "/api/v1/doc/comment-list":
			require.Equal(t, "change", in["ref_table"])
			require.Equal(t, float64(12), in["ref_id"])
			rows := []map[string]any{}
			for _, d := range b.rows {
				kind := d["doc_type"].(string)
				id := d["id"].(int)
				if r.URL.Path == "/api/v1/doc/list" || (r.URL.Path == "/api/v1/doc/comment-list" && kind == "comment") || (r.URL.Path == "/api/v1/doc/list-active" && b.active[kind] == id) {
					rows = append(rows, d)
				}
			}
			writeProgramJSON(w, rows)
		case "/api/v1/doc/insert", "/api/v1/doc/comment-insert":
			require.Equal(t, "change", in["ref_table"])
			require.Equal(t, float64(12), in["ref_id"])
			require.Equal(t, false, in["agent_edit"])
			kind := "comment"
			if r.URL.Path == "/api/v1/doc/insert" {
				kind = in["doc_type"].(string)
				require.NotEqual(t, "comment", kind)
			}
			id := 100 + len(b.writes)
			b.writes = append(b.writes, in)
			d := programDocVersion(id, "change", 12, kind, in["body"].(string), false)
			b.rows = append([]map[string]any{d}, b.rows...)
			if kind != "comment" {
				b.active[kind] = id
			}
			w.WriteHeader(201)
			writeProgramJSON(w, map[string]int{"id": id})
		case "/api/v1/doc/comment-update", "/api/v1/doc/delete", "/api/v1/doc/active-set", "/api/v1/doc/comment-undelete":
			id := intFromJSON(in["id"])
			b.writes = append(b.writes, in)
			for _, d := range b.rows {
				if d["id"].(int) == id {
					kind := d["doc_type"].(string)
					switch r.URL.Path {
					case "/api/v1/doc/comment-update":
						require.Equal(t, "comment", kind)
						body, exists := in["body"]
						require.True(t, exists)
						d["body"] = body
					case "/api/v1/doc/delete":
						d["deleted_at"] = "2026-09-28T12:00:00Z"
						if b.active[kind] == id {
							delete(b.active, kind)
						}
					case "/api/v1/doc/active-set":
						require.NotEqual(t, "comment", kind)
						b.active[kind] = id
						d["deleted_at"] = nil
					case "/api/v1/doc/comment-undelete":
						require.Equal(t, "comment", kind)
						d["deleted_at"] = nil
					}
					w.WriteHeader(204)
					return
				}
			}
			http.Error(w, "missing", 404)
		default:
			t.Errorf("unexpected 031 route %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return b, server
}

func TestCLIProgram031DocumentCommentsAndHistory(t *testing.T) {
	b, server := docs031Server(t)
	root := t.TempDir()
	writeProgramConfig(t, root, server.URL)
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	s := startProgram(t, root, "new independent comment")
	capture := filepath.Join(root, "editor-seed.md")
	edited := filepath.Join(root, "edited.md")
	script := filepath.Join(root, "editor.sh")
	require.NoError(t, os.WriteFile(edited, []byte("new independent comment"), 0o600))
	require.NoError(t, os.WriteFile(script, []byte("#!/bin/sh\ncp \"$1\" \"$SEED\"\ncp \"$EDITED\" \"$1\"\n"), 0o700))
	t.Setenv("SEED", capture)
	t.Setenv("EDITED", edited)
	t.Setenv("EDITOR", script)
	s.navigate(t, "/changes\r", "Rows 1-1 of 1")
	s.navigate(t, "\r", "loaded change")
	s.navigate(t, "/new-comment\r", "committed comment-insert document #100")
	require.NoFileExists(t, filepath.Join(tmp, "mch-history-"))
	s.send(t, strings.Repeat("\x1b[B", 12))
	s.waitFor(t, "new independent comment")
	// Select the new comment by walking from fixed identity: 8 editable fields,
	// two document slots, the Comments heading, then the newest comment.
	s.navigate(t, "\r", "committed comment-update document #100")
	require.NoError(t, os.WriteFile(edited, []byte(""), 0o600))
	s.navigate(t, "\r", "committed comment-update document #100")
	// Delete opens the bottom confirmation; cancellation never sends a mutation.
	s.navigate(t, "\x1b[3~", "Are you sure?")
	s.navigate(t, "\x1b", "status cancel")
	b.mu.Lock()
	require.Zero(t, b.calls["/api/v1/doc/delete"])
	b.mu.Unlock()
	s.navigate(t, "\x1b[3~", "Are you sure?")
	s.navigate(t, "\r", "committed delete document #100")
	// Comments heading remains a history entry even after selecting a deleted comment.
	s.send(t, "\x1b[A\x08")
	s.waitFor(t, "Space undelete comment")
	s.waitFor(t, "deleted_at:")
	s.navigate(t, " ", "committed undelete document #100")
	s.navigate(t, "\x03", "returned from history")
	// Spec slot navigation starts at the newest retained ID rather than the active ID.
	s.send(t, strings.Repeat("\x1b[A", 2)+"\x08")
	s.waitFor(t, "Newest retained")
	s.waitFor(t, "created_at:")
	s.waitFor(t, "deleted_at:")
	s.navigate(t, "\x1b[C", "Selected active")
	s.navigate(t, "\x1b[D", "Newest retained")
	s.navigate(t, "\x1b[6~", "newest body line")
	s.navigate(t, " ", "committed active selection document #10")
	s.navigate(t, "\x1b", "returned from history")
	// Confirmed deletion empties the slot; no historical fallback is selected.
	s.navigate(t, "\x1b[3~", "Are you sure?")
	s.navigate(t, "\r", "committed delete document #10")
	s.waitFor(t, "spec [ ]")
	s.navigate(t, "/return\r", "Rows")
	s.navigate(t, "/return\r", "MainScreen")
	s.send(t, "/quit\r")
	require.NoError(t, s.waitDone(t))
	b.mu.Lock()
	defer b.mu.Unlock()
	require.Equal(t, 1, b.calls["/api/v1/doc/comment-insert"])
	require.GreaterOrEqual(t, b.calls["/api/v1/doc/comment-update"], 2)
	require.Zero(t, b.calls["/api/v1/doc/insert"])
	require.Equal(t, 1, b.calls["/api/v1/doc/comment-undelete"])
	require.Equal(t, 1, b.calls["/api/v1/doc/active-set"])
	require.True(t, slices.ContainsFunc(b.rows, func(d map[string]any) bool { return d["id"] == 100 && d["body"] == "" && d["deleted_at"] == nil }))
	entries, err := os.ReadDir(tmp)
	require.NoError(t, err)
	for _, e := range entries {
		require.NotContains(t, e.Name(), "mch-history-")
	}
}

func TestCLIProgram031InactiveChangesAndEpicSelection(t *testing.T) {
	b, server := docs031Server(t)
	root := t.TempDir()
	writeProgramConfig(t, root, server.URL)
	s := startProgram(t, root, "unused")
	s.navigate(t, "/changes\r", "Rows 1-1 of 1")
	s.navigate(t, "\x08", "Inactive change")
	s.waitFor(t, "Space activate")
	s.navigate(t, " ", "activated change #20")
	s.waitFor(t, "No changes")
	s.navigate(t, "\x03", "returned from inactive changes")
	s.navigate(t, "\r", "loaded change")
	s.navigate(t, "/epic\r", "Active epic #3")
	require.NotContains(t, s.output.String(), "Hidden inactive epic")
	s.navigate(t, "\x1b", "status cancel")
	s.navigate(t, "/return\r", "Rows")
	s.navigate(t, "/return\r", "MainScreen")
	s.send(t, "/quit\r")
	require.NoError(t, s.waitDone(t))
	b.mu.Lock()
	defer b.mu.Unlock()
	require.Equal(t, 1, b.calls["/api/v1/change/update-active"])
	require.GreaterOrEqual(t, b.calls["/api/v1/change/list-inactive"], 2)
}

func TestCLIProgram031BatFailureKeepsHistoryAndReturns(t *testing.T) {
	b, server := docs031Server(t)
	root := t.TempDir()
	writeProgramConfig(t, root, server.URL)
	bin := filepath.Join(root, "bin")
	require.NoError(t, os.Mkdir(bin, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(bin, "bat"), []byte("#!/bin/sh\nexit 31\n"), 0o700))
	t.Setenv("PATH", bin+":"+os.Getenv("PATH"))
	s := startProgram(t, root, "unused")
	s.navigate(t, "/changes\r", "Rows")
	s.navigate(t, "\r", "loaded change")
	s.send(t, strings.Repeat("\x1b[B", 9)+"\x08")
	s.waitFor(t, "history bat:")
	s.waitFor(t, "deleted_at:")
	s.send(t, "/retry\r")
	require.Eventually(t, func() bool { b.mu.Lock(); defer b.mu.Unlock(); return b.calls["/api/v1/doc/list"] >= 2 }, 2*time.Second, 10*time.Millisecond)
	s.navigate(t, "\x03", "returned from history")
	s.navigate(t, "/return\r", "Rows")
	s.navigate(t, "/return\r", "MainScreen")
	s.send(t, "/quit\r")
	require.NoError(t, s.waitDone(t))
}
