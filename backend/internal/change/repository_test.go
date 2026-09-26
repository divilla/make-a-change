package change

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteTestCasesForChange(t *testing.T) {
	failure := errors.New("database error")
	for _, tc := range []struct {
		name     string
		rows     *testCaseRows
		queryErr error
		execErr  error
		wantIDs  []int
		wantErr  error
	}{
		{name: "multiple cases", rows: &testCaseRows{ids: []int{17, 18}}, wantIDs: []int{17, 18}},
		{name: "no cases", rows: &testCaseRows{}},
		{name: "query failure", queryErr: failure, wantErr: failure},
		{name: "scan failure", rows: &testCaseRows{ids: []int{17}, scanErr: failure}, wantErr: failure},
		{name: "iteration failure", rows: &testCaseRows{ids: []int{17}, err: failure}, wantErr: failure},
		{name: "procedure failure", rows: &testCaseRows{ids: []int{17, 18}}, execErr: failure, wantIDs: []int{17}, wantErr: failure},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx := &deleteCasesTx{t: t, rows: tc.rows, queryErr: tc.queryErr, execErr: tc.execErr}
			err := deleteTestCasesForChange(context.Background(), tx, 42)
			require.ErrorIs(t, err, tc.wantErr)
			assert.Equal(t, []any{42}, tx.queryArgs)
			assert.Equal(t, tc.wantIDs, tx.deletedIDs)
			if tc.rows != nil {
				assert.True(t, tc.rows.closed)
			}
		})
	}
}

type deleteCasesTx struct {
	pgx.Tx
	t          *testing.T
	rows       *testCaseRows
	queryErr   error
	execErr    error
	queryArgs  []any
	deletedIDs []int
}

func (tx *deleteCasesTx) Query(_ context.Context, _ string, args ...any) (pgx.Rows, error) {
	tx.queryArgs = args
	return tx.rows, tx.queryErr
}

func (tx *deleteCasesTx) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	require.True(tx.t, tx.rows.closed, "close query results before executing a procedure on the same connection")
	require.Equal(tx.t, "call public.sp_test_case_delete($1)", sql)
	require.Len(tx.t, args, 1)
	tx.deletedIDs = append(tx.deletedIDs, args[0].(int))
	return pgconn.CommandTag{}, tx.execErr
}

type testCaseRows struct {
	pgx.Rows
	ids     []int
	index   int
	closed  bool
	scanErr error
	err     error
}

func (r *testCaseRows) Next() bool { return r.index < len(r.ids) }
func (r *testCaseRows) Close()     { r.closed = true }
func (r *testCaseRows) Err() error { return r.err }
func (r *testCaseRows) Scan(dest ...any) error {
	if r.scanErr != nil {
		return r.scanErr
	}
	*dest[0].(*int) = r.ids[r.index]
	r.index++
	return nil
}
