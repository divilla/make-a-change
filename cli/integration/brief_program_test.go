package integration_test

import (
	"cli/internal/agent"
	"cli/pkg/briefprocess"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// programAgent uses owned local processes, including Bubble Tea's real terminal handoff.
func programAgent(t *testing.T, root, mode string) agent.Runner {
	t.Helper()
	prompts := filepath.Join(root, ".mch/default/prompts")
	require.NoError(t, os.MkdirAll(prompts, 0o700))
	for _, name := range []string{"brief-rewrite", "spec-write"} {
		require.NoError(t, os.WriteFile(filepath.Join(prompts, name+".md"), []byte(name+"\n[brief-file-path.md]"), 0o600))
	}
	script := filepath.Join(root, "codex-test")
	body := `#!/bin/sh
set -eu
mode='MODE'
if [ "$1" = resume ]; then
 [ "$2" = '0198a86f-9b8a-7d89-ae5b-6f25b528b04c' ] || exit 8
 brief=$(cat "$0.brief-path")
 if [ "$mode" != resume-missing ] && [ "$mode" != resume-existing ] && [ "$mode" != resume-identical ]; then
  printf '# Resumed spec\n## Testcases\n- Click save → saved.\n' > "$(dirname "$brief")/spec.md"
 fi
 exit 0
fi
if [ "$1" = -C ]; then
 brief=$(printf '%s' "$3" | tail -n 1)
 printf '%s' "$brief" > "$0.brief-path"
 if [ "$mode" != unchanged ]; then
  printf '# Agent rewrite\nClarified body\n' > "$brief"
  touch -m -d '2031-01-01 00:00:00' "$brief"
 fi
 exit 0
fi
[ "$1" = exec ] || exit 8
[ "$#" = 12 ] && [ "$8" = --sandbox ] && [ "$9" = workspace-write ] && [ "${10}" = --add-dir ] || exit 8
brief=$(printf '%s' "${12}" | tail -n 1)
[ "${11}" = "$(dirname "$brief")" ] || exit 8
printf '\033[32mLIVE spec progress\033[0m\n'
sleep 0.1
printf 'session id: 0198a86f-9b8a-7d89-ae5b-6f25b528b04c\n' >&2
if [ "$mode" = exec-fail ]; then exit 3; fi
if [ "$mode" = resume-existing ] || [ "$mode" = resume-identical ]; then
 printf '# Generated spec\n## Testcases\n- Click save → saved.\n' > "$(dirname "$brief")/spec.md"
 printf 'Question?' > "$7"
elif [ "$mode" = resume ] || [ "$mode" = resume-missing ]; then
 printf 'Question?' > "$7"
else
 printf '# Generated spec\n## Testcases\n- Click save → saved.\n' > "$(dirname "$brief")/spec.md"
 printf 'Done.' > "$7"
fi
`
	require.NoError(t, os.WriteFile(script, []byte(strings.ReplaceAll(body, "MODE", mode)), 0o700))
	return briefprocess.Runner{Executable: script}
}

func TestCLIProgram032BriefSpecFlow(t *testing.T) {
	for _, mode := range []string{"done", "resume", "resume-existing", "resume-identical", "resume-missing", "unchanged", "exec-fail"} {
		t.Run(mode, func(t *testing.T) {
			t.Setenv("TMPDIR", t.TempDir())
			var mu sync.Mutex
			var creates, inserts []map[string]any
			var typesUpdates []map[string]any
			var cases []map[string]any
			active := map[string]map[string]any{}
			change := programChange(12, "Editor title")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				var payload map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
				switch r.URL.Path {
				case "/api/v1/project/details":
					writeProgramJSON(w, programProject(7, "Program Project"))
				case "/api/v1/project/config":
					writeProgramJSON(w, programProjectConfig())
				case "/api/v1/change/list":
					writeProgramJSON(w, []any{programChange(11, "Previously loaded change")})
				case "/api/v1/change/create":
					require.Equal(t, "POST", r.Method)
					require.Equal(t, float64(7), payload["project_id"])
					require.Equal(t, "Editor title", payload["title"])
					require.Equal(t, "\n# Editor title\nTypes: feature", payload["brief"])
					require.Regexp(t, `^[0-9a-f-]{36}$`, payload["ref_uuid"])
					creates = append(creates, payload)
					change["ref_uuid"] = payload["ref_uuid"]
					active["brief"] = programDocument("brief", strings.TrimSpace(payload["brief"].(string)))
					w.WriteHeader(201)
					writeProgramJSON(w, map[string]any{"id": 12})
				case "/api/v1/change/update-types":
					require.Len(t, creates, 1)
					require.Empty(t, inserts)
					require.Equal(t, map[string]any{"id": float64(12), "change_types": []any{"feature"}}, payload)
					typesUpdates = append(typesUpdates, payload)
					change["change_types"] = payload["change_types"]
					w.WriteHeader(http.StatusNoContent)
				case "/api/v1/change/details":
					if payload["id"] == float64(11) {
						writeProgramJSON(w, programChange(11, "Previously loaded change"))
					} else {
						writeProgramJSON(w, change)
					}
				case "/api/v1/doc/list-active":
					if payload["ref_id"] == float64(11) {
						doc := programDocument("brief", "# Previously saved brief")
						doc["ref_id"] = 11
						writeProgramJSON(w, []any{doc})
						return
					}
					rows := []any{}
					for _, kind := range []string{"spec", "brief"} {
						if doc := active[kind]; doc != nil {
							rows = append(rows, doc)
						}
					}
					writeProgramJSON(w, rows)
				case "/api/v1/doc/comment-list":
					writeProgramJSON(w, []any{})
				case "/api/v1/doc/insert":
					if payload["ref_id"] == float64(11) {
						require.Equal(t, "spec", payload["doc_type"])
						require.Equal(t, false, payload["agent_edit"])
						http.Error(w, "editor save refused", http.StatusServiceUnavailable)
						return
					}
					require.Equal(t, float64(12), payload["ref_id"])
					require.Equal(t, "change", payload["ref_table"])
					require.Equal(t, true, payload["agent_edit"])
					inserts = append(inserts, payload)
					doc := programDocument(payload["doc_type"].(string), strings.TrimSpace(payload["body"].(string)))
					doc["id"] = 100 + len(inserts)
					doc["agent_edit"] = true
					active[payload["doc_type"].(string)] = doc
					if mode == "resume-identical" && payload["doc_type"] == "brief" {
						// Another save becomes active while the agent is running.
						spec := programDocument("spec", "# Generated spec\n## Testcases\n- Click save → saved.")
						spec["id"] = 200
						active["spec"] = spec
					}
					w.WriteHeader(201)
					writeProgramJSON(w, map[string]any{"id": doc["id"]})
				case "/api/v1/test-case/list":
					if payload["change_id"] == float64(11) {
						writeProgramJSON(w, []any{})
						return
					}
					require.Equal(t, float64(12), payload["change_id"])
					if cases == nil {
						cases = []map[string]any{}
					}
					writeProgramJSON(w, cases)
				case "/api/v1/test-case/create":
					require.Equal(t, map[string]any{"change_id": float64(12), "scenario": "Click save → saved."}, payload)
					cases = append(cases, map[string]any{"id": 31, "change_id": 12, "scenario": payload["scenario"], "done": false, "created_at": "2026-09-28T10:00:00Z", "updated_at": "2026-09-28T10:00:00Z"})
					w.WriteHeader(201)
					writeProgramJSON(w, map[string]any{"id": 31})
				default:
					t.Errorf("unexpected %s", r.URL.Path)
					http.NotFound(w, r)
				}
			}))
			t.Cleanup(server.Close)
			root := t.TempDir()
			writeProgramConfig(t, root, server.URL)
			runner := programAgent(t, root, mode)
			s := startProgram(t, root, "\n# Editor title\nTypes: feature", runner)
			s.navigate(t, "/changes\r", "Rows 1-1 of 1")
			s.navigate(t, "\r", "loaded change")
			s.navigate(t, "/edit-spec\r", "editor save refused")
			drafts, err := filepath.Glob(filepath.Join(os.TempDir(), "mch-project-*.md"))
			require.NoError(t, err)
			require.Len(t, drafts, 1)
			s.navigate(t, "\x1b", "prompt cleared")
			s.navigate(t, "/return\r", "ChangesListScreen")
			s.send(t, "/new-change\r")
			switch mode {
			case "done", "resume", "resume-existing", "resume-identical":
				s.waitFor(t, "testcases synchronized")
			case "resume-missing":
				s.waitFor(t, "Error generating `spec`")
			case "exec-fail":
				s.waitFor(t, "spec writing:")
			case "unchanged":
				s.waitFor(t, "brief unchanged")
			}
			s.waitFor(t, "ChangeDetailsScreen")
			require.Equal(t, "\n# Editor title\nTypes: feature", readFile(t, drafts[0]), "another change's workflow must retain the failed editor save")
			if mode == "exec-fail" || mode == "resume-missing" {
				s.navigate(t, "/brief\r", "Load change details before")
				s.navigate(t, "/retry\r", "loaded change")
			}
			if mode != "unchanged" {
				require.Contains(t, s.output.String(), "AgentExecScreen")
				s.waitFor(t, "LIVE spec progress")
				require.Contains(t, s.output.String(), "\x1b[32m")
			}
			s.send(t, "/return\r")
			s.finishFromChanges(t)
			mu.Lock()
			defer mu.Unlock()
			require.Len(t, creates, 1)
			require.Len(t, typesUpdates, 1)
			expected := 2
			switch mode {
			case "unchanged":
				expected = 0
			case "resume-missing", "exec-fail", "resume-identical":
				expected = 1
			}
			require.Len(t, inserts, expected)
			if mode == "resume-identical" {
				require.Contains(t, s.output.String(), "spec unchanged")
				require.Equal(t, 200, active["spec"]["id"])
			}
			if expected == 2 || mode == "resume-identical" {
				require.Len(t, cases, 1)
				require.False(t, cases[0]["done"].(bool))
			} else {
				require.Empty(t, cases)
			}
			require.NotContains(t, s.output.String(), "ChangeCreateScreen")
			require.NotContains(t, s.output.String(), "BriefScreen")
		})
	}
}

func TestCLIProgram032InvalidBriefAndCreationFailure(t *testing.T) {
	for _, brief := range []string{"", "## Wrong", "# ", "text\n# Title", "# Save fails"} {
		t.Run(fmt.Sprintf("%q", brief), func(t *testing.T) {
			t.Setenv("TMPDIR", t.TempDir())
			creates := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/api/v1/project/details":
					writeProgramJSON(w, programProject(7, "Program Project"))
				case "/api/v1/project/config":
					writeProgramJSON(w, programProjectConfig())
				case "/api/v1/change/list":
					writeProgramJSON(w, []any{})
				case "/api/v1/change/create":
					creates++
					http.Error(w, "creation refused", 500)
				default:
					t.Errorf("unexpected %s", r.URL.Path)
					http.NotFound(w, r)
				}
			}))
			t.Cleanup(server.Close)
			root := t.TempDir()
			writeProgramConfig(t, root, server.URL)
			s := startProgram(t, root, brief, programAgent(t, root, "done"))
			s.navigate(t, "/changes\r", "no changes")
			s.send(t, "/new-change\r")
			if brief == "# Save fails" {
				s.waitFor(t, "/api/v1/change/create: 500")
			} else {
				s.waitFor(t, "brief title is required")
			}
			s.waitFor(t, "ChangesListScreen")
			s.finishFromChanges(t)
			if brief == "# Save fails" {
				require.Equal(t, 1, creates)
			} else {
				require.Zero(t, creates)
			}
			paths, err := filepath.Glob(filepath.Join(os.TempDir(), "mch/*/brief.md"))
			require.NoError(t, err)
			require.Len(t, paths, 1)
			require.Equal(t, brief, readFile(t, paths[0]))
		})
	}
}

func TestCLIProgram032ManualSpecAndEditedBrief(t *testing.T) {
	for _, partial := range []bool{false, true} {
		t.Run(fmt.Sprintf("partial=%t", partial), func(t *testing.T) {
			testProgram032ManualSpecAndEditedBrief(t, partial)
		})
	}
}

func testProgram032ManualSpecAndEditedBrief(t *testing.T, partial bool) {
	t.Setenv("TMPDIR", t.TempDir())
	var mu sync.Mutex
	q := "Click save → saved."
	active := map[string]map[string]any{"brief": programDocument("brief", "# Original user brief"), "spec": programDocument("spec", "# Before")}
	active["brief"]["id"] = 90
	active["spec"]["id"] = 91
	caseRow := func(id int, text string, done bool) map[string]any {
		return map[string]any{"id": id, "change_id": 12, "scenario": text, "done": done, "created_at": "2026-09-28T10:00:00Z", "updated_at": "2026-09-28T11:00:00Z"}
	}
	checked := caseRow(2, q, true)
	cases := []map[string]any{caseRow(1, q, false), checked, caseRow(3, "Old action → old result", false)}
	var inserts []map[string]any
	var deleted []int
	creates := 0
	syncFailure := partial
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		switch r.URL.Path {
		case "/api/v1/project/details":
			writeProgramJSON(w, programProject(7, "Program Project"))
		case "/api/v1/project/config":
			writeProgramJSON(w, programProjectConfig())
		case "/api/v1/change/list":
			writeProgramJSON(w, []any{programChange(12, "Existing change")})
		case "/api/v1/change/details":
			writeProgramJSON(w, programChange(12, "Existing change"))
		case "/api/v1/doc/list-active":
			rows := []any{}
			if active["brief"]["id"].(int) > active["spec"]["id"].(int) {
				rows = append(rows, active["brief"], active["spec"])
			} else {
				rows = append(rows, active["spec"], active["brief"])
			}
			writeProgramJSON(w, rows)
		case "/api/v1/doc/comment-list":
			writeProgramJSON(w, []any{})
		case "/api/v1/doc/insert":
			inserts = append(inserts, payload)
			kind := payload["doc_type"].(string)
			doc := programDocument(kind, strings.TrimSpace(payload["body"].(string)))
			doc["id"] = 100 + len(inserts)
			doc["agent_edit"] = payload["agent_edit"]
			active[kind] = doc
			w.WriteHeader(201)
			writeProgramJSON(w, map[string]any{"id": doc["id"]})
		case "/api/v1/test-case/list":
			require.Equal(t, float64(12), payload["change_id"])
			writeProgramJSON(w, cases)
		case "/api/v1/test-case/delete":
			id := int(payload["id"].(float64))
			deleted = append(deleted, id)
			if syncFailure && id == 3 {
				http.Error(w, "testcase delete unavailable", http.StatusServiceUnavailable)
				return
			}
			for i, c := range cases {
				if c["id"].(int) == id {
					cases = append(cases[:i], cases[i+1:]...)
					break
				}
			}
			w.WriteHeader(204)
		case "/api/v1/test-case/create":
			creates++
			http.Error(w, "unexpected testcase create", 500)
		case "/api/v1/change/create":
			t.Error("edited brief must not create another change")
			http.Error(w, "unexpected change create", 500)
		default:
			t.Errorf("unexpected %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	root := t.TempDir()
	writeProgramConfig(t, root, server.URL)
	editor := filepath.Join(root, "editor")
	require.NoError(t, os.WriteFile(editor, []byte("#!/bin/sh\ncp \"$0.text\" \"$1\"\n"), 0o700))
	cfg := filepath.Join(root, ".mch/config.yaml")
	require.NoError(t, os.WriteFile(cfg, []byte("backend_url: "+server.URL+"\nproject_id: 7\neditor: "+editor+"\n"), 0o600))
	require.NoError(t, os.WriteFile(editor+".text", []byte("# Manual spec\n## Testcases\n- "+q), 0o600))
	s := startProgram(t, root, "unused fallback editor", programAgent(t, root, "done"))
	s.navigate(t, "/changes\r", "Rows 1-1 of 1")
	s.navigate(t, "\r", "loaded change")
	s.send(t, "/edit-spec\r")
	if partial {
		s.waitFor(t, "reads current details and testcases")
		// Editing stays blocked while rows are stale; unit tests also cover toggling
		// and deletion, including cancellation and failed refreshes.
		s.navigate(t, "\r", "load change details with /retry before editing")
		mu.Lock()
		require.Len(t, inserts, 1)
		require.Equal(t, "# Manual spec\n## Testcases\n- "+q, active["spec"]["body"])
		require.Equal(t, []int{1, 3}, deleted)
		require.Equal(t, []map[string]any{checked, caseRow(3, "Old action → old result", false)}, cases)
		mu.Unlock()
		s.navigate(t, "/retry\r", "loaded change")
		mu.Lock()
		require.Equal(t, []int{1, 3}, deleted, "retry must only read")
		require.Len(t, inserts, 1)
		syncFailure = false
		mu.Unlock()
	} else {
		s.waitFor(t, "testcases synchronized")
	}
	drafts, err := filepath.Glob(filepath.Join(os.TempDir(), "mch-project-*.md"))
	require.NoError(t, err)
	if partial {
		require.Len(t, drafts, 1, "failed synchronization retains the owning editor draft")
	} else {
		require.Empty(t, drafts, "successful synchronization cleans its owning editor draft")
	}
	mu.Lock()
	require.Len(t, inserts, 1)
	require.Equal(t, false, inserts[0]["agent_edit"])
	require.Equal(t, []int{1, 3}, deleted)
	if !partial {
		require.Equal(t, []map[string]any{checked}, cases)
	}
	mu.Unlock()
	require.NoError(t, os.WriteFile(editor+".text", []byte("# Human edited brief"), 0o600))
	count := s.output.count("testcases synchronized")
	s.send(t, "/brief\r")
	s.output.waitForCount(t, "testcases synchronized", count+1)
	s.waitFor(t, "LIVE spec progress")
	drafts, err = filepath.Glob(filepath.Join(os.TempDir(), "mch-project-*.md"))
	require.NoError(t, err)
	if partial {
		require.Len(t, drafts, 1, "the earlier failed spec synchronization still owns its draft")
	} else {
		require.Empty(t, drafts, "the successful brief flow cleans its owning editor draft")
	}
	s.send(t, "/return\r")
	s.finishFromChanges(t)
	mu.Lock()
	defer mu.Unlock()
	require.Len(t, inserts, 4)
	require.Equal(t, []any{"spec", "brief", "brief", "spec"}, []any{inserts[0]["doc_type"], inserts[1]["doc_type"], inserts[2]["doc_type"], inserts[3]["doc_type"]})
	require.Equal(t, []any{false, false, true, true}, []any{inserts[0]["agent_edit"], inserts[1]["agent_edit"], inserts[2]["agent_edit"], inserts[3]["agent_edit"]})
	require.Zero(t, creates)
	require.Equal(t, []map[string]any{checked}, cases)
	if partial {
		require.Equal(t, []int{1, 3, 3}, deleted)
	} else {
		require.Equal(t, []int{1, 3}, deleted)
	}
}
