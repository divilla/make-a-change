package db

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

func TestPoolCreationAndPanicCause(t *testing.T) {
	pool := Pool(context.Background(), "postgres://localhost/test")
	require.NotNil(t, pool)
	pool.Close()
	var recovered any
	func() { defer func() { recovered = recover() }(); Pool(context.Background(), ":invalid") }()
	err, ok := recovered.(error)
	require.True(t, ok)
	require.Contains(t, err.Error(), "connect database:")
	require.NotNil(t, errors.Unwrap(err))
	var parseErr *pgconn.ParseConfigError
	require.ErrorAs(t, err, &parseErr)
}
