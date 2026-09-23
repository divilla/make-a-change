package testcase_test

import (
	"aipm/api-tests/shared"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type project struct {
	ID int `json:"id"`
}

type epic struct {
	ID        int   `json:"id"`
	DoneTC    int16 `json:"done_tc"`
	TotalTC   int16 `json:"total_tc"`
	Completed int16 `json:"completed"`
}

type change struct {
	ID        int   `json:"id"`
	Version   int16 `json:"version"`
	DoneTC    int16 `json:"done_tc"`
	TotalTC   int16 `json:"total_tc"`
	Completed int16 `json:"completed"`
}

type testCase struct {
	ID       int    `json:"id"`
	Version  int16  `json:"version"`
	ChangeID int    `json:"change_id"`
	Scenario string `json:"scenario"`
	Done     bool   `json:"done"`
}

type mutation struct {
	TestCase  *testCase  `json:"test_case"`
	Change    change     `json:"change"`
	TestCases []testCase `json:"test_cases"`
}

func TestTestCaseCRUDRecalculatesChangeAndEpicCompleteness(t *testing.T) {
	client := shared.NewClient(t)

	projectID := createProject(t, client)
	defer shared.CleanupProject(t, client, projectID)
	epicID := createEpic(t, client, projectID)
	changeID := createChange(t, client, projectID, &epicID)

	var listed []testCase
	status := client.Post(t, "/api/v1/test-case/list", map[string]any{"change_id": changeID}, &listed)
	require.Equal(t, http.StatusOK, status)
	assert.Empty(t, listed)

	first := createTestCase(t, client, changeID, "Add test case create endpoint test.")
	second := createTestCase(t, client, changeID, "Add test case update endpoint test.")
	assert.Equal(t, changeID, first.ChangeID)
	assert.False(t, first.Done)
	assert.False(t, second.Done)

	status = client.Post(t, "/api/v1/test-case/list", map[string]any{"change_id": changeID}, &listed)
	require.Equal(t, http.StatusOK, status)
	require.Len(t, listed, 2)

	var updated mutation
	status = client.Post(t, "/api/v1/test-case/update-done", map[string]any{"id": first.ID, "done": true}, &updated)
	require.Equal(t, http.StatusOK, status)
	require.NotNil(t, updated.TestCase)
	assert.True(t, updated.TestCase.Done)
	assert.Equal(t, int16(50), updated.Change.Completed)
	require.Len(t, updated.TestCases, 2)
	assert.Equal(t, first.Version, updated.TestCase.Version)
	first = *updated.TestCase
	assertEpicCompleteness(t, client, epicID, 1, 2, 50)

	status = client.Post(t, "/api/v1/test-case/update-done", map[string]any{"id": second.ID, "done": true}, &updated)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, int16(100), updated.Change.Completed)
	second = *updated.TestCase

	status = client.Post(t, "/api/v1/test-case/update", map[string]any{
		"id":       second.ID,
		"scenario": "Updated test case scenario.",
	}, &updated)
	require.Equal(t, http.StatusOK, status)
	require.NotNil(t, updated.TestCase)
	assert.Equal(t, "Updated test case scenario.", updated.TestCase.Scenario)
	assert.Equal(t, second.Version+1, updated.TestCase.Version)
	second = *updated.TestCase

	status = client.Post(t, "/api/v1/test-case/update", map[string]any{
		"id":       second.ID,
		"scenario": "Updated test case scenario again.",
	}, &updated)
	require.Equal(t, http.StatusOK, status)
	require.NotNil(t, updated.TestCase)
	assert.Equal(t, "Updated test case scenario again.", updated.TestCase.Scenario)
	assert.Equal(t, second.Version+1, updated.TestCase.Version)
	second = *updated.TestCase

	status = client.Post(t, "/api/v1/test-case/update-done", map[string]any{"id": second.ID, "done": false}, &updated)
	require.Equal(t, http.StatusOK, status)
	require.NotNil(t, updated.TestCase)
	assert.False(t, updated.TestCase.Done)
	assert.Equal(t, int16(50), updated.Change.Completed)
	assert.Equal(t, second.Version, updated.TestCase.Version)

	newChangeID := createChange(t, client, projectID, &epicID)
	previousVersion := updated.TestCase.Version
	status = client.Post(t, "/api/v1/test-case/update-change", map[string]any{
		"id": second.ID, "change_id": newChangeID,
	}, &updated)
	require.Equal(t, http.StatusOK, status)
	require.NotNil(t, updated.TestCase)
	assert.Equal(t, newChangeID, updated.TestCase.ChangeID)
	assert.Equal(t, previousVersion, updated.TestCase.Version)
	assert.Equal(t, newChangeID, updated.Change.ID)

	var deleted mutation
	status = client.Post(t, "/api/v1/test-case/delete", map[string]any{"id": first.ID}, &deleted)
	require.Equal(t, http.StatusOK, status)
	assert.Nil(t, deleted.TestCase)
	assert.Equal(t, int16(0), deleted.Change.Completed)
	assert.Empty(t, deleted.TestCases)

	status = client.Post(t, "/api/v1/test-case/delete", map[string]any{"id": first.ID}, nil)
	assert.Equal(t, http.StatusNotFound, status)
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

	status = client.Post(t, "/api/v1/test-case/update-change", map[string]any{"id": 999999999}, nil)
	assert.Equal(t, http.StatusBadRequest, status)

	status = client.Post(t, "/api/v1/test-case/update-change", map[string]any{
		"id": 999999999, "change_id": 999999999,
	}, nil)
	assert.Equal(t, http.StatusNotFound, status)

	status = client.Post(t, "/api/v1/test-case/delete", map[string]any{}, nil)
	assert.Equal(t, http.StatusBadRequest, status)

	status = client.Post(t, "/api/v1/test-case/delete", map[string]any{"id": 999999999}, nil)
	assert.Equal(t, http.StatusNotFound, status)
}

func TestTestCaseDeleteLastItemReturnsZeroCompleted(t *testing.T) {
	client := shared.NewClient(t)
	projectID := createProject(t, client)
	defer shared.CleanupProject(t, client, projectID)
	changeID := createChange(t, client, projectID, nil)
	testCase := createTestCase(t, client, changeID, "Delete the final test case transactionally.")

	var deleted mutation
	status := client.Post(t, "/api/v1/test-case/delete", map[string]any{"id": testCase.ID}, &deleted)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, int16(0), deleted.Change.Completed)
	assert.Equal(t, int16(0), deleted.Change.DoneTC)
	assert.Equal(t, int16(0), deleted.Change.TotalTC)
	assert.Empty(t, deleted.TestCases)
}

func TestTestCaseDoneUpdateDoesNotIncrementVersion(t *testing.T) {
	client := shared.NewClient(t)
	projectID := createProject(t, client)
	defer shared.CleanupProject(t, client, projectID)
	changeID := createChange(t, client, projectID, nil)
	testCase := createTestCase(t, client, changeID, "Done changes preserve the version.")

	var updated mutation
	status := client.Post(t, "/api/v1/test-case/update-done", map[string]any{"id": testCase.ID, "done": true}, &updated)
	require.Equal(t, http.StatusOK, status)
	require.NotNil(t, updated.TestCase)
	assert.Equal(t, testCase.Version, updated.TestCase.Version)
	assert.True(t, updated.TestCase.Done)
}

func createProject(t *testing.T, client *shared.Client) int {
	t.Helper()
	var created project
	status := client.Post(t, "/api/v1/project/create", map[string]string{
		"name": fmt.Sprintf("api-test-test-case-project-%d", time.Now().UnixNano()),
	}, &created)
	require.Equal(t, http.StatusCreated, status)
	require.NotEmpty(t, created.ID)
	return created.ID
}

func createEpic(t *testing.T, client *shared.Client, projectID int) int {
	t.Helper()
	var created epic
	status := client.Post(t, "/api/v1/epic/create", map[string]any{
		"project_id": projectID,
		"name":       fmt.Sprintf("api-test-test-case-epic-%d", time.Now().UnixNano()),
	}, &created)
	require.Equal(t, http.StatusCreated, status)
	require.NotEmpty(t, created.ID)
	return created.ID
}

func createChange(t *testing.T, client *shared.Client, projectID int, epicID *int) int {
	t.Helper()
	var created change
	status := client.Post(t, "/api/v1/change/create", map[string]any{
		"project_id": projectID,
		"title":      fmt.Sprintf("api-test-test-case-change-%d", time.Now().UnixNano()),
		"def":        "Test case def",
	}, &created)
	require.Equal(t, http.StatusCreated, status)
	require.NotEmpty(t, created.ID)
	if epicID != nil {
		status = client.Post(t, "/api/v1/change/update-epic", map[string]any{"id": created.ID, "epic_id": *epicID}, &created)
		require.Equal(t, http.StatusOK, status)
	}
	assert.Equal(t, int16(0), created.Completed)
	return created.ID
}

func createTestCase(t *testing.T, client *shared.Client, changeID int, scenario string) testCase {
	t.Helper()
	var created mutation
	status := client.Post(t, "/api/v1/test-case/create", map[string]any{
		"change_id": changeID,
		"scenario":  scenario,
	}, &created)
	require.Equal(t, http.StatusCreated, status)
	require.NotNil(t, created.TestCase)
	require.NotEmpty(t, created.TestCase.ID)
	assert.Equal(t, int16(0), created.Change.Completed)
	return *created.TestCase
}

func assertEpicCompleteness(t *testing.T, client *shared.Client, id int, doneTC, totalTC, completed int16) {
	t.Helper()
	var fetched epic
	status := client.Post(t, "/api/v1/epic/get", map[string]any{"id": id}, &fetched)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, doneTC, fetched.DoneTC)
	assert.Equal(t, totalTC, fetched.TotalTC)
	assert.Equal(t, completed, fetched.Completed)
}
