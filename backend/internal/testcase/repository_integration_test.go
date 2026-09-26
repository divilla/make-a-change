package testcase

import (
	"context"
	"os"
	"testing"

	"aipm/internal/change"
	"aipm/internal/dto"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRepositoryHistoryProcedures requires a disposable database initialized with db/init.sql.
func TestRepositoryHistoryProcedures(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if testing.Short() || databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL to a disposable database initialized with db/init.sql")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	var projectID, epicID, changeID int
	require.NoError(t, pool.QueryRow(ctx, "insert into public.project(name) values ('history regression') returning id").Scan(&projectID))
	t.Cleanup(func() {
		for _, statement := range []string{
			"delete from public.test_case_history where change_id in (select id from public.change where project_id = $1)",
			"delete from public.test_case where change_id in (select id from public.change where project_id = $1)",
			"delete from public.change where project_id = $1",
			"delete from public.epic where project_id = $1",
			"delete from public.project where id = $1",
		} {
			_, err := pool.Exec(ctx, statement, projectID)
			assert.NoError(t, err)
		}
	})
	require.NoError(t, pool.QueryRow(ctx, "insert into public.epic(project_id, name) values ($1, 'history regression') returning id", projectID).Scan(&epicID))
	require.NoError(t, pool.QueryRow(ctx, "insert into public.change(project_id, epic_id) values ($1, $2) returning id", projectID, epicID).Scan(&changeID))
	t.Cleanup(func() {
		_, err := pool.Exec(ctx, "delete from public.test_case_history where change_id = $1", changeID)
		assert.NoError(t, err)
	})

	repo := NewRepo(pool)
	first, err := repo.Create(ctx, dto.TestCaseCreateRequest{ChangeID: changeID, Scenario: "Initial scenario"})
	require.NoError(t, err)
	require.NotNil(t, first.TestCase)
	assert.Equal(t, int16(0), first.TestCase.Version)
	assert.False(t, first.TestCase.Done)
	assert.Equal(t, int16(1), first.Change.TotalTC)
	second, err := repo.Create(ctx, dto.TestCaseCreateRequest{ChangeID: changeID, Scenario: "Unedited scenario"})
	require.NoError(t, err)
	require.NotNil(t, second.TestCase)
	assert.Equal(t, int16(2), second.Change.TotalTC)
	firstID, secondID := first.TestCase.ID, second.TestCase.ID
	var initialHistoryCount int
	require.NoError(t, pool.QueryRow(ctx, "select count(*) from public.test_case_history where change_id = $1 and version = 0 and not deleted", changeID).Scan(&initialHistoryCount))
	assert.Equal(t, 2, initialHistoryCount)
	updated, err := repo.Update(ctx, dto.TestCaseUpdateRequest{ID: firstID, Scenario: "Updated scenario"})
	require.NoError(t, err)
	require.NotNil(t, updated.TestCase)
	assert.Equal(t, int16(1), updated.TestCase.Version)
	assert.Equal(t, "Updated scenario", updated.TestCase.Scenario)
	unchanged, err := repo.Update(ctx, dto.TestCaseUpdateRequest{ID: firstID, Scenario: "Updated scenario"})
	require.NoError(t, err)
	require.NotNil(t, unchanged.TestCase)
	assert.Equal(t, int16(1), unchanged.TestCase.Version)

	_, err = repo.UpdateDone(ctx, dto.TestCaseUpdateDoneRequest{ID: firstID, Done: true})
	require.NoError(t, err)
	require.NoError(t, change.NewRepo(pool).Delete(ctx, dto.ChangeIDRequest{ID: changeID}))
	_, err = repo.Create(ctx, dto.TestCaseCreateRequest{ChangeID: changeID, Scenario: "Missing parent"})
	require.ErrorIs(t, err, ErrNotFound)

	type historyEntry struct {
		ID       int
		Version  int16
		Scenario string
		Deleted  bool
	}
	rows, err := pool.Query(ctx, "select id, version, scenario, deleted from public.test_case_history where change_id = $1 order by id, version", changeID)
	require.NoError(t, err)
	history, err := pgx.CollectRows(rows, pgx.RowToStructByPos[historyEntry])
	require.NoError(t, err)
	assert.Equal(t, []historyEntry{
		{firstID, 0, "Initial scenario", false},
		{firstID, 1, "Updated scenario", false},
		{firstID, 2, "Updated scenario", true},
		{secondID, 0, "Unedited scenario", false},
		{secondID, 1, "Unedited scenario", true},
	}, history)
	var count, done, total int
	require.NoError(t, pool.QueryRow(ctx, "select count(*) from public.test_case where change_id = $1", changeID).Scan(&count))
	assert.Zero(t, count)
	require.NoError(t, pool.QueryRow(ctx, "select count(*) from public.change where id = $1", changeID).Scan(&count))
	assert.Zero(t, count)
	require.NoError(t, pool.QueryRow(ctx, "select done_tc, total_tc from public.epic where id = $1", epicID).Scan(&done, &total))
	assert.Zero(t, done)
	assert.Zero(t, total)
}
