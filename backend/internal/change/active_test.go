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

func TestListActiveBinding(t *testing.T) {
	for _, tc := range []struct {
		body   string
		active *bool
		valid  bool
	}{
		{`{"project_id":7}`, nil, true},
		{`{"project_id":7,"active":null}`, nil, true},
		{`{"project_id":7,"active":true}`, boolPtr(true), true},
		{`{"project_id":7,"active":false}`, boolPtr(false), true},
		{`{"project_id":7,"active":"false"}`, nil, false},
		{`{"project_id":7,"active":0}`, nil, false},
	} {
		t.Run(tc.body, func(t *testing.T) {
			repo := &fakeChangeRepository{}
			e := echo.New()
			NewAPI(e, NewService(repo, nil))
			req := httptest.NewRequest("POST", "/api/v1/change/list", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			if !tc.valid {
				require.Equal(t, 400, rec.Code)
				require.Empty(t, repo.calls)
				return
			}
			require.Equal(t, 200, rec.Code)
			require.Equal(t, domain.ChangeListRequest{ProjectID: 7, Active: tc.active}, repo.requests[0])
		})
	}
}

func TestUpdateActiveValidationAndForwarding(t *testing.T) {
	for _, value := range []bool{false, true} {
		for _, cause := range []error{nil, errors.New("database failed")} {
			repo := &fakeChangeRepository{err: cause}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			req := domain.ChangeUpdateActiveRequest{ID: 7, Active: &value}
			err := NewService(repo, nil).UpdateActive(ctx, req)
			require.ErrorIs(t, err, cause)
			require.Equal(t, req, repo.requests[0])
			require.Same(t, ctx, repo.contexts[0])
		}
	}
	for _, req := range []domain.ChangeUpdateActiveRequest{
		{ID: 7}, {ID: 0, Active: boolPtr(false)}, {ID: -1, Active: boolPtr(true)},
	} {
		require.ErrorIs(t, NewService(nil, nil).UpdateActive(context.Background(), req), app.ErrChangeInvalidInput)
	}
	for _, tc := range []struct{ body, message string }{
		{`{"id":7}`, "invalid change payload"},
		{`{"id":7,"active":null}`, "invalid change payload"},
		{`{"id":7,"active":"false"}`, "invalid change active payload"},
		{`{"id":7,"active":0}`, "invalid change active payload"},
		{`{"id":0,"active":false}`, "invalid change payload"},
	} {
		e := echo.New()
		repo := &fakeChangeRepository{}
		NewAPI(e, NewService(repo, nil))
		req := httptest.NewRequest("POST", "/api/v1/change/update-active", strings.NewReader(tc.body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		require.Equal(t, 400, rec.Code)
		require.JSONEq(t, `{"message":"`+tc.message+`"}`, rec.Body.String())
		require.Empty(t, repo.calls)
	}
}
