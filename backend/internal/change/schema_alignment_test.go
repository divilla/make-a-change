package change

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

func TestInactiveChangeServiceValidatesAndDerivesCompletion(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, err := (&Service{}).ListInactive(ctx, domain.ChangeListRequest{})
	require.ErrorIs(t, err, app.ErrChangeInvalidInput)
	failure := errors.New("read failed")
	for _, cause := range []error{nil, failure} {
		r := &fakeChangeRepository{err: cause, list: []domain.ChangeListItem{{ID: 7, DoneTC: 1, TotalTC: 2}, {ID: 8, TotalTC: 0}}}
		got, err := NewService(r, nil).ListInactive(ctx, domain.ChangeListRequest{ProjectID: 9})
		require.ErrorIs(t, err, cause)
		require.Equal(t, []string{"ListInactive"}, r.calls)
		require.Equal(t, []any{domain.ChangeListRequest{ProjectID: 9}}, r.requests)
		require.Same(t, ctx, r.contexts[0])
		if cause == nil {
			require.Equal(t, int64(50), got[0].Completed)
			require.Zero(t, got[1].Completed)
		}
	}
}

func TestInactiveChangeAPIValidationAndErrors(t *testing.T) {
	for _, tc := range []struct {
		body   string
		cause  error
		status int
	}{
		{`{"project_id":9}`, nil, 200},
		{`{`, nil, 400},
		{`{"project_id":"bad"}`, nil, 400},
		{`{}`, nil, 400},
		{`{"project_id":-1}`, nil, 400},
		{`{"project_id":9}`, errors.New("private failure"), 500},
	} {
		r := &fakeChangeRepository{err: tc.cause}
		e := echo.New()
		NewAPI(e, NewService(r, nil))
		req := httptest.NewRequest("POST", "/api/v1/change/list-inactive", strings.NewReader(tc.body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		require.Equal(t, tc.status, rec.Code, rec.Body.String())
		if tc.status == 200 {
			require.JSONEq(t, `[]`, rec.Body.String())
		}
		if tc.status == 400 {
			require.Empty(t, r.calls)
		}
	}
}

func TestUpdateOpenRouteIsAbsent(t *testing.T) {
	e := echo.New()
	NewAPI(e, nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest("POST", "/api/v1/change/update-open", strings.NewReader(`{}`)))
	require.Equal(t, 404, rec.Code)
}
