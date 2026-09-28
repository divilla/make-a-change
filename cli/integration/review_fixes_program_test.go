package integration_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testProgramSlashPrefixedTestCase(t *testing.T) {
	for _, scenario := range []string{"/api/v1/health returns 200", "/cancel"} {
		t.Run(scenario, func(t *testing.T) { testProgramTestCaseRetry(t, scenario, true) })
	}
	t.Run("ordinary prompt", func(t *testing.T) {
		testProgramTestCaseRetry(t, "/api/v1/health returns 200", false)
	})
}

func testProgramTestCaseRetry(t *testing.T, scenario string, useEditor bool) {
	var mu sync.Mutex
	var saved []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		change := map[string]any{"id": 12, "project_id": 7, "title": "Existing"}
		switch r.URL.Path {
		case "/api/v1/options/change-phases-list", "/api/v1/options/change-types-list":
			writeProgramJSON(w, []any{})
		case "/api/v1/project/get":
			writeProgramJSON(w, map[string]any{"id": 7, "name": "Program Project"})
		case "/api/v1/change/list":
			writeProgramJSON(w, []any{change})
		case "/api/v1/change/get":
			writeProgramJSON(w, change)
		case "/api/v1/test-case/create":
			var body struct {
				ChangeID int    `json:"change_id"`
				Scenario string `json:"scenario"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, 12, body.ChangeID)
			saved = append(saved, body.Scenario)
			if len(saved) == 1 {
				http.Error(w, "save rejected", http.StatusInternalServerError)
				return
			}
			change["test_cases"] = []any{map[string]any{"id": 31, "change_id": 12, "scenario": body.Scenario}}
			writeProgramJSON(w, change)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	root := t.TempDir()
	writeProgramConfig(t, root, server.URL)
	session := startProgram(t, root, scenario)
	session.navigate(t, "/changes\r", "Rows 1-1 of 1")
	session.navigate(t, "\r", "loaded change")
	session.navigate(t, "/new-testcase\r", "TestCaseCreateScreen")
	if useEditor {
		session.send(t, "\x05")
	} else {
		session.send(t, "\x1b[200~"+scenario+"\x1b[201~\r")
	}
	session.waitFor(t, "status save failed")
	saveFrames := session.output.count("status save")
	session.send(t, "\r")
	session.output.waitForCount(t, "status save", saveFrames+1)
	session.finishFromDetails(t)
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []string{scenario, scenario}, saved)
	assert.NotContains(t, session.output.String(), "unknown command")
	if useEditor {
		assert.Contains(t, session.output.String(), "\x1b[2J")
	}
}
