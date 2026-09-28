package project

import (
	"context"
	"errors"
	apperror "mch_api/internal/error"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

type errorPool struct {
	projectPool
	row pgx.Row
	tag string
	err error
}

func (p errorPool) QueryRow(context.Context, string, ...any) pgx.Row { return p.row }
func (p errorPool) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag(p.tag), p.err
}

type errorRow struct {
	err    error
	exists bool
}

func (r errorRow) Scan(dest ...any) error {
	if r.err == nil {
		*dest[0].(*bool) = r.exists
	}
	return r.err
}

func TestProjectRepositoryMissingAndConflictContracts(t *testing.T) {
	failure := errors.New("database failure")
	for _, cause := range []error{pgx.ErrNoRows, apperror.Wrap(pgx.ErrNoRows, "query"), failure} {
		repo := &Repo{pool: errorPool{row: errorRow{err: cause}}}
		_, err := repo.Get(context.Background(), 1)
		require.ErrorIs(t, err, cause)
		if errors.Is(cause, pgx.ErrNoRows) {
			require.ErrorIs(t, err, apperror.ErrProjectNotFound)
		}
	}
	for _, tc := range []struct {
		name, tag     string
		row           errorRow
		execErr, want error
	}{
		{name: "deleted", tag: "DELETE 1"},
		{name: "missing", tag: "DELETE 0", want: apperror.ErrProjectNotFound},
		{name: "conflict", row: errorRow{exists: true}, want: apperror.ErrProjectHasChanges},
		{name: "exec error", execErr: failure, want: failure},
		{name: "existence scan", row: errorRow{err: failure}, want: failure},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &Repo{pool: errorPool{row: tc.row, tag: tc.tag, err: tc.execErr}}
			require.ErrorIs(t, repo.Delete(context.Background(), 1), tc.want)
		})
	}
	repo := &Repo{pool: errorPool{tag: "UPDATE 0"}}
	_, err := repo.Update(context.Background(), 1, "Name")
	require.ErrorIs(t, err, apperror.ErrProjectNotFound)
}
