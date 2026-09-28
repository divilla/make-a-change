package integration_test

import (
	"cli/internal/agent"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCLIProgramBriefFailuresAndStaleCancellation(t *testing.T) {
	var mu sync.Mutex
	withSpec := false
	created := 0
	createAttempts := 0
	readsFailed := 0
	var writeAndReadCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		switch r.URL.Path {
		case "/api/v1/project/details":
			writeProgramJSON(w, programProject(7, "Program Project"))
		case "/api/v1/project/config":
			cfg := programProjectConfig()
			if !withSpec {
				cfg["change_docs"] = []string{"brief"}
			}
			writeProgramJSON(w, cfg)
		case "/api/v1/change/create":
			writeAndReadCalls = append(writeAndReadCalls, "create")
			createAttempts++
			if createAttempts == 1 {
				http.Error(w, "create down", http.StatusInternalServerError)
				return
			}
			created++
			require.Equal(t, "Draft", body["brief"])
			w.WriteHeader(201)
			writeProgramJSON(w, map[string]int{"id": 12})
		case "/api/v1/change/details":
			writeProgramJSON(w, programChange(12, "Title"))
		case "/api/v1/doc/current":
			writeAndReadCalls = append(writeAndReadCalls, "current")
			if readsFailed < 2 {
				readsFailed++
				http.Error(w, "read down", 500)
				return
			}
			d := programDocument("brief", "Draft")
			writeProgramJSON(w, []any{d})
		case "/api/v1/health":
			writeProgramJSON(w, map[string]any{"status": "ok"})
		default:
			t.Errorf("unexpected route %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	root := t.TempDir()
	writeProgramConfig(t, root, server.URL)
	runner := &scriptedBriefRunner{err: errors.New("runner unavailable")}
	s := startProgram(t, root, "", runner)
	s.navigate(t, "/brief-new\r", "requires configured brief and spec")
	mu.Lock()
	require.Zero(t, created)
	withSpec = true
	mu.Unlock()
	s.navigate(t, "/retry\r", "brief ready for editing")
	s.send(t, "Draft\r")
	s.navigate(t, "/title\r", "Input: title")
	s.send(t, "Title\r")
	s.navigate(t, "/confirm\r", "create down")
	s.waitFor(t, "Draft")
	mu.Lock()
	require.Equal(t, []string{"create"}, writeAndReadCalls)
	require.Zero(t, created)
	mu.Unlock()
	s.navigate(t, "/confirm\r", "committed ID 12")
	s.send(t, "/confirm\r")
	require.Eventually(t, func() bool { mu.Lock(); defer mu.Unlock(); return readsFailed == 2 }, 3*time.Second, 10*time.Millisecond)
	s.navigate(t, "/retry\r", "runner unavailable")
	runner.mu.Lock()
	runner.err = nil
	runner.results = []agent.Output{{RewrittenBrief: "Draft", Questions: []agent.Question{{ID: "Q1", Text: "Which?", Context: "intro"}, {ID: "Q1", Text: "When?", Context: "timeline"}}, Unresolved: []string{"Q1"}}, {}}
	runner.mu.Unlock()
	s.navigate(t, "/retry\r", "duplicate or incomplete question")
	s.navigate(t, "/retry\r", "agent output is empty")
	s.navigate(t, "/return\r", "MainScreen")
	s.send(t, "/quit\r")
	require.NoError(t, s.waitDone(t))
	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, 1, created, "read-only retries must not recreate the change")
	require.Equal(t, 2, createAttempts)
	require.Equal(t, 2, readsFailed)
	require.Equal(t, []string{"create", "create", "current", "current", "current", "current", "current", "current"}, writeAndReadCalls)
}

type scriptedBriefRunner struct {
	mu       sync.Mutex
	results  []agent.Output
	requests []agent.Request
	err      error
}

type cancelBriefRunner struct{ started, canceled chan struct{} }

func (r *cancelBriefRunner) Run(ctx context.Context, req agent.Request) (agent.Output, string, error) {
	close(r.started)
	<-ctx.Done()
	close(r.canceled)
	return agent.Output{InputRevision: req.Revision, RewrittenBrief: "Late ready", Questions: []agent.Question{}, Unresolved: []string{}, ReadyForSpec: true}, "/tmp/late-brief", nil
}

func TestCLIProgramBriefStaleCancellation(t *testing.T) {
	var mu sync.Mutex
	inserts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		switch r.URL.Path {
		case "/api/v1/project/details":
			writeProgramJSON(w, programProject(7, "Program Project"))
		case "/api/v1/project/config":
			writeProgramJSON(w, programProjectConfig())
		case "/api/v1/change/list":
			writeProgramJSON(w, []any{programChange(12, "Original")})
		case "/api/v1/change/details":
			writeProgramJSON(w, programChange(12, "Original"))
		case "/api/v1/doc/current":
			writeProgramJSON(w, []any{programDocument("brief", "Original")})
		case "/api/v1/test-case/list":
			writeProgramJSON(w, []any{})
		case "/api/v1/doc/insert":
			inserts++
			w.WriteHeader(201)
			writeProgramJSON(w, map[string]int{"id": 92})
		default:
			t.Errorf("unexpected route %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	root := t.TempDir()
	writeProgramConfig(t, root, server.URL)
	runner := &cancelBriefRunner{started: make(chan struct{}), canceled: make(chan struct{})}
	s := startProgram(t, root, "", runner)
	s.navigate(t, "/changes\r", "ChangesListScreen")
	s.navigate(t, "\r", "loaded change")
	s.navigate(t, "/brief-clarify\r", "brief ready for editing")
	s.send(t, "/confirm\r")
	select {
	case <-runner.started:
	case <-time.After(3 * time.Second):
		t.Fatal("runner did not start")
	}
	s.navigate(t, "\x1b", "ChangeDetailsScreen")
	select {
	case <-runner.canceled:
	case <-time.After(3 * time.Second):
		t.Fatal("runner was not canceled")
	}
	require.NotContains(t, s.output.String(), "brief ready for spec writing")
	s.finishFromDetails(t)
	mu.Lock()
	defer mu.Unlock()
	require.Zero(t, inserts)
}

func (r *scriptedBriefRunner) Run(_ context.Context, req agent.Request) (agent.Output, string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.requests = append(r.requests, req)
	if r.err != nil {
		return agent.Output{}, "/tmp/failed-brief", r.err
	}
	out := r.results[0]
	r.results = r.results[1:]
	out.InputRevision = req.Revision
	return out, filepath.Dir(req.InputPath), nil
}

func TestCLIProgramBriefNewAndExistingPersistence(t *testing.T) {
	var mu sync.Mutex
	var calls []string
	var created, inserted []map[string]any
	failedInsertRefreshes := 0
	doc := map[string]any(nil)
	change := programChange(12, "Existing")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		calls = append(calls, r.URL.Path)
		switch r.URL.Path {
		case "/api/v1/test-case/list":
			writeProgramJSON(w, []any{})
		case "/api/v1/project/details":
			writeProgramJSON(w, programProject(7, "Program Project"))
		case "/api/v1/project/config":
			writeProgramJSON(w, programProjectConfig())
		case "/api/v1/change/list":
			writeProgramJSON(w, []any{change})
		case "/api/v1/change/details":
			writeProgramJSON(w, change)
		case "/api/v1/doc/current":
			if len(inserted) == 4 && failedInsertRefreshes < 1 {
				failedInsertRefreshes++
				http.Error(w, "read down", http.StatusInternalServerError)
				return
			}
			if doc == nil {
				writeProgramJSON(w, []any{})
			} else {
				writeProgramJSON(w, []any{doc})
			}
		case "/api/v1/change/create":
			created = append(created, body)
			change["title"] = body["title"]
			doc = programDocument("brief", strings.TrimSpace(body["brief"].(string)))
			doc["ref_id"] = 12
			w.WriteHeader(http.StatusCreated)
			writeProgramJSON(w, map[string]int{"id": 12})
		case "/api/v1/doc/insert":
			inserted = append(inserted, body)
			doc = programDocument("brief", strings.TrimSpace(body["body"].(string)))
			doc["id"] = 91 + len(inserted)
			doc["agent_edit"] = body["agent_edit"]
			w.WriteHeader(http.StatusCreated)
			writeProgramJSON(w, map[string]int{"id": 91 + len(inserted)})
		default:
			t.Errorf("unexpected route %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	root := t.TempDir()
	writeProgramConfig(t, root, server.URL)
	runner := &scriptedBriefRunner{results: []agent.Output{{RewrittenBrief: "Rewritten one", Questions: []agent.Question{{ID: "Q1", Text: "Which?", Context: "intro"}}, Unresolved: []string{"Q1"}}, {RewrittenBrief: "Rewritten two", Questions: []agent.Question{{ID: "Q2", Text: "When?", Context: "timeline"}}, Unresolved: []string{"Q2"}}, {RewrittenBrief: "Rewritten three", Questions: []agent.Question{{ID: "Q2", Text: "When?", Context: "timeline"}}, Unresolved: []string{}, ReadyForSpec: true}, {RewrittenBrief: "Existing human edit", Questions: []agent.Question{}, Unresolved: []string{}, ReadyForSpec: true}}}
	s := startProgram(t, root, "Existing human edit", runner)
	s.navigate(t, "/brief-new\r", "BriefScreen")
	s.waitFor(t, "brief ready for editing")
	s.send(t, "User brief\r")
	s.navigate(t, "/title\r", "Input: title")
	s.send(t, "New title\r")
	s.navigate(t, "/confirm\r", "Agent proposed rewrite")
	s.navigate(t, "/approve\r", "Question Q1")
	s.waitFor(t, "brief ready for editing") // retained screen remains responsive after version refresh
	s.navigate(t, "\x01", "Input: answer")
	s.send(t, "Q1: Alpha\r")
	s.navigate(t, "/resolve\r", "Rewritten two")
	s.navigate(t, "/approve\r", "Unresolved: Q2")
	s.navigate(t, "/resolve\r", "unanswered blocker")
	s.send(t, "Q2: Tomorrow\r")
	s.navigate(t, "/resolve\r", "Rewritten three")
	s.navigate(t, "/approve\r", "brief ready for spec writing")
	s.navigate(t, "/return\r", "MainScreen")
	s.navigate(t, "/changes\r", "ChangesListScreen")
	s.navigate(t, "\r", "ChangeDetailsScreen")
	s.navigate(t, "/brief-clarify\r", "BriefScreen")
	s.waitFor(t, "brief ready for editing")
	s.navigate(t, "\x05", "brief edited")
	s.navigate(t, "/confirm\r", "committed ID 95; retry read or agent step")
	s.navigate(t, "/confirm\r", "Agent proposed rewrite: Existing human edit")
	s.navigate(t, "/approve\r", "brief ready for spec writing")
	s.navigate(t, "/return\r", "ChangeDetailsScreen")
	s.finishFromDetails(t)
	mu.Lock()
	defer mu.Unlock()
	require.Len(t, created, 1)
	require.Equal(t, "New title", created[0]["title"])
	require.Equal(t, "User brief", created[0]["brief"])
	require.Len(t, inserted, 4)
	require.Equal(t, 1, failedInsertRefreshes)
	require.Equal(t, true, inserted[0]["agent_edit"])
	require.Equal(t, "Rewritten one", inserted[0]["body"])
	require.Equal(t, "Rewritten two", inserted[1]["body"])
	require.Equal(t, "Rewritten three", inserted[2]["body"])
	require.Equal(t, false, inserted[3]["agent_edit"])
	require.Equal(t, "Existing human edit", inserted[3]["body"])
	require.Contains(t, calls, "/api/v1/change/details")
	runner.mu.Lock()
	defer runner.mu.Unlock()
	require.Equal(t, "brief-rewrite", runner.requests[0].Prompt)
	require.Equal(t, "brief-resolve", runner.requests[1].Prompt)
	require.Equal(t, "Alpha", runner.requests[1].Questions[0].Answer)
	require.Equal(t, "brief-resolve", runner.requests[2].Prompt)
	require.Equal(t, "Tomorrow", runner.requests[2].Questions[0].Answer)
	require.Equal(t, "Alpha", runner.requests[2].Questions[1].Answer)
}
