package health

import (
	"cli/internal/dto"
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeAPI struct {
	value dto.Health
	err   error
	calls []string
}

func (f *fakeAPI) CheckHealth(_ context.Context, route string) (dto.Health, error) {
	f.calls = append(f.calls, route)
	return f.value, f.err
}

func TestP704HealthRouteSelectionAndDegradedDisplay(t *testing.T) {
	api := &fakeAPI{value: dto.Health{Route: "/api/health", HTTPStatus: 503, Status: "degraded", API: "ok", Database: "error", Error: "database unavailable"}}
	m := Model{}.SelectRoute("/api/health")
	m, cmd := m.Begin(context.Background(), api)
	m, ok := m.Apply(cmd().(Result))
	require.True(t, ok)
	assert.Contains(t, View(m), "HTTP 503")
	assert.Contains(t, View(m), "database unavailable")
	assert.Equal(t, []string{"/api/health"}, api.calls)
	m = m.SelectRoute("/api/v1/health")
	assert.Nil(t, m.Last)
	assert.Contains(t, View(m), "/api/v1/health")
	assert.NotContains(t, View(m), "HTTP 503")
}

func TestP704HealthFailureRetryAndStaleRouteResults(t *testing.T) {
	api := &fakeAPI{value: dto.Health{Route: "/api/v1/health", HTTPStatus: 200, Status: "ok", API: "ok", Database: "ok"}}
	m := Model{}.SelectRoute("/api/v1/health")
	m, cmd := m.Begin(context.Background(), api)
	m, _ = m.Apply(cmd().(Result))
	api.err = errors.New("network down")
	m, cmd = m.Begin(context.Background(), api)
	m, _ = m.Apply(cmd().(Result))
	assert.Contains(t, View(m), "HTTP 200")
	assert.Contains(t, View(m), "network down")
	m, cmd = m.Begin(context.Background(), api)
	old := cmd().(Result)
	m = m.SelectRoute("/api/health")
	_, accepted := m.Apply(old)
	assert.False(t, accepted)
	assert.Nil(t, m.Last)
	m, pending := m.Begin(context.Background(), api)
	m = m.Invalidate()
	assert.False(t, m.Busy)
	_, accepted = m.Apply(pending().(Result))
	assert.False(t, accepted)
}

func TestP704HealthTerminalSafePresentation(t *testing.T) {
	m := Model{Route: "/api/health", Last: &dto.Health{HTTPStatus: 503, Status: "degraded", API: "ok", Database: "error", Error: "\x1b[2Jbad"}}
	assert.NotContains(t, View(m), "\x1b[2J")
}
