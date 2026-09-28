package testcase

import (
	"mch_api/internal/domain"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestTestCaseAPIExactContracts(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
		want       string
		request    any
	}{
		{"list", `{"change_id":1099511627776}`, 200, "[]\n", domain.TestCaseListRequest{ChangeID: 1 << 40}},
		{"create", `{"change_id":1099511627776,"scenario":"  trimmed  "}`, 201, "{\"id\":3}\n", domain.TestCaseCreateRequest{ChangeID: 1 << 40, Scenario: "trimmed"}},
		{"update", `{"id":1099511627776,"scenario":"  trimmed  "}`, 204, "", domain.TestCaseUpdateRequest{ID: 1 << 40, Scenario: "trimmed"}},
		{"update-done", `{"id":1099511627776,"done":true}`, 204, "", domain.TestCaseUpdateDoneRequest{ID: 1 << 40, Done: true}},
		{"update-done", `{"id":1,"done":false}`, 204, "", domain.TestCaseUpdateDoneRequest{ID: 1}},
		{"update-done", `{"id":1}`, 204, "", domain.TestCaseUpdateDoneRequest{ID: 1}},
		{"update-done", `{"id":1,"done":null}`, 204, "", domain.TestCaseUpdateDoneRequest{ID: 1}},
		{"delete", `{"id":1099511627776}`, 204, "", domain.TestCaseIDRequest{ID: 1 << 40}},
	} {
		t.Run(tc.name+tc.body, func(t *testing.T) {
			r := &fakeTestCaseRepository{}
			e := echo.New()
			NewAPI(e, NewService(r))
			req := httptest.NewRequest("POST", "/api/v1/test-case/"+tc.name, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			require.Equal(t, tc.status, rec.Code)
			require.Equal(t, tc.want, rec.Body.String())
			require.Equal(t, []any{tc.request}, r.calls)
		})
	}
}

func TestTestCaseAPIInvalidPayloads(t *testing.T) {
	for _, op := range []struct{ name, key, payload string }{{"list", "change_id", "list"}, {"create", "change_id", "create"}, {"update", "id", "update"}, {"update-done", "id", "done"}, {"delete", "id", "delete"}} {
		bodies := []struct {
			body        string
			bindFailure bool
		}{
			{"{", true},
			{`{"` + op.key + `":"wrong"}`, true},
			{`{"` + op.key + `":0,"scenario":"valid"}`, false},
			{`{"` + op.key + `":-1,"scenario":"valid"}`, false},
			{`{}`, false},
		}
		if op.name == "create" || op.name == "update" {
			bodies = append(bodies, struct {
				body        string
				bindFailure bool
			}{`{"` + op.key + `":1,"scenario":false}`, true})
			for _, scenario := range []string{`" \t "`, `null`} {
				bodies = append(bodies, struct {
					body        string
					bindFailure bool
				}{`{"` + op.key + `":1,"scenario":` + scenario + `}`, false})
			}
		}
		if op.name == "update-done" {
			for _, done := range []string{`"true"`, `1`, `[]`} {
				bodies = append(bodies, struct {
					body        string
					bindFailure bool
				}{`{"id":1,"done":` + done + `}`, true})
			}
		}
		for _, tc := range bodies {
			t.Run(op.name+tc.body, func(t *testing.T) {
				r := &fakeTestCaseRepository{}
				e := echo.New()
				NewAPI(e, NewService(r))
				req := httptest.NewRequest("POST", "/api/v1/test-case/"+op.name, strings.NewReader(tc.body))
				req.Header.Set("Content-Type", "application/json")
				rec := httptest.NewRecorder()
				e.ServeHTTP(rec, req)
				require.Equal(t, 400, rec.Code)
				require.Empty(t, r.calls)
				message := "invalid test case payload"
				if tc.bindFailure {
					message = "invalid test case " + op.payload + " payload"
				}
				require.JSONEq(t, `{"message":"`+message+`"}`, rec.Body.String())
			})
		}
	}
}

func TestTestCaseListExactCurrentFields(t *testing.T) {
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	r := &fakeTestCaseRepository{cases: []domain.TestCase{{ID: 1 << 40, ChangeID: 42, Scenario: "Current", Done: true, Created: now, Modified: now}}}
	e := echo.New()
	NewAPI(e, NewService(r))
	req := httptest.NewRequest("POST", "/api/v1/test-case/list", strings.NewReader(`{"change_id":42}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)
	require.JSONEq(t, `[{"id":1099511627776,"change_id":42,"scenario":"Current","done":true,"created":"2026-09-28T12:00:00Z","modified":"2026-09-28T12:00:00Z"}]`, rec.Body.String())
	require.Equal(t, []any{domain.TestCaseListRequest{ChangeID: 42}}, r.calls)
}
