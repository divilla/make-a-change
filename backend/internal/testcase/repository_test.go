package testcase

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateTestCase(t *testing.T) {
	now := time.Now()
	tx := &createTx{rows: []pgx.Row{
		schemaRow{17},
		schemaRow{17, int16(0), "Initial scenario", false, 42, now, now},
	}}
	got, err := createTestCase(context.Background(), tx, 42, "Initial scenario")
	require.NoError(t, err)
	require.Len(t, tx.queries, 2)
	assert.Equal(t, "select public.fn_test_case_insert($1, $2)", tx.queries[0])
	assert.Equal(t, [][]any{{42, "Initial scenario"}, {17}}, tx.args)
	assert.Equal(t, 17, got.ID)
	assert.Equal(t, 42, got.ChangeID)
	assert.Equal(t, "Initial scenario", got.Scenario)
	assert.Zero(t, got.Version)
	assert.False(t, got.Done)
}

func TestCreateTestCaseErrors(t *testing.T) {
	failure := errors.New("database error")
	missingParent := &pgconn.PgError{Code: "23503"}
	otherPostgresError := &pgconn.PgError{Code: "23505"}
	for _, tc := range []struct {
		name string
		rows []pgx.Row
		want error
	}{
		{"function", []pgx.Row{errorRow{failure}}, failure},
		{"missing parent", []pgx.Row{errorRow{missingParent}}, ErrNotFound},
		{"wrapped missing parent", []pgx.Row{errorRow{fmt.Errorf("insert: %w", missingParent)}}, ErrNotFound},
		{"other postgres error", []pgx.Row{errorRow{otherPostgresError}}, otherPostgresError},
		{"missing testcase", []pgx.Row{schemaRow{17}, errorRow{pgx.ErrNoRows}}, ErrNotFound},
		{"read", []pgx.Row{schemaRow{17}, errorRow{failure}}, failure},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx := &createTx{rows: tc.rows}
			_, err := createTestCase(context.Background(), tx, 42, "Initial scenario")
			require.ErrorIs(t, err, tc.want)
			assert.Len(t, tx.queries, len(tc.rows))
		})
	}
}

type createTx struct {
	pgx.Tx
	rows    []pgx.Row
	queries []string
	args    [][]any
}

func (tx *createTx) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	tx.queries = append(tx.queries, sql)
	tx.args = append(tx.args, args)
	return tx.rows[len(tx.queries)-1]
}

func TestUpdateTestCaseDone(t *testing.T) {
	for _, done := range []bool{false, true} {
		t.Run(map[bool]string{false: "incomplete", true: "complete"}[done], func(t *testing.T) {
			now := time.Now()
			tx := &doneTx{row: schemaRow{17, int16(3), "Scenario", done, 42, now, now}}
			got, err := updateTestCaseDone(context.Background(), tx, 17, done)
			require.NoError(t, err)
			assert.Equal(t, "call public.sp_test_case_update_done($1, $2)", tx.sql)
			assert.Equal(t, []any{17, done}, tx.args)
			assert.Equal(t, []any{17}, tx.queryArgs)
			assert.Equal(t, 17, got.ID)
			assert.Equal(t, 42, got.ChangeID)
			assert.Equal(t, done, got.Done)
			assert.Equal(t, int16(3), got.Version)
		})
	}
}

func TestUpdateTestCaseDoneErrors(t *testing.T) {
	failure := errors.New("database error")
	for _, tc := range []struct {
		name string
		tx   *doneTx
		want error
	}{
		{"procedure", &doneTx{execErr: failure}, failure},
		{"missing testcase", &doneTx{row: errorRow{pgx.ErrNoRows}}, ErrNotFound},
		{"read", &doneTx{row: errorRow{failure}}, failure},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := updateTestCaseDone(context.Background(), tc.tx, 17, true)
			require.ErrorIs(t, err, tc.want)
			if tc.tx.execErr != nil {
				assert.Nil(t, tc.tx.queryArgs)
			}
		})
	}
}

func TestUpdateTestCaseScenario(t *testing.T) {
	now := time.Now()
	tx := &doneTx{row: schemaRow{17, int16(4), "Updated scenario", true, 42, now, now}}
	got, err := updateTestCaseScenario(context.Background(), tx, 17, "Updated scenario")
	require.NoError(t, err)
	assert.Equal(t, "call public.sp_test_case_update_scenario($1, $2)", tx.sql)
	assert.Equal(t, []any{17, "Updated scenario"}, tx.args)
	assert.Equal(t, []any{17}, tx.queryArgs)
	assert.Equal(t, "Updated scenario", got.Scenario)
	assert.Equal(t, int16(4), got.Version)
	assert.True(t, got.Done)
}

func TestUpdateTestCaseScenarioErrors(t *testing.T) {
	failure := errors.New("database error")
	for _, tc := range []struct {
		name string
		tx   *doneTx
		want error
	}{
		{"procedure", &doneTx{execErr: failure}, failure},
		{"missing testcase", &doneTx{row: errorRow{pgx.ErrNoRows}}, ErrNotFound},
		{"read", &doneTx{row: errorRow{failure}}, failure},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := updateTestCaseScenario(context.Background(), tc.tx, 17, "Updated scenario")
			require.ErrorIs(t, err, tc.want)
			if tc.tx.execErr != nil {
				assert.Nil(t, tc.tx.queryArgs)
			}
		})
	}
}

type doneTx struct {
	pgx.Tx
	sql       string
	args      []any
	queryArgs []any
	execErr   error
	row       pgx.Row
}

func (tx *doneTx) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	tx.sql, tx.args = sql, args
	return pgconn.CommandTag{}, tx.execErr
}

func (tx *doneTx) QueryRow(_ context.Context, _ string, args ...any) pgx.Row {
	if tx.sql == "" {
		panic("testcase must be read after the procedure call")
	}
	tx.queryArgs = args
	return tx.row
}
