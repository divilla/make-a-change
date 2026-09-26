package testcase

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
)

func TestTestCaseMoveRouteRemoved(t *testing.T) {
	e := echo.New()
	NewAPI(e, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/test-case/update-change", strings.NewReader(`{"id":1,"change_id":2}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}
