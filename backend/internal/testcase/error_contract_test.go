package testcase

import (
	"errors"
	"fmt"
	"mch_api/internal/app"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/require"
)

func TestTestCaseHandlerErrorContracts(t *testing.T) {
	for _, op := range []struct {
		name, body, payload string
		handler             func(*API, *echo.Context) error
		success             int
	}{
		{"list", `{"change_id":1}`, "list", (*API).list, 200},
		{"create", `{"change_id":1,"scenario":"Test"}`, "create", (*API).create, 201},
		{"update", `{"id":1,"scenario":"Test"}`, "update", (*API).updateTestCase, 204},
		{"update-done", `{"id":1,"done":true}`, "done", (*API).updateTestCaseDone, 204},
		{"delete", `{"id":1}`, "delete", (*API).delete, 204},
	} {
		t.Run(op.name, func(t *testing.T) {
			unknown := errors.New("private database detail")
			for _, tc := range []struct {
				name    string
				cause   error
				code    int
				message string
			}{
				{"success", nil, op.success, ""},
				{"missing", app.ErrTestCaseNotFound, 404, "test case not found"},
				{"wrapped missing", fmt.Errorf("outer: %w", app.ErrTestCaseNotFound), 404, "test case not found"},
				{"invalid", app.ErrTestCaseInvalidInput, 400, "invalid test case payload"},
				{"unknown", unknown, 500, "Internal Server Error"},
			} {
				t.Run(tc.name, func(t *testing.T) {
					e := echo.New()
					a := NewAPI(e, NewService(&fakeTestCaseRepository{err: tc.cause}))
					rec := httptest.NewRecorder()
					req := httptest.NewRequest("POST", "/", strings.NewReader(op.body))
					req.Header.Set("Content-Type", "application/json")
					ctx := e.NewContext(req, rec)
					err := op.handler(a, ctx)
					if tc.cause == nil {
						require.NoError(t, err)
						require.Equal(t, op.success, rec.Code)
						return
					}
					require.ErrorIs(t, err, tc.cause)
					var he *echo.HTTPError
					require.ErrorAs(t, err, &he)
					require.Equal(t, tc.code, he.Code)
					require.Equal(t, tc.message, he.Message)
					e.HTTPErrorHandler(ctx, err)
					require.Equal(t, tc.code, rec.Code)
					require.Equal(t, `{"message":"`+tc.message+`"}`+"\n", rec.Body.String())
				})
			}
			for _, body := range []string{"{", `{"id":"wrong","change_id":"wrong"}`} {
				e := echo.New()
				a := NewAPI(e, nil)
				rec := httptest.NewRecorder()
				req := httptest.NewRequest("POST", "/", strings.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				ctx := e.NewContext(req, rec)
				err := op.handler(a, ctx)
				var he *echo.HTTPError
				require.ErrorAs(t, err, &he)
				require.NotNil(t, errors.Unwrap(he))
				require.Equal(t, 400, he.Code)
				require.Equal(t, "invalid test case "+op.payload+" payload", he.Message)
			}
		})
	}
}
