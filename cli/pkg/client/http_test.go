package client

import (
	"cli/internal/dto"
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

func TestHTTPClientChangeListCreateUpdateAndGetPayloads(t *testing.T) {
	var paths []string
	var payloads []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		paths = append(paths, r.URL.Path)

		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		payloads = append(payloads, payload)

		switch r.URL.Path {
		case "/api/v1/change/list":
			writeJSON(t, w, []map[string]any{{
				"id":           12,
				"ref":          3,
				"slug":         "new-change",
				"title":        "New Change",
				"change_phase": "backlog",
				"change_types": []string{"feature", "test"},
				"epic_id":      5,
				"epic_name":    "Epic Five",
				"def":          "def",
				"spec":         "spec",
				"pr":           "pr body",
				"pr_url":       "https://example.test/pr/12",
				"agent_edit":   true,
				"open":         true,
				"done_tc":      2,
				"total_tc":     5,
				"completed":    40,
				"modified":     "2026-06-29T10:45:00Z",
			}})
		case "/api/v1/change/create":
			writeJSON(t, w, map[string]any{"id": 12, "title": payload["title"], "def": payload["def"]})
		case "/api/v1/change/update-title":
			writeJSON(t, w, map[string]any{"id": payload["id"], "title": payload["title"]})
		case "/api/v1/change/update-def":
			writeJSON(t, w, map[string]any{"id": payload["id"], "def": payload["def"], "agent_edit": payload["agent_edit"]})
		case "/api/v1/change/update-spec":
			writeJSON(t, w, map[string]any{"id": payload["id"], "spec": payload["spec"], "agent_edit": payload["agent_edit"]})
		case "/api/v1/change/update-pr":
			writeJSON(t, w, map[string]any{"id": payload["id"], "pr": payload["pr"], "agent_edit": payload["agent_edit"]})
		case "/api/v1/change/update-pr-url":
			writeJSON(t, w, map[string]any{"id": payload["id"], "pr_url": payload["pr_url"]})
		case "/api/v1/change/update-change-types":
			writeJSON(t, w, map[string]any{"id": payload["id"], "change_types": payload["change_types"]})
		case "/api/v1/change/update-phase":
			writeJSON(t, w, map[string]any{"id": payload["id"], "change_phase": payload["change_phase"]})
		case "/api/v1/change/update-open":
			writeJSON(t, w, map[string]any{"id": payload["id"], "open": payload["open"]})
		case "/api/v1/change/update-epic":
			writeJSON(t, w, map[string]any{"id": payload["id"], "epic_id": payload["epic_id"]})
		case "/api/v1/test-case/create":
			writeJSON(t, w, map[string]any{"change": map[string]any{"id": payload["change_id"]}, "test_case": map[string]any{"id": 31, "scenario": payload["scenario"], "change_id": payload["change_id"]}})
		case "/api/v1/test-case/update":
			writeJSON(t, w, map[string]any{"change": map[string]any{"id": 12}, "test_case": map[string]any{"id": payload["id"], "scenario": payload["scenario"], "change_id": 12}})
		case "/api/v1/test-case/update-done":
			writeJSON(t, w, map[string]any{"change": map[string]any{"id": 12}, "test_case": map[string]any{"id": payload["id"], "done": payload["done"]}})
		case "/api/v1/test-case/delete":
			writeJSON(t, w, map[string]any{"change": map[string]any{"id": 12}, "test_cases": []map[string]any{}})
		case "/api/v1/change/delete":
			w.WriteHeader(http.StatusNoContent)
		case "/api/v1/change/get":
			writeJSON(t, w, map[string]any{
				"change": map[string]any{
					"id":           payload["id"],
					"ref":          3,
					"slug":         "new-change",
					"title":        "Fetched Change",
					"change_phase": "backlog",
					"change_types": []any{"feature"},
					"def":          "fetched definition",
					"spec":         "fetched spec",
					"pr":           "fetched pr body",
					"pr_url":       "https://example.test/pr/12",
					"agent_edit":   true,
					"open":         true,
				},
				"test_cases": []map[string]any{{
					"id":        31,
					"scenario":  "first scenario",
					"done":      true,
					"change_id": payload["id"],
				}},
			})
		default:
			require.Failf(t, "unexpected path", "path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL)

	listed, err := client.ListChangeRows("7")
	require.NoError(t, err)
	require.Len(t, listed, 1)
	assert.Equal(t, dto.Change{
		ID:          "12",
		Ref:         "3",
		Slug:        "new-change",
		EpicID:      "5",
		EpicName:    "Epic Five",
		ChangePhase: "backlog",
		ChangeTypes: []string{"feature", "test"},
		Title:       "New Change",
		Def:         "def",
		Spec:        "spec",
		PR:          "pr body",
		PRUrl:       "https://example.test/pr/12",
		AgentEdit:   true,
		Open:        true,
		Done:        2,
		Total:       5,
		Completed:   40,
		Modified:    "2026-06-29T10:45:00Z",
	}, listed[0])

	created, err := client.CreateChange(dto.ChangeCreateInput{
		ProjectID: 7,
		RefUUID:   "0198a86f-9b8a-7d89-ae5b-6f25b528b04c",
		Title:     "New Change",
		Def:       "# New Change\n\nInitial definition",
	})
	require.NoError(t, err)
	assert.Equal(t, dto.Change{ID: "12", Title: "New Change", Def: "# New Change\n\nInitial definition"}, created)

	_, err = client.UpdateChangeTitle(12, "Renamed")
	require.NoError(t, err)
	_, err = client.UpdateChangeDef(12, "Full definition", false)
	require.NoError(t, err)
	_, err = client.UpdateChangeDef(12, "Rewritten definition", true)
	require.NoError(t, err)
	fullSpec := "Full spec"
	_, err = client.UpdateChangeSpec(12, fullSpec, false)
	require.NoError(t, err)
	_, err = client.UpdateChangePR(12, "Full PR body", false)
	require.NoError(t, err)
	_, err = client.UpdateChangePRUrl(12, "https://example.test/pr/99")
	require.NoError(t, err)
	_, err = client.UpdateChangeTypes(12, []string{"docs"})
	require.NoError(t, err)
	_, err = client.UpdateChangePhase(12, "stage")
	require.NoError(t, err)
	_, err = client.UpdateChangeOpen(12, false)
	require.NoError(t, err)
	_, err = client.UpdateChangeEpic(12, nil)
	require.NoError(t, err)
	_, err = client.CreateTestCase(12, "new scenario")
	require.NoError(t, err)
	_, err = client.UpdateTestCase(31, "updated scenario")
	require.NoError(t, err)
	_, err = client.UpdateTestCaseDone(31, true)
	require.NoError(t, err)
	_, err = client.DeleteTestCase(31)
	require.NoError(t, err)
	require.NoError(t, client.DeleteChange(12))
	fetched, err := client.GetChange(12)
	require.NoError(t, err)
	assert.Equal(t, dto.Change{
		ID:          "12",
		Ref:         "3",
		Slug:        "new-change",
		ChangePhase: "backlog",
		ChangeTypes: []string{"feature"},
		Title:       "Fetched Change",
		Def:         "fetched definition",
		Spec:        "fetched spec",
		PR:          "fetched pr body",
		PRUrl:       "https://example.test/pr/12",
		AgentEdit:   true,
		Open:        true,
		TestCases: []dto.TestCase{{
			ID:       "31",
			Scenario: "first scenario",
			Done:     true,
			ChangeID: "12",
		}},
	}, fetched)

	assert.Equal(t, []string{
		"/api/v1/change/list",
		"/api/v1/change/create",
		"/api/v1/change/update-title",
		"/api/v1/change/update-def",
		"/api/v1/change/update-def",
		"/api/v1/change/update-spec",
		"/api/v1/change/update-pr",
		"/api/v1/change/update-pr-url",
		"/api/v1/change/update-change-types",
		"/api/v1/change/update-phase",
		"/api/v1/change/update-open",
		"/api/v1/change/update-epic",
		"/api/v1/test-case/create",
		"/api/v1/test-case/update",
		"/api/v1/test-case/update-done",
		"/api/v1/test-case/delete",
		"/api/v1/change/delete",
		"/api/v1/change/get",
	}, paths)
	assert.Equal(t, map[string]any{"project_id": float64(7)}, payloads[0])
	assert.Equal(t, map[string]any{
		"project_id": float64(7),
		"ref_uuid":   "0198a86f-9b8a-7d89-ae5b-6f25b528b04c",
		"title":      "New Change",
		"def":        "# New Change\n\nInitial definition",
	}, payloads[1])
	assert.Equal(t, map[string]any{"id": float64(12), "title": "Renamed"}, payloads[2])
	assert.Equal(t, map[string]any{"id": float64(12), "def": "Full definition", "agent_edit": false}, payloads[3])
	assert.Equal(t, map[string]any{"id": float64(12), "def": "Rewritten definition", "agent_edit": true}, payloads[4])
	assert.Equal(t, map[string]any{"id": float64(12), "spec": "Full spec", "agent_edit": false}, payloads[5])
	assert.Equal(t, map[string]any{"id": float64(12), "pr": "Full PR body", "agent_edit": false}, payloads[6])
	assert.Equal(t, map[string]any{"id": float64(12), "pr_url": "https://example.test/pr/99"}, payloads[7])
	assert.Equal(t, map[string]any{"id": float64(12), "change_types": []any{"docs"}}, payloads[8])
	assert.Equal(t, map[string]any{"id": float64(12), "change_phase": "stage"}, payloads[9])
	assert.Equal(t, map[string]any{"id": float64(12), "open": false}, payloads[10])
	assert.Equal(t, map[string]any{"id": float64(12), "epic_id": nil}, payloads[11])
	assert.Equal(t, map[string]any{"change_id": float64(12), "scenario": "new scenario"}, payloads[12])
	assert.Equal(t, map[string]any{"id": float64(31), "scenario": "updated scenario"}, payloads[13])
	assert.Equal(t, map[string]any{"id": float64(31), "done": true}, payloads[14])
	assert.Equal(t, map[string]any{"id": float64(31)}, payloads[15])
	assert.Equal(t, map[string]any{"id": float64(12)}, payloads[16])
	assert.Equal(t, map[string]any{"id": float64(12)}, payloads[17])
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
