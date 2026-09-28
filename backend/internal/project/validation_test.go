package project

import (
	"errors"
	"mch_api/internal/app"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gookit/validate/v2"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/require"
)

func TestAPIValidationCauses(t *testing.T) {
	for _, tc := range []struct{ path, body, field, rule, bindMessage string }{
		{"details", `{"id":-1,"scenario":"valid"}`, "id", "min", "details"},
		{"create", `{"name":"","scenario":"valid"}`, "name", "required", "create"},
		{"update", `{"id":-1,"scenario":"valid"}`, "id", "min", "update"},
		{"delete", `{"id":-1,"scenario":"valid"}`, "id", "min", "delete"},
		{"config", `{"id":-1,"scenario":"valid"}`, "id", "min", "config"},
		{"update", `{"id":1}`, "name", "required", "update"},
	} {
		t.Run(tc.path+tc.field, func(t *testing.T) {
			for _, binding := range []bool{false, true} {
				repo := &fakeProjectRepository{}

				e := echo.New()
				NewAPI(e, NewService(repo))
				var returned error
				e.HTTPErrorHandler = func(c *echo.Context, err error) {
					returned = err
					echo.DefaultHTTPErrorHandler(false)(c, err)
				}
				body := tc.body
				if binding {
					body = "{"
				}
				req := httptest.NewRequest("POST", "/api/v1/project/"+tc.path, strings.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				rec := httptest.NewRecorder()
				e.ServeHTTP(rec, req)
				require.Error(t, returned)
				require.Equal(t, 400, rec.Code)
				require.Empty(t, repo.calls)

				if binding {
					require.NotNil(t, errors.Unwrap(returned))
					require.JSONEq(t, `{"message":"invalid project `+tc.bindMessage+` payload"}`, rec.Body.String())
				} else {
					require.ErrorIs(t, returned, app.ErrProjectInvalidInput)
					var validation validate.Errors
					require.ErrorAs(t, returned, &validation)
					require.Contains(t, validation, tc.field)
					require.NotEmpty(t, validation[tc.field][tc.rule])
					require.JSONEq(t, `{"message":"invalid project payload"}`, rec.Body.String())
				}
			}
		})
	}
}
