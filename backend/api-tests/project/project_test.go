package project_test

import (
	"context"
	"fmt"
	"mch_api/api-tests/shared"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type project struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	LastRef     int32     `json:"last_ref"`
	Created     time.Time `json:"created"`
	Modified    time.Time `json:"modified"`
	ChangeCount int       `json:"change_count"`
}

func TestProjectCRUD(t *testing.T) {
	client := shared.NewClient(t)
	name := fmt.Sprintf("api-test-project-%d", time.Now().UnixNano())
	updatedName := name + "-updated"

	var created project
	status := client.Post(t, "/api/v1/project/create", map[string]string{"name": name}, &created)
	require.Equal(t, http.StatusCreated, status)
	require.NotEmpty(t, created.ID)
	require.Equal(t, http.StatusOK, client.Post(t, "/api/v1/project/get", map[string]any{"id": created.ID}, &created))
	assert.Equal(t, name, created.Name)
	assert.Equal(t, int32(0), created.LastRef)
	assert.False(t, created.Created.IsZero())
	assert.False(t, created.Modified.IsZero())
	assert.Equal(t, 0, created.ChangeCount)

	defer shared.CleanupProject(t, client, created.ID)

	var listed []project
	status = client.Post(t, "/api/v1/project/list", map[string]any{"last_ref": 999}, &listed)
	require.Equal(t, http.StatusOK, status)
	assert.Contains(t, listed, created)

	var fetched project
	status = client.Post(t, "/api/v1/project/get", map[string]any{"id": created.ID}, &fetched)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, created, fetched)

	var updated project
	status = client.Post(t, "/api/v1/project/update", map[string]any{"id": created.ID, "name": updatedName}, nil)
	require.Equal(t, http.StatusNoContent, status)
	require.Equal(t, http.StatusOK, client.Post(t, "/api/v1/project/get", map[string]any{"id": created.ID}, &updated))
	assert.Equal(t, updatedName, updated.Name)
	assert.True(t, updated.Modified.After(created.Modified))
	// A same-name update must still advance modified.
	require.Equal(t, http.StatusNoContent, client.Post(t, "/api/v1/project/update", map[string]any{"id": created.ID, "name": updatedName}, nil))
	var same project
	require.Equal(t, http.StatusOK, client.Post(t, "/api/v1/project/get", map[string]any{"id": created.ID}, &same))
	require.True(t, same.Modified.After(updated.Modified))

	status = client.Post(t, "/api/v1/project/delete", map[string]any{"id": created.ID}, nil)
	require.Equal(t, http.StatusNoContent, status)

	status = client.Post(t, "/api/v1/project/get", map[string]any{"id": created.ID}, nil)
	assert.Equal(t, http.StatusNotFound, status)
}

func TestProjectRejectsInvalidInputAndMissingRows(t *testing.T) {
	client := shared.NewClient(t)

	status := client.Post(t, "/api/v1/project/create", map[string]any{"name": "   "}, nil)
	assert.Equal(t, http.StatusBadRequest, status)

	status = client.Post(t, "/api/v1/project/get", map[string]any{"id": 999999999}, nil)
	assert.Equal(t, http.StatusNotFound, status)

	status = client.Post(t, "/api/v1/project/update", map[string]any{
		"id":   999999999,
		"name": "missing project",
	}, nil)
	assert.Equal(t, http.StatusNotFound, status)

	status = client.Post(t, "/api/v1/project/update", map[string]any{
		"id":   999999999,
		"name": "   ",
	}, nil)
	assert.Equal(t, http.StatusBadRequest, status)

	status = client.Post(t, "/api/v1/project/delete", map[string]any{"id": 999999999}, nil)
	assert.Equal(t, http.StatusNotFound, status)

	status = client.Post(t, "/api/v1/project/delete", map[string]any{}, nil)
	assert.Equal(t, http.StatusBadRequest, status)
}

// This SQL assertion supplements APIHydra: project/epic document read APIs do
// not exist. The harness supplies only its owned disposable database URL.
func TestDeletionRetainsAppendOnlyDocuments(t *testing.T) {
	client := shared.NewClient(t)
	databaseURL := os.Getenv("API_TEST_DB_URL")
	require.NotEmpty(t, databaseURL, "API_TEST_DB_URL must identify the disposable API-test database")
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, databaseURL)
	require.NoError(t, err)
	defer func() { require.NoError(t, conn.Close(ctx)) }()
	var parent, child struct {
		ID int `json:"id"`
	}
	require.Equal(t, http.StatusCreated, client.Post(t, "/api/v1/project/create", map[string]any{"name": "Document history parent"}, &parent))
	require.Equal(t, http.StatusCreated, client.Post(t, "/api/v1/epic/create", map[string]any{"project_id": parent.ID, "name": "Document history epic"}, &child))
	for table, id := range map[string]int{"project": parent.ID, "epic": child.ID} {
		_, err = conn.Exec(ctx, `insert into public.doc(ref_id,ref_table,doc_type,body,current) values($1,$2,'brief','historic',false),($1,$2,'brief','current',true)`, id, table)
		require.NoError(t, err)
	}
	require.Equal(t, http.StatusNoContent, client.Post(t, "/api/v1/epic/delete", map[string]any{"id": child.ID}, nil))
	require.Equal(t, http.StatusNoContent, client.Post(t, "/api/v1/project/delete", map[string]any{"id": parent.ID}, nil))
	for table, id := range map[string]int{"project": parent.ID, "epic": child.ID} {
		var bodies []string
		require.NoError(t, conn.QueryRow(ctx, `select array_agg(body order by id) from public.doc where ref_id=$1 and ref_table=$2`, id, table).Scan(&bodies))
		require.Equal(t, []string{"historic", "current"}, bodies)
	}
}
