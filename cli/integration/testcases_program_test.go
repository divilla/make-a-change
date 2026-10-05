package integration_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCLIProgramTestCaseLifecycle(t *testing.T) {
	var mu sync.Mutex
	change := programChange(12, "Existing")
	rows := []map[string]any{}
	requests := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/doc/comment-list":
			writeProgramJSON(w, []any{})
		case "/api/v1/project/details":
			writeProgramJSON(w, programProject(7, "Program Project"))
		case "/api/v1/project/config":
			writeProgramJSON(w, programProjectConfig())
		case "/api/v1/change/list":
			writeProgramJSON(w, []any{change})
		case "/api/v1/change/details":
			done := 0
			for _, row := range rows {
				if row["done"] == true {
					done++
				}
			}
			change["done_tc"] = done
			change["total_tc"] = len(rows)
			writeProgramJSON(w, change)
		case "/api/v1/doc/list-active":
			writeProgramJSON(w, []any{})
		case "/api/v1/test-case/list":
			require.Equal(t, map[string]any{"change_id": float64(12)}, payload)
			requests = append(requests, "list")
			writeProgramJSON(w, rows)
		case "/api/v1/test-case/create":
			require.Equal(t, map[string]any{"change_id": float64(12), "scenario": "/delete literal"}, payload)
			requests = append(requests, "create")
			rows = append(rows, map[string]any{"id": 31, "change_id": 12, "scenario": payload["scenario"], "done": false, "created_at": "2026-09-28T10:00:00Z", "updated_at": "2026-09-28T10:00:00Z"})
			w.WriteHeader(201)
			writeProgramJSON(w, map[string]any{"id": 31})
		case "/api/v1/test-case/update":
			require.Equal(t, map[string]any{"id": float64(31), "scenario": "Edited scenario"}, payload)
			requests = append(requests, "update")
			rows[0]["scenario"] = payload["scenario"]
			w.WriteHeader(204)
		case "/api/v1/test-case/update-done":
			require.Equal(t, float64(31), payload["id"])
			requests = append(requests, "done")
			rows[0]["done"] = payload["done"]
			w.WriteHeader(204)
		case "/api/v1/test-case/delete":
			require.Equal(t, map[string]any{"id": float64(31)}, payload)
			requests = append(requests, "delete")
			rows = nil
			w.WriteHeader(204)
		default:
			t.Errorf("unexpected route %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	root := t.TempDir()
	writeProgramConfig(t, root, server.URL)
	s := startProgram(t, root, "Edited scenario")
	s.navigate(t, "/changes\r", "Rows 1-1 of 1")
	s.navigate(t, "\r", "no test cases")
	s.navigate(t, "/new-testcase\r", "TestCaseCreateScreen")
	s.navigate(t, "\x1b[200~/delete literal\x1b[201~\r", "saved test case")
	// Identity, five editable metadata fields and three docs precede the testcase.
	s.send(t, strings.Repeat("\x1b[B", 9)+"\r")
	s.waitFor(t, "TestCaseUpdateScreen")
	s.waitFor(t, "saved test case")
	s.waitFor(t, "Edited scenario")
	s.navigate(t, " ", "[✓] Edited scenario")
	s.navigate(t, " ", "[ ] Edited scenario")
	s.navigate(t, "h", "testcase history unavailable")
	s.navigate(t, "\x1b[3~", "Are you sure?")
	s.navigate(t, "\r", "deleted test case")
	s.finishFromDetails(t)
	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, []string{"list", "create", "list", "update", "list", "done", "list", "done", "list", "delete", "list"}, requests)
	require.Empty(t, rows)
}

func TestCLIProgramTestCaseCommittedWriteAndStaleRecovery(t *testing.T) {
	var mu sync.Mutex
	change := programChange(12, "Existing")
	rows := []map[string]any{}
	createCalls, doneCalls := 0, 0
	failList, malformed := 0, false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/doc/comment-list":
			writeProgramJSON(w, []any{})
		case "/api/v1/project/details":
			writeProgramJSON(w, programProject(7, "Program Project"))
		case "/api/v1/project/config":
			writeProgramJSON(w, programProjectConfig())
		case "/api/v1/change/list":
			writeProgramJSON(w, []any{change})
		case "/api/v1/change/details":
			change["done_tc"] = 0
			change["total_tc"] = len(rows)
			writeProgramJSON(w, change)
		case "/api/v1/doc/list-active":
			writeProgramJSON(w, []any{})
		case "/api/v1/test-case/list":
			if failList > 0 {
				failList--
				http.Error(w, fmt.Sprintf("read unavailable %d", failList), 500)
				return
			}
			if malformed {
				malformed = false
				writeProgramJSON(w, map[string]any{"wrong": "shape"})
				return
			}
			if rows == nil {
				writeProgramJSON(w, []any{})
			} else {
				writeProgramJSON(w, rows)
			}
		case "/api/v1/test-case/create":
			require.Equal(t, map[string]any{"change_id": float64(12), "scenario": "saved once"}, payload)
			createCalls++
			rows = append(rows, map[string]any{"id": 31, "change_id": 12, "scenario": "saved once", "done": false, "created_at": "2026-09-28T10:00:00Z", "updated_at": "2026-09-28T10:00:00Z"})
			failList = 3
			w.WriteHeader(201)
			writeProgramJSON(w, map[string]any{"id": 31})
		case "/api/v1/test-case/update-done":
			doneCalls++
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
	s.navigate(t, "/changes\r", "Rows 1-1 of 1")
	s.navigate(t, "\r", "no test cases")
	s.navigate(t, "/new-testcase\r", "TestCaseCreateScreen")
	s.navigate(t, "saved once\r", "saved test case; refresh failed")
	s.navigate(t, "/retry\r", "read unavailable 1")
	require.Contains(t, s.output.String(), "saved test case; refresh failed (#31)")
	s.navigate(t, "/retry\r", "read unavailable 0")
	require.Contains(t, s.output.String(), "saved test case; refresh failed (#31)")
	s.navigate(t, "/retry\r", "saved test case; refreshed test cases")
	s.waitFor(t, "#31")
	s.navigate(t, "/retry\r", "loaded change")
	mu.Lock()
	malformed = true
	mu.Unlock()
	s.navigate(t, "/retry\r", "load failed")
	s.send(t, " ")
	s.navigate(t, "/retry\r", "loaded change")
	s.finishFromDetails(t)
	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, 1, createCalls)
	require.Zero(t, doneCalls)
}

func TestCLIProgramTestCaseShutdownCancelsRequest(t *testing.T) {
	started, canceled := make(chan struct{}), make(chan struct{})
	var mu sync.Mutex
	created := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/doc/comment-list":
			writeProgramJSON(w, []any{})
		case "/api/v1/project/details":
			writeProgramJSON(w, programProject(7, "Program Project"))
		case "/api/v1/project/config":
			writeProgramJSON(w, programProjectConfig())
		case "/api/v1/change/list":
			writeProgramJSON(w, []any{programChange(12, "Existing")})
		case "/api/v1/change/details":
			writeProgramJSON(w, programChange(12, "Existing"))
		case "/api/v1/doc/list-active":
			writeProgramJSON(w, []any{})
		case "/api/v1/test-case/list":
			mu.Lock()
			pending := created
			mu.Unlock()
			if !pending {
				writeProgramJSON(w, []any{})
				return
			}
			close(started)
			<-r.Context().Done()
			close(canceled)
		case "/api/v1/test-case/create":
			mu.Lock()
			created = true
			mu.Unlock()
			w.WriteHeader(201)
			writeProgramJSON(w, map[string]any{"id": 31})
		default:
			t.Errorf("unexpected route %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	root := t.TempDir()
	writeProgramConfig(t, root, server.URL)
	s := startProgram(t, root, "")
	s.navigate(t, "/changes\r", "Rows 1-1 of 1")
	s.navigate(t, "\r", "no test cases")
	s.navigate(t, "/new-testcase\r", "TestCaseCreateScreen")
	s.send(t, "pending\r")
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("testcase refresh did not start")
	}
	controller := <-s.controller
	controller.Quit()
	require.NoError(t, s.waitDone(t))
	select {
	case <-canceled:
	case <-time.After(time.Second):
		t.Fatal("testcase HTTP request remained active")
	}
}

func TestCLIProgramTestCaseKeyboardCancelsBusyRequest(t *testing.T) {
	for _, stall := range []string{"write", "refresh"} {
		for _, cancelKey := range []string{"\x1b", "\x03"} {
			t.Run(stall+"/"+cancelKey, func(t *testing.T) {
				started, canceled := make(chan struct{}), make(chan struct{})
				var mu sync.Mutex
				createCalls, listCalls := 0, 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					var payload map[string]any
					require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
					w.Header().Set("Content-Type", "application/json")
					switch r.URL.Path {
					case "/api/v1/doc/comment-list":
						writeProgramJSON(w, []any{})
					case "/api/v1/project/details":
						writeProgramJSON(w, programProject(7, "Program Project"))
					case "/api/v1/project/config":
						writeProgramJSON(w, programProjectConfig())
					case "/api/v1/change/list":
						writeProgramJSON(w, []any{programChange(12, "Existing")})
					case "/api/v1/change/details":
						writeProgramJSON(w, programChange(12, "Existing"))
					case "/api/v1/doc/list-active":
						writeProgramJSON(w, []any{})
					case "/api/v1/test-case/list":
						mu.Lock()
						listCalls++
						currentList, currentCreate := listCalls, createCalls
						mu.Unlock()
						if stall == "refresh" && currentList == 2 {
							close(started)
							<-r.Context().Done()
							close(canceled)
							return
						}
						if currentCreate > 0 && stall == "refresh" {
							writeProgramJSON(w, []any{map[string]any{"id": 31, "change_id": 12, "scenario": "pending", "done": false, "created_at": "2026-09-28T10:00:00Z", "updated_at": "2026-09-28T10:00:00Z"}})
						} else {
							writeProgramJSON(w, []any{})
						}
					case "/api/v1/test-case/create":
						mu.Lock()
						createCalls++
						mu.Unlock()
						if stall == "write" {
							close(started)
							<-r.Context().Done()
							close(canceled)
							return
						}
						w.WriteHeader(201)
						writeProgramJSON(w, map[string]any{"id": 31})
					default:
						t.Errorf("unexpected route %s", r.URL.Path)
						http.NotFound(w, r)
					}
				}))
				t.Cleanup(server.Close)
				root := t.TempDir()
				writeProgramConfig(t, root, server.URL)
				s := startProgram(t, root, "")
				s.navigate(t, "/changes\r", "Rows 1-1 of 1")
				s.navigate(t, "\r", "no test cases")
				s.navigate(t, "/new-testcase\r", "TestCaseCreateScreen")
				s.send(t, "pending\r")
				select {
				case <-started:
				case <-time.After(time.Second):
					t.Fatal("testcase request did not start")
				}
				// A second submit while busy must not replay the write.
				loadedBefore := s.output.count("loaded change")
				s.send(t, "\r"+cancelKey)
				select {
				case <-canceled:
				case <-time.After(time.Second):
					t.Fatal("terminal cancellation left testcase request active")
				}
				s.output.waitForCount(t, "loaded change", loadedBefore+1)
				s.waitFor(t, "ChangeDetailsScreen")
				s.finishFromDetails(t)
				mu.Lock()
				defer mu.Unlock()
				require.Equal(t, 1, createCalls)
			})
		}
	}
}
