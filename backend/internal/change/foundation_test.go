package change

import (
	"context"
	"errors"
	"mch_api/internal/domain"
	apperror "mch_api/internal/error"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

type foundationPool struct {
	changePool
	t   *testing.T
	row pgx.Row
	tag pgconn.CommandTag
	err error
}

func (p foundationPool) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	require.Contains(p.t, sql, "from public.vw_change_details where id = $1")
	require.NotContains(p.t, sql, "version")
	require.NotContains(p.t, sql, "brief")
	require.Contains(p.t, sql, "coalesce(100 * done_tc / nullif(total_tc, 0), 0)")
	require.Equal(p.t, []any{7}, args)
	return p.row
}

func (p foundationPool) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	require.Equal(p.t, "update public.change set change_types = $1, modified = now() where id = $2", sql)
	require.Equal(p.t, []any{[]string{"fix"}, 7}, args)
	return p.tag, p.err
}

func TestDetailsCurrentViewAndMissingRecords(t *testing.T) {
	now := time.Now()
	row := schemaRow{7, "uuid", (*int32)(nil), (*string)(nil), 1, "backlog", []string{"fix"}, (*int)(nil), (*string)(nil), "title", true, int16(1), int16(2), int16(50), now, "https://pr", now}
	failure := errors.New("scan failure")
	for _, tc := range []struct {
		row pgx.Row
		err error
	}{
		{row, nil}, {errorRow{pgx.ErrNoRows}, apperror.ErrChangeNotFound}, {errorRow{failure}, failure},
	} {
		repo := &Repo{pool: foundationPool{t: t, row: tc.row}}
		got, err := repo.Details(context.Background(), 7)
		require.ErrorIs(t, err, tc.err)
		if tc.err == nil {
			require.Equal(t, domain.ChangeDetails{ChangeListItem: domain.ChangeListItem{
				ID: 7, RefUUID: "uuid", ProjectID: 1, ChangePhase: "backlog", ChangeTypes: []string{"fix"},
				Title: "title", Open: true, DoneTC: 1, TotalTC: 2, Completed: 50, Modified: now,
			}, PRUrl: "https://pr", Created: now}, got)
		}
	}
}

func TestTypeMutationErrorOnly(t *testing.T) {
	failure := errors.New("exec failure")
	for _, tc := range []struct {
		tag         string
		dbErr, want error
	}{
		{"UPDATE 1", nil, nil}, {"UPDATE 0", nil, apperror.ErrChangeNotFound}, {"", failure, failure},
	} {
		repo := &Repo{pool: foundationPool{t: t, tag: pgconn.NewCommandTag(tc.tag), err: tc.dbErr}}
		require.ErrorIs(t, repo.UpdateChangeTypes(context.Background(), domain.ChangeUpdateChangeTypesRequest{ID: 7, ChangeTypes: []string{"fix"}}), tc.want)
	}
	service := NewService(&fakeChangeRepository{}, Renderer{})
	require.ErrorIs(t, service.UpdateChangeTypes(context.Background(), domain.ChangeUpdateChangeTypesRequest{}), apperror.ErrChangeInvalidInput)
}

type listFoundationPool struct {
	changePool
	rows pgx.Rows
}

func (p listFoundationPool) Query(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
	return p.rows, nil
}

type listFoundationRows struct {
	pgx.Rows
	row  schemaRow
	done bool
}

func (r *listFoundationRows) Next() bool             { return !r.done }
func (r *listFoundationRows) Scan(dest ...any) error { r.done = true; return r.row.Scan(dest...) }
func (r *listFoundationRows) Close()                 {}
func (r *listFoundationRows) Err() error             { return nil }

func TestListScansTotalPointer(t *testing.T) {
	require.False(t, strings.Contains(changeListColumns, "\n\tcompleted,"))
	rows := &listFoundationRows{row: schemaRow{7, "uuid", (*int32)(nil), (*string)(nil), 1, "backlog", []string{}, (*int)(nil), (*string)(nil), "title", true, int16(1), int16(2), int16(50), time.Now()}}
	repo := &Repo{pool: listFoundationPool{rows: rows}}
	got, err := repo.List(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, int16(2), got[0].TotalTC)
}

func TestChangeMissingRowKeepsCause(t *testing.T) {
	cause := apperror.Wrap(pgx.ErrNoRows, "read")
	repo := &Repo{pool: foundationPool{t: t, row: errorRow{cause}}}
	_, err := repo.Details(context.Background(), 7)
	require.ErrorIs(t, err, apperror.ErrChangeNotFound)
	require.ErrorIs(t, err, cause)
	require.ErrorIs(t, apperror.HTTP(err), pgx.ErrNoRows)
}

func TestChangeHelperMissingCauses(t *testing.T) {
	cause := apperror.Wrap(pgx.ErrNoRows, "nested scan")
	tx := stateTx{row: errorRow{cause}}
	_, err := getChange(context.Background(), tx, 7)
	require.ErrorIs(t, err, apperror.ErrChangeNotFound)
	require.ErrorIs(t, err, cause)
	_, err = getState(context.Background(), tx, 7)
	require.ErrorIs(t, err, apperror.ErrChangeNotFound)
	require.ErrorIs(t, err, cause)
}
