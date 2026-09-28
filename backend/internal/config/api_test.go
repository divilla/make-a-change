package config

import (
	"encoding/json"
	"errors"
	"mch_api/internal/app"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/require"
)

func TestAPIContracts(t *testing.T) {
	for _, op := range []string{"list", "details", "insert", "update", "delete"} {
		raw, _ := json.Marshal(validConfig())
		body := string(raw)
		for _, scenario := range []string{"success", "malformed", "invalid", "failure", "missing", "conflict"} {
			t.Run(op+scenario, func(t *testing.T) {
				r := &fakeRepo{}
				input := body
				status := 200
				if op == "insert" {
					status = 201
				}
				if op == "update" || op == "delete" {
					status = 204
				}
				switch scenario {
				case "malformed":
					input = "{"
					status = 400
				case "invalid":
					input = `{}`
					status = 400
				case "failure":
					r.err = errors.New("private failure")
					status = 500
				case "missing":
					r.err = app.ErrConfigNotFound
					status = 404
				case "conflict":
					r.err = app.ErrConfigInUse
					status = 409
				}
				if op == "list" && (scenario == "malformed" || scenario == "invalid") {
					status = 200
				}
				e := echo.New()
				NewAPI(e, NewService(r))
				req := httptest.NewRequest("POST", "/api/v1/config/"+op, strings.NewReader(input))
				req.Header.Set("Content-Type", "application/json")
				rec := httptest.NewRecorder()
				e.ServeHTTP(rec, req)
				require.Equal(t, status, rec.Code, rec.Body.String())
				if status == 204 {
					require.Empty(t, rec.Body.String())
				}
				if scenario == "success" && op == "insert" {
					require.JSONEq(t, `{"slug":"custom"}`, rec.Body.String())
				}
				if scenario == "failure" {
					require.NotContains(t, rec.Body.String(), "private")
				}
			})
		}
	}
}
