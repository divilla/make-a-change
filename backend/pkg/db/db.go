package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool initializes or executes Pool behavior.
func Pool(ctx context.Context, url string) *pgxpool.Pool {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		panic(err)
	}

	return pool
}
