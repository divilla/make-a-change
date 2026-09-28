package doc

import (
	"errors"
	"mch_api/internal/app"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/require"
)

func TestAPIContracts(t *testing.T) {
	for _, op := range []string{"list", "current", "details", "insert"} {
		body := `{"id":8,"ref_id":7,"ref_table":"change","doc_type":"spec","body":"raw","agent_edit":false}`
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
					r.err = app.ErrDocNotFound
					status = 404
				case "conflict":
					r.err = app.ErrDocInvalidReference
					status = 400
				}

				e := echo.New()
				NewAPI(e, NewService(r, Renderer{}, &fakeConfig{}))
				req := httptest.NewRequest("POST", "/api/v1/doc/"+op, strings.NewReader(input))
				req.Header.Set("Content-Type", "application/json")
				rec := httptest.NewRecorder()
				e.ServeHTTP(rec, req)
				require.Equal(t, status, rec.Code, rec.Body.String())
				if status == 204 {
					require.Empty(t, rec.Body.String())
				}
				if scenario == "success" && op == "insert" {
					require.JSONEq(t, `{"id":8}`, rec.Body.String())
				}
				if scenario == "failure" {
					require.NotContains(t, rec.Body.String(), "private")
				}
			})
		}
	}
}
