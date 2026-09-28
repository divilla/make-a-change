package health

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/require"
)

func TestHealthAliasExactContracts(t *testing.T) {
	for _, path := range []string{"/api/v1/health", "/api/health"} {
		for _, degraded := range []bool{false, true} {
			var failure error
			status, body := 200, `{"status":"ok","api":"ok","database":"ok"}`
			if degraded {
				failure = errors.New("private DB error")
				status, body = 503, `{"status":"degraded","api":"ok","database":"error","error":"database unavailable"}`
			}
			e := echo.New()
			NewAPI(e, NewService(fakeHealthRepository{err: failure}))
			response := httptest.NewRecorder()
			e.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
			require.Equal(t, status, response.Code)
			require.JSONEq(t, body, response.Body.String())
		}
	}
}
