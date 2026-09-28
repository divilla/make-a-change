package change_test

import (
	"context"
	"mch_api/api-tests/shared"
	"mch_api/internal/domain"
	"net/http"
	"os"
	"testing"

	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

// APIHydra owns the route campaign. These additional HTTP/SQL checks prove
// timestamp ordering, generated UUID versions and retained history that its
// subset comparator cannot prove. They never contribute to APIHydra coverage.
func TestChangeIdentityOrderingAndSameValueTimestamps(t *testing.T) {
	client := shared.NewClient(t)
	var project domain.ProjectIDRequest
	require.Equal(t, http.StatusCreated, client.Post(t, "/api/v1/project/create", map[string]any{"name": "P3 independent checks"}, &project))
	defer shared.CleanupProject(t, client, project.ID)
	var ids []int
	for _, title := range []string{"First", "Second"} {
		var id domain.ChangeIDRequest
		require.Equal(t, 201, client.Post(t, "/api/v1/change/create", map[string]any{"project_id": project.ID, "title": title, "brief": "Brief"}, &id))
		ids = append(ids, id.ID)
		var got domain.ChangeDetails
		require.Equal(t, 200, client.Post(t, "/api/v1/change/get", id, &got))
		u, err := uuid.FromString(got.RefUUID)
		require.NoError(t, err)
		require.Equal(t, byte(7), u.Version())
	}
	var list []domain.ChangeListItem
	require.Equal(t, 200, client.Post(t, "/api/v1/change/list", domain.ChangeListRequest{ProjectID: project.ID}, &list))
	require.Equal(t, []int{ids[1], ids[0]}, []int{list[0].ID, list[1].ID})
	var before domain.ChangeDetails
	id := domain.ChangeIDRequest{ID: ids[0]}
	require.Equal(t, 200, client.Post(t, "/api/v1/change/get", id, &before))
	for _, update := range []struct {
		path string
		body map[string]any
	}{
		{"update-title", map[string]any{"title": "First"}},
		{"update-open", map[string]any{"open": true}},
		{"update-change-types", map[string]any{"change_types": []string{}}},
		{"update-pr-url", map[string]any{"pr_url": "https://example.test"}},
		{"update-pr-url", map[string]any{"pr_url": "https://example.test"}},
	} {
		update.body["id"] = id.ID
		require.Equal(t, 204, client.Post(t, "/api/v1/change/"+update.path, update.body, nil))
		var after domain.ChangeDetails
		require.Equal(t, 200, client.Post(t, "/api/v1/change/get", id, &after))
		require.True(t, after.Modified.After(before.Modified), update.path)
		before = after
	}
	require.Equal(t, 200, client.Post(t, "/api/v1/change/list", domain.ChangeListRequest{ProjectID: project.ID}, &list))
	require.Equal(t, []int{ids[0], ids[1]}, []int{list[0].ID, list[1].ID})
}

func TestChangeDeletionRetainsAppendOnlyDocuments(t *testing.T) {
	client := shared.NewClient(t)
	databaseURL := os.Getenv("API_TEST_DB_URL")
	require.NotEmpty(t, databaseURL, "only the harness-owned disposable database is permitted")
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, databaseURL)
	require.NoError(t, err)
	defer func() { require.NoError(t, conn.Close(ctx)) }()
	var project domain.ProjectIDRequest
	require.Equal(t, 201, client.Post(t, "/api/v1/project/create", map[string]any{"name": "P3 history contract"}, &project))
	defer shared.CleanupProject(t, client, project.ID)
	var id domain.ChangeIDRequest
	require.Equal(t, 201, client.Post(t, "/api/v1/change/create", map[string]any{"project_id": project.ID, "title": "History", "brief": "Identical body"}, &id))
	// Repeated same-body writes still append, retaining both previous rows.
	for range 2 {
		require.Equal(t, 204, client.Post(t, "/api/v1/change/update-brief", map[string]any{"id": id.ID, "brief": "Identical body", "agent_edit": true}, nil))
		var docs []domain.ChangeDocument
		require.Equal(t, 200, client.Post(t, "/api/v1/change/documents", id, &docs))
		require.Len(t, docs, 1)
		require.Equal(t, "Identical body", docs[0].Body)
		require.True(t, docs[0].AgentEdit)
	}
	var bodies []string
	var current []bool
	var ids []int64
	require.NoError(t, conn.QueryRow(ctx, `select array_agg(id order by id), array_agg(body order by id), array_agg(current order by id) from public.doc where ref_table='change' and ref_id=$1`, id.ID).Scan(&ids, &bodies, &current))
	require.Len(t, ids, 3)
	require.Less(t, ids[0], ids[1])
	require.Less(t, ids[1], ids[2])
	require.Equal(t, []string{"Identical body", "Identical body", "Identical body"}, bodies)
	require.Equal(t, []bool{false, false, true}, current)
	// An actual testcase FK blocks one DELETE and leaves all state intact.
	var testcaseID domain.TestCaseIDRequest
	require.Equal(t, 201, client.Post(t, "/api/v1/test-case/create", domain.TestCaseCreateRequest{ChangeID: id.ID, Scenario: "Block deletion"}, &testcaseID))
	require.Equal(t, 409, client.Post(t, "/api/v1/change/delete", id, nil))
	var surviving domain.ChangeDetails
	require.Equal(t, 200, client.Post(t, "/api/v1/change/get", id, &surviving))
	require.Equal(t, id.ID, surviving.ID)
	var survivingDocs []domain.ChangeDocument
	require.Equal(t, 200, client.Post(t, "/api/v1/change/documents", id, &survivingDocs))
	require.Len(t, survivingDocs, 1)
	require.Equal(t, ids[2], int64(survivingDocs[0].ID))
	require.Equal(t, "Identical body", survivingDocs[0].Body)

	var count int
	require.NoError(t, conn.QueryRow(ctx, `select count(*) from public.testcase where id=$1`, testcaseID.ID).Scan(&count))
	require.Equal(t, 1, count)
	require.Equal(t, 204, client.Post(t, "/api/v1/test-case/delete", testcaseID, nil))
	require.Equal(t, 204, client.Post(t, "/api/v1/change/delete", id, nil))
	require.Equal(t, 404, client.Post(t, "/api/v1/change/get", id, nil))
	require.Equal(t, 404, client.Post(t, "/api/v1/change/documents", id, nil))
	require.Equal(t, 404, client.Post(t, "/api/v1/change/set-document", map[string]any{"id": id.ID, "doc_type": "brief", "body": "No resurrection", "agent_edit": false}, nil))
	require.Equal(t, 404, client.Post(t, "/api/v1/test-case/list", domain.TestCaseListRequest{ChangeID: id.ID}, nil))
	require.Equal(t, 404, client.Post(t, "/api/v1/test-case/create", domain.TestCaseCreateRequest{ChangeID: id.ID, Scenario: "Removed"}, nil))
	require.NoError(t, conn.QueryRow(ctx, `select count(*) from public.change where id=$1`, id.ID).Scan(&count))
	require.Zero(t, count)
	require.NoError(t, conn.QueryRow(ctx, `select count(*) from public.testcase where change_id=$1`, id.ID).Scan(&count))
	require.Zero(t, count)

	var retained []int64
	var retainedBodies []string
	var retainedCurrent []bool
	require.NoError(t, conn.QueryRow(ctx, `select array_agg(id order by id),array_agg(body order by id),array_agg(current order by id) from public.doc where ref_table='change' and ref_id=$1`, id.ID).Scan(&retained, &retainedBodies, &retainedCurrent))
	require.Equal(t, ids, retained)
	require.Equal(t, bodies, retainedBodies)
	require.Equal(t, current, retainedCurrent)
}
