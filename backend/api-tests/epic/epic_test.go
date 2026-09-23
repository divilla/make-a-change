package epic_test

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
	ID          int       `json:"id"`
	Version     int16     `json:"version"`
	ProjectID   int       `json:"project_id"`
	Name        string    `json:"name"`
	DoneTC      int16     `json:"done_tc"`
	TotalTC     int16     `json:"total_tc"`
	Completed   int16     `json:"completed"`
	ChangeCount int       `json:"change_count"`
	Created     time.Time `json:"created"`
	Modified    time.Time `json:"modified"`
}

type change struct {
	ID int `json:"id"`
}

func TestEpicCRUDAndProjectScopedList(t *testing.T) {
	client := shared.NewClient(t)

	projectID := createProject(t, client)
	defer shared.CleanupProject(t, client, projectID)
	otherProjectID := createProject(t, client)
	defer shared.CleanupProject(t, client, otherProjectID)

	name := fmt.Sprintf("api-test-epic-%d", time.Now().UnixNano())
	var created epic
	status := client.Post(t, "/api/v1/epic/create", map[string]any{
		"project_id": projectID,
		"name":       " " + name + " ",
	}, &created)
	require.Equal(t, http.StatusCreated, status)
	require.NotEmpty(t, created.ID)
	assert.Equal(t, projectID, created.ProjectID)
	assert.Equal(t, name, created.Name)
	assert.Equal(t, int16(0), created.Version)
	assert.Equal(t, int16(0), created.DoneTC)
	assert.Equal(t, int16(0), created.TotalTC)
	assert.Equal(t, int16(0), created.Completed)
	assert.Equal(t, 0, created.ChangeCount)
	assert.False(t, created.Created.IsZero())
	assert.False(t, created.Modified.IsZero())

	otherEpic := createEpic(t, client, otherProjectID)

	var listed []epic
	status = client.Post(t, "/api/v1/epic/list", map[string]any{"project_id": projectID}, &listed)
	require.Equal(t, http.StatusOK, status)
	require.Len(t, listed, 1)
	assert.Equal(t, created.ID, listed[0].ID)
	assert.Equal(t, projectID, listed[0].ProjectID)
	assert.NotEqual(t, otherEpic, listed[0].ID)

	status = client.Post(t, "/api/v1/epic/list", map[string]any{"project_id": otherProjectID}, &listed)
	require.Equal(t, http.StatusOK, status)
	require.Len(t, listed, 1)
	assert.Equal(t, otherEpic, listed[0].ID)
	assert.Equal(t, otherProjectID, listed[0].ProjectID)

	var fetched epic
	status = client.Post(t, "/api/v1/epic/get", map[string]any{"id": created.ID}, &fetched)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, created, fetched)

	updatedName := name + "-updated"
	var updated epic
	status = client.Post(t, "/api/v1/epic/update", map[string]any{
		"id":   created.ID,
		"name": " " + updatedName + " ",
	}, &updated)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, created.ID, updated.ID)
	assert.Equal(t, projectID, updated.ProjectID)
	assert.Equal(t, updatedName, updated.Name)
	assert.Equal(t, created.Version+1, updated.Version)
	assert.Equal(t, 0, updated.ChangeCount)
	assert.False(t, updated.Modified.Before(updated.Created))

	secondUpdatedName := name + "-updated-again"
	var secondUpdated epic
	status = client.Post(t, "/api/v1/epic/update", map[string]any{
		"id":   created.ID,
		"name": " " + secondUpdatedName + " ",
	}, &secondUpdated)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, secondUpdatedName, secondUpdated.Name)
	assert.Equal(t, updated.Version+1, secondUpdated.Version)

	status = client.Post(t, "/api/v1/epic/delete", map[string]any{"id": created.ID}, nil)
	require.Equal(t, http.StatusNoContent, status)

	status = client.Post(t, "/api/v1/epic/get", map[string]any{"id": created.ID}, nil)
	assert.Equal(t, http.StatusNotFound, status)
}

func TestEpicDeleteRejectsEpicsWithChanges(t *testing.T) {
	client := shared.NewClient(t)

	projectID := createProject(t, client)
	defer shared.CleanupProject(t, client, projectID)
	epicID := createEpic(t, client, projectID)

	var createdChange change
	status := client.Post(t, "/api/v1/change/create", map[string]any{
		"project_id": projectID,
		"title":      fmt.Sprintf("api-test-epic-conflict-change-%d", time.Now().UnixNano()),
		"def":        "Epic conflict def",
	}, &createdChange)
	require.Equal(t, http.StatusCreated, status)
	require.NotEmpty(t, createdChange.ID)
	status = client.Post(t, "/api/v1/change/update-epic", map[string]any{"id": createdChange.ID, "epic_id": epicID}, &createdChange)
	require.Equal(t, http.StatusOK, status)

	var listed []epic
	status = client.Post(t, "/api/v1/epic/list", map[string]any{"project_id": projectID}, &listed)
	require.Equal(t, http.StatusOK, status)
	require.Len(t, listed, 1)
	assert.Equal(t, 1, listed[0].ChangeCount)

	var fetched epic
	status = client.Post(t, "/api/v1/epic/get", map[string]any{"id": epicID}, &fetched)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, 1, fetched.ChangeCount)

	status = client.Post(t, "/api/v1/epic/delete", map[string]any{"id": epicID}, nil)
	assert.Equal(t, http.StatusConflict, status)

	status = client.Post(t, "/api/v1/change/delete", map[string]any{"id": createdChange.ID}, nil)
	require.Equal(t, http.StatusNoContent, status)

	status = client.Post(t, "/api/v1/epic/delete", map[string]any{"id": epicID}, nil)
	assert.Equal(t, http.StatusNoContent, status)
}

func TestEpicRejectsInvalidInputAndMissingRows(t *testing.T) {
	client := shared.NewClient(t)

	status := client.Post(t, "/api/v1/epic/list", map[string]any{}, nil)
	assert.Equal(t, http.StatusBadRequest, status)

	status = client.Post(t, "/api/v1/epic/create", map[string]any{
		"project_id": 0,
		"name":       "orphan epic",
	}, nil)
	assert.Equal(t, http.StatusBadRequest, status)

	status = client.Post(t, "/api/v1/epic/create", map[string]any{
		"project_id": 999999999,
		"name":       "orphan epic",
	}, nil)
	assert.Equal(t, http.StatusNotFound, status)

	projectID := createProject(t, client)
	defer shared.CleanupProject(t, client, projectID)

	status = client.Post(t, "/api/v1/epic/create", map[string]any{
		"project_id": projectID,
		"name":       "   ",
	}, nil)
	assert.Equal(t, http.StatusBadRequest, status)

	status = client.Post(t, "/api/v1/epic/get", map[string]any{"id": 999999999}, nil)
	assert.Equal(t, http.StatusNotFound, status)

	status = client.Post(t, "/api/v1/epic/update", map[string]any{
		"id":   999999999,
		"name": "missing epic",
	}, nil)
	assert.Equal(t, http.StatusNotFound, status)

	status = client.Post(t, "/api/v1/epic/update", map[string]any{
		"id":   1,
		"name": "   ",
	}, nil)
	assert.Equal(t, http.StatusBadRequest, status)

	status = client.Post(t, "/api/v1/epic/delete", map[string]any{"id": 999999999}, nil)
	assert.Equal(t, http.StatusNotFound, status)

	status = client.Post(t, "/api/v1/epic/delete", map[string]any{}, nil)
	assert.Equal(t, http.StatusBadRequest, status)
}

func createProject(t *testing.T, client *shared.Client) int {
	t.Helper()

	var created project
	status := client.Post(t, "/api/v1/project/create", map[string]string{
		"name": fmt.Sprintf("api-test-epic-project-%d", time.Now().UnixNano()),
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
		"name":       fmt.Sprintf("api-test-epic-%d", time.Now().UnixNano()),
	}, &created)
	require.Equal(t, http.StatusCreated, status)
	require.NotEmpty(t, created.ID)
	assert.Equal(t, projectID, created.ProjectID)
	return created.ID
}
