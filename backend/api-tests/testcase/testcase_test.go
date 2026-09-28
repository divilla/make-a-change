package testcase_test

import (
	"mch_api/api-tests/shared"
	"mch_api/internal/domain"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// APIHydra covers the lifecycle; this owned HTTP campaign additionally proves
// timestamp advancement and unchanged parent/created/unrelated state.
func TestTestCaseCurrentStateAndSameValueTimestamps(t *testing.T) {
	c := shared.NewClient(t)
	var project domain.ProjectIDRequest
	require.Equal(t, 201, c.Post(t, "/api/v1/project/create", map[string]any{"name": "P4 timestamp checks"}, &project))
	defer shared.CleanupProject(t, c, project.ID)
	var epic domain.EpicIDRequest
	require.Equal(t, 201, c.Post(t, "/api/v1/epic/create", map[string]any{"project_id": project.ID, "name": "P4 epic"}, &epic))
	var change domain.ChangeIDRequest
	require.Equal(t, 201, c.Post(t, "/api/v1/change/create", map[string]any{"project_id": project.ID, "title": "P4 checks", "brief": "Brief"}, &change))
	require.Equal(t, 204, c.Post(t, "/api/v1/change/update-epic", map[string]any{"id": change.ID, "epic_id": epic.ID}, nil))
	var parentBefore domain.ChangeDetails
	var epicBefore domain.Epic
	require.Equal(t, 200, c.Post(t, "/api/v1/change/get", change, &parentBefore))
	require.Equal(t, 200, c.Post(t, "/api/v1/epic/get", epic, &epicBefore))
	var first, second domain.TestCaseIDRequest
	require.Equal(t, 201, c.Post(t, "/api/v1/test-case/create", domain.TestCaseCreateRequest{ChangeID: change.ID, Scenario: "  First  "}, &first))
	require.Equal(t, 201, c.Post(t, "/api/v1/test-case/create", domain.TestCaseCreateRequest{ChangeID: change.ID, Scenario: "Second"}, &second))
	list := func() []domain.TestCase {
		t.Helper()
		var rows []domain.TestCase
		require.Equal(t, 200, c.Post(t, "/api/v1/test-case/list", domain.TestCaseListRequest{ChangeID: change.ID}, &rows))
		require.NotNil(t, rows)
		return rows
	}
	counts := func(done, total, completed int64) {
		t.Helper()
		var p domain.ChangeDetails
		var e domain.Epic
		require.Equal(t, 200, c.Post(t, "/api/v1/change/get", change, &p))
		require.Equal(t, 200, c.Post(t, "/api/v1/epic/get", epic, &e))
		require.Equal(t, []int64{done, total, completed}, []int64{p.DoneTC, p.TotalTC, p.Completed})
		require.Equal(t, []int64{done, total, completed}, []int64{e.DoneTC, e.TotalTC, e.Completed})
		require.Equal(t, parentBefore.Modified, p.Modified)
		require.Equal(t, epicBefore.Modified, e.Modified)
	}
	rows := list()
	require.Len(t, rows, 2)
	require.Equal(t, first.ID, rows[0].ID)
	require.Equal(t, second.ID, rows[1].ID)
	require.Equal(t, "First", rows[0].Scenario)
	require.False(t, rows[0].Done)
	counts(0, 2, 0)
	before := rows[0]
	for _, op := range []struct {
		path     string
		body     any
		scenario string
		done     bool
	}{
		{"update-done", domain.TestCaseUpdateDoneRequest{ID: first.ID, Done: true}, "First", true},
		{"update-done", domain.TestCaseUpdateDoneRequest{ID: first.ID, Done: true}, "First", true},
		{"update", domain.TestCaseUpdateRequest{ID: first.ID, Scenario: "  Edited  "}, "Edited", true},
		{"update", domain.TestCaseUpdateRequest{ID: first.ID, Scenario: "Edited"}, "Edited", true},
	} {
		require.Equal(t, 204, c.Post(t, "/api/v1/test-case/"+op.path, op.body, nil))
		after := list()[0]
		require.True(t, after.Modified.After(before.Modified))
		require.Equal(t, before.Created, after.Created)
		require.Equal(t, before.ChangeID, after.ChangeID)
		require.Equal(t, before.ID, after.ID)
		require.Equal(t, op.scenario, after.Scenario)
		require.Equal(t, op.done, after.Done)
		require.Equal(t, rows[1], list()[1])
		before = after
		counts(1, 2, 50)
	}
	require.Equal(t, 204, c.Post(t, "/api/v1/test-case/update-done", domain.TestCaseUpdateDoneRequest{ID: second.ID, Done: true}, nil))
	counts(2, 2, 100)
	require.Equal(t, 204, c.Post(t, "/api/v1/test-case/update-done", domain.TestCaseUpdateDoneRequest{ID: second.ID, Done: false}, nil))
	counts(1, 2, 50)
	require.Equal(t, 204, c.Post(t, "/api/v1/test-case/delete", first, nil))
	counts(0, 1, 0)
	require.Equal(t, second.ID, list()[0].ID)
	require.Equal(t, 404, c.Post(t, "/api/v1/test-case/delete", first, nil))
	require.Equal(t, 204, c.Post(t, "/api/v1/test-case/delete", second, nil))
	counts(0, 0, 0)
	require.Empty(t, list())
	require.Equal(t, 404, c.Post(t, "/api/v1/test-case/update-change", map[string]any{"id": second.ID, "change_id": change.ID}, nil))
}

func TestTestCaseRejectsInvalidInputAndMissingRows(t *testing.T) {
	client := shared.NewClient(t)

	status := client.Post(t, "/api/v1/test-case/list", map[string]any{}, nil)
	assert.Equal(t, http.StatusBadRequest, status)

	status = client.Post(t, "/api/v1/test-case/list", map[string]any{"change_id": 999999999}, nil)
	assert.Equal(t, http.StatusNotFound, status)

	status = client.Post(t, "/api/v1/test-case/create", map[string]any{
		"change_id": -1,
		"scenario":  "orphan test case",
	}, nil)
	assert.Equal(t, http.StatusBadRequest, status)

	status = client.Post(t, "/api/v1/test-case/create", map[string]any{
		"change_id": -1,
		"scenario":  "   ",
	}, nil)
	assert.Equal(t, http.StatusBadRequest, status)

	status = client.Post(t, "/api/v1/test-case/create", map[string]any{
		"change_id": 999999999,
		"scenario":  "missing change test case",
	}, nil)
	assert.Equal(t, http.StatusNotFound, status)

	status = client.Post(t, "/api/v1/test-case/update", map[string]any{
		"id":       999999999,
		"scenario": "missing test case",
	}, nil)
	assert.Equal(t, http.StatusNotFound, status)

	status = client.Post(t, "/api/v1/test-case/update", map[string]any{
		"id":       999999999,
		"scenario": "   ",
	}, nil)
	assert.Equal(t, http.StatusBadRequest, status)

	status = client.Post(t, "/api/v1/test-case/update-done", map[string]any{}, nil)
	assert.Equal(t, http.StatusBadRequest, status)

	status = client.Post(t, "/api/v1/test-case/update-done", map[string]any{"id": 999999999, "done": true}, nil)
	assert.Equal(t, http.StatusNotFound, status)

	status = client.Post(t, "/api/v1/test-case/delete", map[string]any{}, nil)
	assert.Equal(t, http.StatusBadRequest, status)

	status = client.Post(t, "/api/v1/test-case/delete", map[string]any{"id": 999999999}, nil)
	assert.Equal(t, http.StatusNotFound, status)
}
