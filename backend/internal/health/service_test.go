package health

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServiceCheckReturnsOKWhenDatabasePings(t *testing.T) {
	service := NewService(fakeHealthRepository{})

	res := service.Check(context.Background())

	assert.Equal(t, "ok", res.Status)
	assert.Equal(t, "ok", res.API)
	assert.Equal(t, "ok", res.Database)
	assert.Empty(t, res.Error)
}

func TestServiceCheckReturnsDegradedWhenDatabasePingFails(t *testing.T) {
	service := NewService(fakeHealthRepository{err: errors.New("database unavailable")})

	res := service.Check(context.Background())

	assert.Equal(t, "degraded", res.Status)
	assert.Equal(t, "ok", res.API)
	assert.Equal(t, "error", res.Database)
	assert.Equal(t, "database unavailable", res.Error)
}

type fakeHealthRepository struct {
	err error
}

func (r fakeHealthRepository) Ping(context.Context) error {
	return r.err
}
