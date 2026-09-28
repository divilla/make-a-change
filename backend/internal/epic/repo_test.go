package epic

import (
	"context"
	"errors"
	apperror "mch_api/internal/error"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type errorPool struct {
	epicPool
	row pgx.Row
}

func (p errorPool) QueryRow(context.Context, string, ...any) pgx.Row { return p.row }

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

func TestEpicRepositoryMissingCauses(t *testing.T) {
	failure := errors.New("database failure")
	for _, cause := range []error{pgx.ErrNoRows, apperror.Wrap(pgx.ErrNoRows, "query"), failure} {
		pool := errorPool{row: errorRow{err: cause}}
		repo := &Repo{pool: pool}
		_, err := repo.Get(context.Background(), 1)
		require.ErrorIs(t, err, cause)
		_, nestedErr := getEpic(context.Background(), pool, 1)
		require.ErrorIs(t, nestedErr, cause)
		if errors.Is(cause, pgx.ErrNoRows) {
			require.ErrorIs(t, err, apperror.ErrEpicNotFound)
			require.ErrorIs(t, nestedErr, apperror.ErrEpicNotFound)
		}
	}
	for _, tc := range []struct {
		row  errorRow
		want error
	}{
		{errorRow{}, apperror.ErrEpicNotFound}, {errorRow{exists: true}, nil}, {errorRow{err: failure}, failure},
	} {
		repo := &Repo{pool: errorPool{row: tc.row}}
		require.ErrorIs(t, repo.ensureProject(context.Background(), 1), tc.want)
	}
}
