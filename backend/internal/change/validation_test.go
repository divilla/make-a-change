package change

import (
	"errors"
	apperror "mch_api/internal/error"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gookit/validate/v2"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/require"
)

func TestAPIValidationCauses(t *testing.T) {
	for _, tc := range []struct{ path, body, field, rule, bindMessage string }{
		{"list", `{"project_id":-1,"scenario":"valid"}`, "project_id", "min", "list"},
		{"get", `{"id":-1,"scenario":"valid"}`, "id", "min", "get"},
		{"create", `{"project_id":-1,"scenario":"valid"}`, "project_id", "min", "create"},
		{"update-epic", `{"id":-1,"scenario":"valid"}`, "id", "min", "epic"},
		{"update-change-types", `{"id":-1,"scenario":"valid"}`, "id", "min", "types"},
		{"update-title", `{"id":-1,"scenario":"valid"}`, "id", "min", "title"},
		{"update-brief", `{"id":-1,"scenario":"valid"}`, "id", "min", "brief"},
		{"update-spec", `{"id":-1,"scenario":"valid"}`, "id", "min", "spec"},
		{"update-pr", `{"id":-1,"scenario":"valid"}`, "id", "min", "pr"},
		{"update-pr-url", `{"id":-1,"scenario":"valid"}`, "id", "min", "pr url"},
		{"update-phase", `{"id":-1,"scenario":"valid"}`, "id", "min", "phase"},
		{"update-open", `{"id":-1,"scenario":"valid"}`, "id", "min", "open"},
		{"delete", `{"id":-1,"scenario":"valid"}`, "id", "min", "delete"},
		{"documents", `{"id":-1,"scenario":"valid"}`, "id", "min", "documents"},
		{"set-document", `{"id":-1,"scenario":"valid"}`, "id", "min", "document"},
		{"set-document", `{"id":1,"body":"valid"}`, "doc_type", "required", "document"},
		{"set-document", `{"id":1,"doc_type":"spec"}`, "body", "required", "document"},
	} {
		t.Run(tc.path+tc.field, func(t *testing.T) {
			for _, binding := range []bool{false, true} {
				repo := &fakeChangeRepository{}
				config := defaultConfig()
				e := echo.New()
				NewAPI(e, NewService(repo, Renderer{}, config))
				var returned error
				e.HTTPErrorHandler = func(c *echo.Context, err error) {
					returned = err
					echo.DefaultHTTPErrorHandler(false)(c, err)
				}
				body := tc.body
				if binding {
					body = "{"
				}
				req := httptest.NewRequest("POST", "/api/v1/change/"+tc.path, strings.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				rec := httptest.NewRecorder()
				e.ServeHTTP(rec, req)
				require.Error(t, returned)
				require.Equal(t, 400, rec.Code)
				require.Empty(t, repo.calls)
				require.Empty(t, config.ids)
				if binding {
					require.NotNil(t, errors.Unwrap(returned))
					require.JSONEq(t, `{"message":"invalid change `+tc.bindMessage+` payload"}`, rec.Body.String())
				} else {
					require.ErrorIs(t, returned, apperror.ErrChangeInvalidInput)
					var validation validate.Errors
					require.ErrorAs(t, returned, &validation)
					require.Contains(t, validation, tc.field)
					require.NotEmpty(t, validation[tc.field][tc.rule])
					require.JSONEq(t, `{"message":"invalid change payload"}`, rec.Body.String())
				}
			}
		})
	}
}
