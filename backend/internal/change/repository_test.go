package change

import (
	"context"
	"strings"
	"testing"
	"time"

	"aipm/internal/dto"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListTestCasesQueriesByID(t *testing.T) {
	now := time.Now()
	rows := &testCaseRows{
		items: []dto.TestCase{
			{ID: 11, Version: 1, Scenario: "zzz first by id", Done: false, ChangeID: 7, Created: now.Add(time.Minute), Modified: now},
			{ID: 12, Version: 1, Scenario: "aaa second by id", Done: true, ChangeID: 7, Created: now, Modified: now},
		},
	}
	q := &capturingQueryer{rows: rows}

	got, err := listTestCases(context.Background(), q, 7)
	require.NoError(t, err)

	require.Len(t, got, 2)
	assert.Equal(t, []int{11, 12}, []int{got[0].ID, got[1].ID})
	assert.Equal(t, []any{7}, q.args)
	assert.Contains(t, strings.ToLower(q.sql), "order by id")
	assert.NotContains(t, strings.ToLower(q.sql), "order by created")
}

type capturingQueryer struct {
	sql  string
	args []any
	rows pgx.Rows
}

func (q *capturingQueryer) Query(_ context.Context, sql string, args ...any) (pgx.Rows, error) {
	q.sql = sql
	q.args = args
	return q.rows, nil
}

func (q *capturingQueryer) QueryRow(context.Context, string, ...any) pgx.Row {
	panic("QueryRow is not used by this test")
}

type testCaseRows struct {
	items  []dto.TestCase
	index  int
	closed bool
}

func (r *testCaseRows) Close() {
	r.closed = true
}

func (r *testCaseRows) Err() error {
	return nil
}

func (r *testCaseRows) CommandTag() pgconn.CommandTag {
	return pgconn.CommandTag{}
}

func (r *testCaseRows) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}

func (r *testCaseRows) Next() bool {
	if r.index >= len(r.items) {
		r.Close()
		return false
	}
	r.index++
	return true
}

func (r *testCaseRows) Scan(dest ...any) error {
	item := r.items[r.index-1]
	values := []any{
		item.ID,
		item.Version,
		item.Scenario,
		item.Done,
		item.ChangeID,
		item.Created,
		item.Modified,
	}
	for i := range dest {
		switch target := dest[i].(type) {
		case *int:
			*target = values[i].(int)
		case *int16:
			*target = values[i].(int16)
		case *string:
			*target = values[i].(string)
		case *bool:
			*target = values[i].(bool)
		case *time.Time:
			*target = values[i].(time.Time)
		}
	}
	return nil
}

func (r *testCaseRows) Values() ([]any, error) {
	return nil, nil
}

func (r *testCaseRows) RawValues() [][]byte {
	return nil
}

func (r *testCaseRows) Conn() *pgx.Conn {
	return nil
}
