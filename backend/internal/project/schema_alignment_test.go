package project

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

func TestDeleteDeactivatesOnlyOnDependencyConflict(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	failure := errors.New("database unavailable")
	for _, tc := range []struct {
		deleteErr, deactivateErr error
		wantStatus               int
		wantCalls                []string
	}{
		{nil, nil, 204, []string{"delete"}},
		{app.WrapError(app.ErrProjectHasChanges, "fk"), nil, 204, []string{"delete", "deactivate"}},
		{app.ErrProjectHasChanges, failure, 500, []string{"delete", "deactivate"}},
		{app.ErrProjectHasChanges, app.ErrProjectNotFound, 404, []string{"delete", "deactivate"}},
		{failure, nil, 500, []string{"delete"}},
		{app.ErrProjectNotFound, nil, 404, []string{"delete"}},
	} {
		r := &fakeProjectRepository{err: tc.deleteErr, deactivateErr: tc.deactivateErr}
		e := echo.New()
		NewAPI(e, NewService(r))
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/v1/project/delete", strings.NewReader(`{"id":7}`)).WithContext(ctx)
		req.Header.Set("Content-Type", "application/json")
		e.ServeHTTP(rec, req)
		require.Equal(t, tc.wantStatus, rec.Code, rec.Body.String())
		require.Equal(t, tc.wantCalls, r.calls)
		require.Same(t, ctx, r.ctx)
		require.Equal(t, domain.ProjectIDRequest{ID: 7}, r.req)
		if tc.wantStatus == 204 {
			require.Empty(t, rec.Body.String())
		}
	}
}

func TestRepositoryDeactivationPreservesFailuresAndMissingRows(t *testing.T) {
	failure := errors.New("update failed")
	for _, tc := range []struct {
		err      error
		affected string
		want     error
	}{
		{nil, "1", nil}, {nil, "0", app.ErrProjectNotFound}, {failure, "1", failure},
	} {
		p := newBoundary(t)
		p.args = []any{7}
		p.err, p.tag = tc.err, "UPDATE "+tc.affected
		err := (&Repo{pool: p}).Deactivate(p.ctx, domain.ProjectIDRequest{ID: 7})
		require.ErrorIs(t, err, tc.want)
		require.Equal(t, "update public.project set active = false, updated_at = now() where id = $1", p.sql)
	}
}
