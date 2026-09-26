package change

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// schemaRow enforces the number and types of columns in a database result.
type schemaRow []any

func (r schemaRow) Scan(dest ...any) error {
	if len(dest) != len(r) {
		return fmt.Errorf("got %d scan destinations for %d columns", len(dest), len(r))
	}
	for i, value := range r {
		reflect.ValueOf(dest[i]).Elem().Set(reflect.ValueOf(value))
	}
	return nil
}

type errorRow struct{ err error }

func (r errorRow) Scan(...any) error { return r.err }

func TestScanChangeCurrentSchema(t *testing.T) {
	now := time.Now()
	identity := pgtype.UUID{Bytes: [16]byte{1}, Valid: true}
	for _, nullable := range []bool{true, false} {
		t.Run(fmt.Sprintf("nullable=%t", nullable), func(t *testing.T) {
			row := schemaRow{
				7, identity, pgtype.Int4{Int32: 12, Valid: !nullable}, int16(2),
				pgtype.Text{String: "slug", Valid: !nullable}, 1, "backlog", []string{"fix"},
				pgtype.Int8{Int64: 4, Valid: !nullable}, pgtype.Text{String: "Epic", Valid: !nullable},
				"Title", "Brief", "Spec", "PR", "https://example.test/pr", true,
				int16(1), int16(2), int16(50), now, now,
			}
			got, err := scanChange(row)
			require.NoError(t, err)
			assert.Equal(t, 7, got.ID)
			assert.Equal(t, identity.String(), got.RefUUID)
			assert.Equal(t, "Brief", got.Brief)
			assert.Equal(t, "Spec", got.Spec)
			assert.Equal(t, "PR", got.PR)
			assert.True(t, got.Open)
			assert.Equal(t, int16(50), got.Completed)
			assert.Equal(t, now, got.Modified)
			if nullable {
				assert.Nil(t, got.Ref)
				assert.Nil(t, got.Slug)
				assert.Nil(t, got.EpicID)
				assert.Nil(t, got.EpicName)
			} else {
				require.NotNil(t, got.Ref)
				require.NotNil(t, got.Slug)
				require.NotNil(t, got.EpicID)
				require.NotNil(t, got.EpicName)
				assert.Equal(t, int32(12), *got.Ref)
				assert.Equal(t, "slug", *got.Slug)
				assert.Equal(t, 4, *got.EpicID)
				assert.Equal(t, "Epic", *got.EpicName)
			}
		})
	}
	failure := errors.New("scan failed")
	_, err := scanChange(errorRow{failure})
	assert.ErrorIs(t, err, failure)
}

type stateTx struct {
	pgx.Tx
	row pgx.Row
}

func (tx stateTx) QueryRow(context.Context, string, ...any) pgx.Row { return tx.row }

func TestGetStateCurrentSchema(t *testing.T) {
	for _, nullable := range []bool{true, false} {
		got, err := getState(context.Background(), stateTx{row: schemaRow{
			1, pgtype.Int8{Int64: 4, Valid: !nullable}, "review", []string{"fix"},
			"Title", "Brief", "Spec", "PR", "https://example.test/pr", true,
		}}, 7)
		require.NoError(t, err)
		assert.Equal(t, 1, got.ProjectID)
		assert.Equal(t, "Brief", got.Brief)
		assert.Equal(t, "review", got.ChangePhase)
		assert.True(t, got.Open)
		if nullable {
			assert.Nil(t, got.EpicID)
		} else {
			require.NotNil(t, got.EpicID)
			assert.Equal(t, 4, *got.EpicID)
		}
	}
	_, err := getState(context.Background(), stateTx{row: errorRow{pgx.ErrNoRows}}, 7)
	assert.ErrorIs(t, err, ErrNotFound)
	failure := errors.New("database unavailable")
	_, err = getState(context.Background(), stateTx{row: errorRow{failure}}, 7)
	assert.ErrorIs(t, err, failure)
}
