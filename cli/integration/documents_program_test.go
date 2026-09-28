package integration_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type documentProgramBackend struct {
	mu                      sync.Mutex
	calls                   map[string]int
	rows                    map[string][]map[string]any
	inserts                 []map[string]any
	failInsert, failList    int
	malformed, wrongDetails bool
}

func newDocumentProgramBackend(t *testing.T) (*documentProgramBackend, *httptest.Server) {
	t.Helper()
	b := &documentProgramBackend{calls: map[string]int{}, rows: map[string][]map[string]any{}}
	for _, o := range []struct {
		table string
		id    int
		kind  string
	}{{"project", 7, "readme"}, {"epic", 10, "brief"}, {"change", 12, "spec"}} {
		base := 40
		if o.table == "epic" {
			base = 50
		}
		if o.table == "change" {
			base = 60
		}
		old := programDocVersion(base, o.table, o.id, o.kind, "old historical", false)
		current := programDocVersion(base+1, o.table, o.id, o.kind, "new current", true)
		b.rows[fmt.Sprintf("%s:%d", o.table, o.id)] = []map[string]any{current, old}
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b.mu.Lock()
		defer b.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		b.calls[r.URL.Path]++
		var in map[string]any
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			t.Errorf("decode %s: %v", r.URL.Path, err)
			http.Error(w, "bad", 400)
			return
		}
		table, _ := in["ref_table"].(string)
		owner := intFromJSON(in["ref_id"])
		key := fmt.Sprintf("%s:%d", table, owner)
		switch r.URL.Path {
		case "/api/v1/project/details":
			writeProgramJSON(w, programProject(7, "Program Project"))
		case "/api/v1/project/list":
			writeProgramJSON(w, []any{programProject(7, "Program Project")})
		case "/api/v1/project/config":
			writeProgramJSON(w, programProjectConfig())
		case "/api/v1/epic/list":
			writeProgramJSON(w, []any{programEpic(10, 7, "Program Epic")})
		case "/api/v1/epic/details":
			writeProgramJSON(w, programEpic(10, 7, "Program Epic"))
		case "/api/v1/change/list":
			writeProgramJSON(w, []any{programChange(12, "Program Change")})
		case "/api/v1/change/details":
			writeProgramJSON(w, programChange(12, "Program Change"))
		case "/api/v1/test-case/list":
			writeProgramJSON(w, []any{})
		case "/api/v1/doc/list":
			if b.failList > 0 {
				b.failList--
				http.Error(w, "refresh unavailable", http.StatusServiceUnavailable)
				return
			}
			if b.malformed {
				_, _ = w.Write([]byte(`{"rows":[]}`))
				return
			}
			writeProgramJSON(w, b.rows[key])
		case "/api/v1/doc/current":
			var current []map[string]any
			for _, d := range b.rows[key] {
				if d["current"] == true {
					current = append(current, d)
				}
			}
			if current == nil {
				current = []map[string]any{}
			}
			writeProgramJSON(w, current)
		case "/api/v1/doc/details":
			id := intFromJSON(in["id"])
			for _, docs := range b.rows {
				for _, d := range docs {
					if intFromJSON(d["id"]) == id {
						if b.wrongDetails {
							clone := map[string]any{}
							for k, v := range d {
								clone[k] = v
							}
							clone["ref_id"] = 999
							writeProgramJSON(w, clone)
						} else {
							writeProgramJSON(w, d)
						}
						return
					}
				}
			}
			http.Error(w, "missing", 404)
		case "/api/v1/doc/insert":
			b.inserts = append(b.inserts, in)
			if b.failInsert > 0 {
				b.failInsert--
				http.Error(w, "insert refused", http.StatusConflict)
				return
			}
			id := 91 + len(b.inserts) - 1
			for _, d := range b.rows[key] {
				if d["doc_type"] == in["doc_type"] {
					d["current"] = false
				}
			}
			row := programDocVersion(id, table, owner, in["doc_type"].(string), in["body"].(string), true)
			row["agent_edit"] = in["agent_edit"]
			b.rows[key] = append([]map[string]any{row}, b.rows[key]...)
			w.WriteHeader(201)
			writeProgramJSON(w, map[string]any{"id": id})
		default:
			t.Errorf("unexpected document program path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return b, server
}

func intFromJSON(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	}
	return 0
}

func programDocVersion(id int, table string, owner int, kind, body string, current bool) map[string]any {
	return map[string]any{"id": id, "ref_id": owner, "ref_table": table, "doc_type": kind, "body": body, "agent_edit": false, "current": current, "created_at": "2026-09-28T10:00:00Z", "updated_at": "2026-09-28T11:00:00Z", "html": "<p>server rendered</p>"}
}

func documentSession(t *testing.T, server *httptest.Server) *programSession {
	root := t.TempDir()
	writeProgramConfig(t, root, server.URL)
	return startProgram(t, root, "")
}

func openProgramDocuments(t *testing.T, s *programSession, owner string) {
	t.Helper()
	switch owner {
	case "project":
		s.navigate(t, "/projects\r", "ProjectsListScreen")
		s.navigate(t, "\r", "ProjectDetailsScreen")
	case "epic":
		s.navigate(t, "/epics\r", "EpicsListScreen")
		s.navigate(t, "\r", "EpicDetailsScreen")
	case "change":
		s.navigate(t, "/changes\r", "ChangesListScreen")
		s.navigate(t, "\r", "ChangeDetailsScreen")
	}
	s.navigate(t, "/documents\r", "DocumentScreen")
	s.waitFor(t, "History: 2")
}

func finishDocumentSession(t *testing.T, s *programSession) {
	t.Helper()
	s.send(t, "/return\r")
	s.send(t, "/return\r")
	s.send(t, "/return\r")
	s.send(t, "/quit\r")
	require.NoError(t, s.waitDone(t))
}

func TestCLIProgramDocumentOwnersAndHistory(t *testing.T) {
	for _, owner := range []string{"project", "epic", "change"} {
		t.Run(owner, func(t *testing.T) {
			b, server := newDocumentProgramBackend(t)
			s := documentSession(t, server)
			openProgramDocuments(t, s, owner)
			wantID := 40
			if owner == "epic" {
				wantID = 50
			}
			if owner == "change" {
				wantID = 60
			}
			s.navigate(t, "\x1b[B\r", fmt.Sprintf("Version #%d", wantID))
			require.Contains(t, s.output.String(), "historical")
			require.Contains(t, s.output.String(), "Raw body:")
			require.Contains(t, s.output.String(), "Rendered HTML:")
			s.navigate(t, "/return\r", "History: 2")
			if owner == "change" {
				s.navigate(t, "/type\r", "selected document type: spec")
				s.navigate(t, "/new-document\r", "Document draft:")
				s.navigate(t, "configured choice\r", "saved document #91")
			}
			finishDocumentSession(t, s)
			b.mu.Lock()
			defer b.mu.Unlock()
			require.GreaterOrEqual(t, b.calls["/api/v1/doc/list"], 1)
			require.GreaterOrEqual(t, b.calls["/api/v1/doc/current"], 1)
			if owner == "change" {
				require.Equal(t, "spec", b.inserts[0]["doc_type"])
				require.Equal(t, "configured choice", b.inserts[0]["body"])
				require.GreaterOrEqual(t, b.calls["/api/v1/doc/details"], 2)
			} else {
				require.Equal(t, 1, b.calls["/api/v1/doc/details"])
			}
		})
	}
}

func TestCLIProgramDocumentAppendAndRecovery(t *testing.T) {
	b, server := newDocumentProgramBackend(t)
	s := documentSession(t, server)
	openProgramDocuments(t, s, "project")
	b.mu.Lock()
	b.failInsert = 1
	b.mu.Unlock()
	s.navigate(t, "/new-document\r", "Document draft:")
	s.navigate(t, " /cancel literal\r", "insert refused")
	s.navigate(t, "\r", "saved document #92")
	b.mu.Lock()
	require.Len(t, b.inserts, 2)
	require.Equal(t, b.inserts[0]["body"], b.inserts[1]["body"])
	require.Equal(t, " /cancel literal", b.inserts[1]["body"])
	require.Equal(t, false, b.inserts[1]["agent_edit"])
	b.failList = 2
	b.mu.Unlock()
	s.navigate(t, "/new-document\r", "Document draft:")
	s.navigate(t, "second version\r", "saved document #93; refresh")
	s.send(t, "/retry\r")
	waitDocumentCalls(t, b, "/api/v1/doc/list", 4)
	s.send(t, "/retry\r")
	waitDocumentCalls(t, b, "/api/v1/doc/list", 5)
	s.waitFor(t, "saved document #93; refreshed")
	b.mu.Lock()
	require.Len(t, b.inserts, 3)
	require.Equal(t, 3, b.calls["/api/v1/doc/insert"])
	b.mu.Unlock()
	finishDocumentSession(t, s)
}

func TestCLIProgramDocumentMalformedAndStaleScope(t *testing.T) {
	b, server := newDocumentProgramBackend(t)
	s := documentSession(t, server)
	openProgramDocuments(t, s, "epic")
	s.navigate(t, "\r", "Version #51")
	s.navigate(t, "/return\r", "History: 2")
	b.mu.Lock()
	b.wrongDetails = true
	b.mu.Unlock()
	s.navigate(t, "\r", "different owner")
	b.mu.Lock()
	b.wrongDetails = false
	b.malformed = true
	b.mu.Unlock()
	s.navigate(t, "/retry\r", "backend contract")
	b.mu.Lock()
	b.malformed = false
	b.mu.Unlock()
	s.navigate(t, "/retry\r", "History: 2")
	b.mu.Lock()
	b.rows["epic:10"] = []map[string]any{}
	b.mu.Unlock()
	s.navigate(t, "/retry\r", "History: 0")
	require.NotContains(t, strings.TrimSpace(s.output.String()), "\x1b[2Jold historical")
	finishDocumentSession(t, s)
}

func waitDocumentCalls(t *testing.T, b *documentProgramBackend, path string, want int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		b.mu.Lock()
		n := b.calls[path]
		b.mu.Unlock()
		if n >= want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s count %d", path, want)
}

func TestCLIProgramDocumentNavigationCancelsRequest(t *testing.T) {
	started, canceled := make(chan struct{}, 1), make(chan struct{}, 1)
	release := make(chan struct{})
	defer close(release)
	var mu sync.Mutex
	lists := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/project/config":
			writeProgramJSON(w, programProjectConfig())
		case "/api/v1/project/list":
			writeProgramJSON(w, []any{programProject(7, "Program Project"), programProject(8, "Second Project")})
		case "/api/v1/project/details":
			var request struct {
				ID int `json:"id"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
			if request.ID == 8 {
				writeProgramJSON(w, programProject(8, "Second Project"))
			} else {
				writeProgramJSON(w, programProject(7, "Program Project"))
			}
		case "/api/v1/doc/list":
			mu.Lock()
			lists++
			n := lists
			mu.Unlock()
			if n == 1 {
				_, _ = io.Copy(io.Discard, r.Body)
				started <- struct{}{}
				select {
				case <-r.Context().Done():
					canceled <- struct{}{}
				case <-release:
				}
				return
			}
			writeProgramJSON(w, []any{})
		case "/api/v1/doc/current":
			writeProgramJSON(w, []any{})
		default:
			t.Errorf("unexpected request %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	s := documentSession(t, server)
	s.navigate(t, "/projects\r", "ProjectsListScreen")
	s.navigate(t, "\r", "ProjectDetailsScreen")
	s.navigate(t, "/documents\r", "DocumentScreen")
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("document read did not start")
	}
	s.navigate(t, "\x1b", "ProjectDetailsScreen")
	select {
	case <-canceled:
	case <-time.After(5 * time.Second):
		t.Fatal("document read was not canceled")
	}
	s.navigate(t, "/return\r", "ProjectsListScreen")
	s.send(t, "\x1b[B")
	s.navigate(t, "\r", "Second Project")
	s.navigate(t, "/documents\r", "DocumentScreen")
	s.waitFor(t, "Documents: project #8")
	s.waitFor(t, "History: 0")
	finishDocumentSession(t, s)
	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, 2, lists)
}

func TestCLIProgramDocumentShutdownCancelsRequest(t *testing.T) {
	started, canceled := make(chan struct{}, 1), make(chan struct{}, 1)
	release := make(chan struct{})
	defer close(release)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/project/config":
			writeProgramJSON(w, programProjectConfig())
		case "/api/v1/project/list":
			writeProgramJSON(w, []any{programProject(7, "Program Project")})
		case "/api/v1/project/details":
			writeProgramJSON(w, programProject(7, "Program Project"))
		case "/api/v1/doc/list":
			_, _ = io.Copy(io.Discard, r.Body)
			started <- struct{}{}
			select {
			case <-r.Context().Done():
				canceled <- struct{}{}
			case <-release:
			}
		default:
			t.Errorf("unexpected request %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	s := documentSession(t, server)
	s.navigate(t, "/projects\r", "ProjectsListScreen")
	s.navigate(t, "\r", "ProjectDetailsScreen")
	s.navigate(t, "/documents\r", "DocumentScreen")
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("pending read did not start")
	}
	controller := <-s.controller
	controller.Quit()
	require.NoError(t, s.waitDone(t))
	select {
	case <-canceled:
	case <-time.After(5 * time.Second):
		t.Fatal("pending read was not canceled on shutdown")
	}
}
