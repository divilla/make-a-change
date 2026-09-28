package integration_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Ordinary editor and save-order assertions rehomed from the removed agent programs.
func TestCLIProgramOrdinaryDocumentEditor(t *testing.T) {
	for _, failed := range []bool{false, true} {
		t.Run(fmt.Sprintf("detail load failed=%t", failed), func(t *testing.T) {
			testDocumentEditorWaitsForDetail(t, failed)
		})
	}
	for _, field := range []string{"def", "spec", "pr"} {
		for _, outcome := range []string{"saved", "follow-up failure", "unchanged", "retry enter", "retry unchanged editor"} {
			t.Run(fmt.Sprintf("%s/%s", field, outcome), func(t *testing.T) {
				failure := outcome == "follow-up failure"
				unchanged := outcome == "unchanged"
				retry := strings.HasPrefix(outcome, "retry")
				var mu sync.Mutex
				original := "Types: feature\n\n```make\nbuild:\n\tgo build ./...\n```\n"
				saved := original
				paths := []string{}
				// A trailing newline alone is still a real edit.
				edited := original + "\n"
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					mu.Lock()
					defer mu.Unlock()
					w.Header().Set("Content-Type", "application/json")
					change := map[string]any{"id": 12, "project_id": 7, "title": "Existing", "def": "Original text", "spec": "Original text", "pr": "Original text"}
					change[field] = saved
					change["change_types"] = []string{"bugfix"}
					switch r.URL.Path {
					case "/api/v1/project/config":
						writeProgramJSON(w, programProjectConfig())
					case "/api/v1/project/details":
						writeProgramJSON(w, programProject(7, "Program Project"))
					case "/api/v1/change/list":
						writeProgramJSON(w, []any{change})
					case "/api/v1/change/get":
						writeProgramJSON(w, change)
					case "/api/v1/change/update-" + field:
						var body map[string]any
						require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
						assert.Equal(t, false, body["agent_edit"])
						paths = append(paths, r.URL.Path)
						if retry && len(paths) == 1 {
							http.Error(w, "document save rejected", http.StatusInternalServerError)
							return
						}
						saved = body[field].(string)
						change[field] = saved
						writeProgramJSON(w, change)
					case "/api/v1/change/update-change-types":
						paths = append(paths, r.URL.Path)
						if failure {
							http.Error(w, "type save failed", 500)
							return
						}
						writeProgramJSON(w, change)
					default:
						t.Errorf("unexpected path %s", r.URL.Path)
						http.NotFound(w, r)
					}
				}))
				t.Cleanup(server.Close)
				root := t.TempDir()
				writeProgramConfig(t, root, server.URL)
				session := startProgram(t, root, edited)
				capture := filepath.Join(root, "editor-input.md")
				editorPath := filepath.Join(root, "editor-path.txt")
				script := filepath.Join(root, "editor.sh")
				command := "#!/bin/sh\nset -eu\nalready_opened=0\n[ ! -f \"$CAPTURE\" ] || already_opened=1\ncp \"$1\" \"$CAPTURE\"\nprintf '%s' \"$1\" > \"$EDITOR_PATH\"\n"
				if !unchanged {
					command += "if [ \"$already_opened\" = 0 ]; then printf '\\n' >> \"$1\"; fi\n"
				}
				require.NoError(t, os.WriteFile(script, []byte(command), 0o755))
				t.Setenv("CAPTURE", capture)
				t.Setenv("EDITOR_PATH", editorPath)
				t.Setenv("EDITOR", script)
				session.navigate(t, "/changes\r", "Rows 1-1 of 1")
				session.navigate(t, "\r", "loaded change")
				if field == "spec" {
					session.send(t, "/edit-spec\r")
				} else {
					down := 8
					if field == "pr" {
						down = 10
					}
					session.send(t, strings.Repeat("\x1b[B", down)+"\r")
				}
				saveFrames := 0
				if retry {
					session.waitFor(t, "status save failed")
					saveFrames = session.output.count("status save")
					if outcome == "retry unchanged editor" {
						session.send(t, "\x05")
					} else {
						session.send(t, "\r")
					}
				}
				if unchanged {
					session.waitFor(t, "status unchanged")
				} else if failure {
					session.waitFor(t, "type update failed")
				} else {
					session.waitFor(t, "status save")
					if retry {
						session.output.waitForCount(t, "status save", saveFrames+1)
					}
				}
				mu.Lock()
				if unchanged {
					assert.Equal(t, original, saved)
					assert.Empty(t, paths, "unchanged exit must not write the document or reset selected types")
				} else {
					assert.Equal(t, edited, saved)
					expected := []string{"/api/v1/change/update-" + field, "/api/v1/change/update-change-types"}
					if retry {
						expected = append([]string{"/api/v1/change/update-" + field}, expected...)
					}
					assert.Equal(t, expected, paths)
				}
				mu.Unlock()
				expectedSeed := original
				if outcome == "retry unchanged editor" {
					expectedSeed = edited
				}
				assert.Equal(t, expectedSeed, readFile(t, capture))
				assert.NoFileExists(t, readFile(t, editorPath))
				assert.Contains(t, session.output.String(), "\x1b[2J")
				assert.NoDirExists(t, filepath.Join(root, ".mch/tmp"))
				session.finishFromDetails(t)
			})
		}
	}
}

func testDocumentEditorWaitsForDetail(t *testing.T, failed bool) {
	t.Helper()
	release := make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	var gets atomic.Int32
	original := "Existing spec\n\tkeep this document\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/project/config":
			writeProgramJSON(w, programProjectConfig())
		case "/api/v1/project/details":
			writeProgramJSON(w, programProject(7, "Program Project"))
		case "/api/v1/change/list":
			writeProgramJSON(w, []any{map[string]any{"id": 12, "title": "Existing"}})
		case "/api/v1/change/get":
			if gets.Add(1) == 1 {
				<-release
				if failed {
					http.Error(w, "detail unavailable", http.StatusInternalServerError)
					return
				}
			}
			writeProgramJSON(w, map[string]any{"id": 12, "title": "Existing", "spec": original})
		default:
			t.Errorf("unexpected request (including document writes): %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	t.Cleanup(unblock)
	root := t.TempDir()
	writeProgramConfig(t, root, server.URL)
	session := startProgram(t, root, "must never replace the document")
	capture := filepath.Join(root, "editor-input.md")
	script := filepath.Join(root, "capture-editor.sh")
	require.NoError(t, os.WriteFile(script, []byte("#!/bin/sh\nset -eu\ncp \"$1\" \"$CAPTURE\"\n"), 0o755))
	t.Setenv("CAPTURE", capture)
	t.Setenv("EDITOR", script)
	session.navigate(t, "/changes\r", "Rows 1-1 of 1")
	session.navigate(t, "\r", "selected Existing")
	session.navigate(t, "/edit-spec\r", "Load change details before editing")
	assert.NoFileExists(t, capture, "pending detail must not launch the editor")
	unblock()
	if failed {
		session.waitFor(t, "status load failed")
		frames := session.output.count("Load change details before editing")
		session.send(t, "/edit-spec\r")
		session.output.waitForCount(t, "Load change details before editing", frames+1)
		assert.NoFileExists(t, capture, "failed detail must not launch the editor")
		session.navigate(t, "/return\r", "Rows 1-1 of 1")
		session.send(t, "\r")
	}
	session.waitFor(t, "loaded change")
	session.navigate(t, "/edit-spec\r", "status unchanged")
	assert.Equal(t, original, readFile(t, capture))
	session.finishFromDetails(t)
}
