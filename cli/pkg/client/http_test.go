package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPClientPostsToSelectorEndpoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/api/v1/epic/list", r.URL.Path)
		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		assert.Equal(t, map[string]any{"project_id": float64(7)}, payload)
		writeJSON(t, w, []any{epicFixture()})
	}))
	defer server.Close()
	epics, err := NewHTTPClient(server.URL).ListEpics(context.Background(), 7)
	require.NoError(t, err)
	assert.Equal(t, 3, epics[0].ID)
	assert.Equal(t, "Epic Three", epics[0].Name)
}

func TestListEpicsRequiresCurrentProject(t *testing.T) {
	client := NewHTTPClient("http://example.invalid")

	_, err := client.ListEpics(context.Background(), 0)
	require.Error(t, err)

	_, err = client.ListEpics(context.Background(), -1)
	require.Error(t, err)
}

func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	require.NoError(t, json.NewEncoder(w).Encode(value))
}

// These request assertions remain until testcase mutations migrate in P5.
func TestRetainedTestCaseMutationPayloads(t *testing.T) {
	calls := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		calls = append(calls, r.URL.Path)
		expected := map[string]map[string]any{
			"/api/v1/test-case/create":      {"change_id": float64(12), "scenario": "new scenario"},
			"/api/v1/test-case/update":      {"id": float64(31), "scenario": "updated scenario"},
			"/api/v1/test-case/update-done": {"id": float64(31), "done": true},
			"/api/v1/test-case/delete":      {"id": float64(31)},
		}
		require.Equal(t, expected[r.URL.Path], body)
		if r.URL.Path == "/api/v1/test-case/create" {
			w.WriteHeader(http.StatusCreated)
			writeJSON(t, w, map[string]any{"id": 31})
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer server.Close()
	c := NewHTTPClient(server.URL)
	_, err := c.CreateTestCase(context.Background(), 12, "new scenario")
	require.NoError(t, err)
	err = c.UpdateTestCase(context.Background(), 31, "updated scenario")
	require.NoError(t, err)
	err = c.UpdateTestCaseDone(context.Background(), 31, true)
	require.NoError(t, err)
	err = c.DeleteTestCase(context.Background(), 31)
	require.NoError(t, err)
	require.Len(t, calls, 4)
}
