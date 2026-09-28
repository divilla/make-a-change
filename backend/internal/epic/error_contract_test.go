package epic

import (
	"errors"
	"fmt"
	apperror "mch_api/internal/error"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/require"
)

func TestEpicHandlerErrorContracts(t *testing.T) {
	for _, op := range []struct {
		name, body, payload string
		handler             func(*API, *echo.Context) error
		success             int
	}{
		{"list", `{"project_id":1}`, "list", (*API).listEpics, 200},
		{"get", `{"id":1}`, "get", (*API).getEpic, 200},
		{"create", `{"project_id":1,"name":"Name"}`, "create", (*API).createEpic, 201},
		{"update", `{"id":1,"name":"Name"}`, "update", (*API).updateEpic, 200},
		{"delete", `{"id":1}`, "delete", (*API).deleteEpic, 204},
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
				{"missing", apperror.ErrEpicNotFound, 404, "epic not found"},
				{"wrapped missing", fmt.Errorf("outer: %w", apperror.ErrEpicNotFound), 404, "epic not found"},
				{"invalid", apperror.ErrEpicInvalidInput, 400, "invalid epic payload"},
				{"conflict", apperror.ErrEpicHasChanges, 409, "epic has changes and cannot be deleted"},
				{"unknown", unknown, 500, "Internal Server Error"},
			} {
				t.Run(tc.name, func(t *testing.T) {
					e := echo.New()
					a := NewAPI(e, NewService(&fakeEpicRepository{err: tc.cause}))
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
			if op.payload != "" {
				e := echo.New()
				a := NewAPI(e, nil)
				rec := httptest.NewRecorder()
				req := httptest.NewRequest("POST", "/", strings.NewReader("{"))
				req.Header.Set("Content-Type", "application/json")
				ctx := e.NewContext(req, rec)
				err := op.handler(a, ctx)
				var he *echo.HTTPError
				require.ErrorAs(t, err, &he)
				require.NotNil(t, errors.Unwrap(he))
				require.Equal(t, 400, he.Code)
				require.Equal(t, "invalid epic "+op.payload+" payload", he.Message)
			}
		})
	}
}
