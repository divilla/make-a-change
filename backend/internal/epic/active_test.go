package epic

import (
	"context"
	"errors"
	"mch_api/internal/app"
	"mch_api/internal/domain"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/require"
)

func TestListActiveBinding(t *testing.T) {
	for _, tc := range []struct {
		body   string
		active *bool
		valid  bool
	}{
		{`{"project_id":7}`, nil, true},
		{`{"project_id":7,"active":null}`, nil, true},
		{`{"project_id":7,"active":true}`, activePointer(true), true},
		{`{"project_id":7,"active":false}`, activePointer(false), true},
		{`{"project_id":7,"active":"false"}`, nil, false},
		{`{"project_id":7,"active":0}`, nil, false},
	} {
		t.Run(tc.body, func(t *testing.T) {
			repo := &fakeEpicRepository{}
			e := echo.New()
			NewAPI(e, NewService(repo))
			req := httptest.NewRequest("POST", "/api/v1/epic/list", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			if !tc.valid {
				require.Equal(t, 400, rec.Code)
				require.Empty(t, repo.calls)
				return
			}
			require.Equal(t, 200, rec.Code)
			require.Equal(t, domain.EpicListRequest{ProjectID: 7, Active: tc.active}, repo.req)
		})
	}
}

func activePointer(value bool) *bool { return &value }

func TestUpdateActiveValidationAndForwarding(t *testing.T) {
	for _, value := range []bool{false, true} {
		for _, cause := range []error{nil, errors.New("database failed")} {
			repo := &fakeEpicRepository{err: cause}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			req := domain.EpicUpdateActiveRequest{ID: 7, Active: &value}
			err := NewService(repo).UpdateActive(ctx, req)
			require.ErrorIs(t, err, cause)
			require.Equal(t, req, repo.req)
			require.Same(t, ctx, repo.ctx)
		}
	}
	for _, req := range []domain.EpicUpdateActiveRequest{
		{ID: 7}, {ID: 0, Active: activePointer(false)}, {ID: -1, Active: activePointer(true)},
	} {
		require.ErrorIs(t, NewService(nil).UpdateActive(context.Background(), req), app.ErrEpicInvalidInput)
	}
	for _, tc := range []struct{ body, message string }{
		{`{"id":7}`, "invalid epic payload"},
		{`{"id":7,"active":null}`, "invalid epic payload"},
		{`{"id":7,"active":"false"}`, "invalid epic active payload"},
		{`{"id":7,"active":0}`, "invalid epic active payload"},
		{`{"id":0,"active":false}`, "invalid epic payload"},
	} {
		e := echo.New()
		repo := &fakeEpicRepository{}
		NewAPI(e, NewService(repo))
		req := httptest.NewRequest("POST", "/api/v1/epic/update-active", strings.NewReader(tc.body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		require.Equal(t, 400, rec.Code)
		require.JSONEq(t, `{"message":"`+tc.message+`"}`, rec.Body.String())
		require.Empty(t, repo.calls)
	}
}

func TestRepositoryUpdateActive(t *testing.T) {
	failure := errors.New("database unavailable")
	for _, active := range []bool{true, false} {
		for _, tc := range []struct {
			tag         string
			cause, want error
		}{
			{"UPDATE 1", nil, nil}, {"UPDATE 0", nil, app.ErrEpicNotFound}, {"", failure, failure},
		} {
			p := newBoundary(t)
			p.args = []any{7, &active}
			p.tag, p.err = tc.tag, tc.cause
			err := (&Repo{pool: p}).UpdateActive(p.ctx, domain.EpicUpdateActiveRequest{ID: 7, Active: &active})
			require.ErrorIs(t, err, tc.want)
			require.Equal(t, "update public.epic set active = $2, updated_at = now() where id = $1", p.sql)
		}
	}
}
