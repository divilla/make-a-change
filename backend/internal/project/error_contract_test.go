package project

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

func TestProjectHandlerErrorContracts(t *testing.T) {
	for _, op := range []struct {
		name, body, payload string
		handler             func(*API, *echo.Context) error
		success             int
	}{
		{"config", `{"id":1}`, "config", (*API).config, 200},
		{"list", `{}`, "", (*API).list, 200},
		{"details", `{"id":1}`, "details", (*API).details, 200},
		{"create", `{"name":"Name"}`, "create", (*API).create, 201},
		{"update", `{"id":1,"name":"Name"}`, "update", (*API).updateProject, 204},
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
				{"config missing", app.ErrProjectConfigNotFound, 404, "project configuration not found"},
				{"missing", app.ErrProjectNotFound, 404, "project not found"},
				{"wrapped missing", fmt.Errorf("outer: %w", app.ErrProjectNotFound), 404, "project not found"},
				{"invalid", app.ErrProjectInvalidInput, 400, "invalid project payload"},
				{"conflict", app.ErrProjectHasChanges, 409, "project has dependencies and cannot be deleted"},
				{"unknown", unknown, 500, "Internal Server Error"},
			} {
				t.Run(tc.name, func(t *testing.T) {
					e := echo.New()
					a := NewAPI(e, NewService(&fakeProjectRepository{err: tc.cause}))
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
				require.Equal(t, "invalid project "+op.payload+" payload", he.Message)
			}
		})
	}
}
