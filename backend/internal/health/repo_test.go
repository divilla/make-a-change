package health

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestPingPreservesCancellation(t *testing.T) {
	pool, err := pgxpool.New(context.Background(), "postgres://localhost/test")
	require.NoError(t, err)
	defer pool.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.ErrorIs(t, NewRepo(pool).Ping(ctx), context.Canceled)
	got := NewService(NewRepo(pool)).Check(ctx)
	require.Equal(t, "degraded", got.Status)
	require.Equal(t, "database unavailable", got.Error)
}
