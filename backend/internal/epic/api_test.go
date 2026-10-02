package epic

import (
	"context"
	"mch_api/internal/domain"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/require"
)

func TestAPIRegisteredContracts(t *testing.T) {
	now := time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)
	for _, op := range []string{"list", "list-inactive", "details", "create", "update", "delete"} {
		t.Run(op, func(t *testing.T) {
			r := &fakeEpicRepository{item: domain.Epic{ID: 7, Name: "Name", Active: true, CreatedAt: now, UpdatedAt: now, ProjectID: 7, DoneTC: 1, TotalTC: 2, ChangeCount: 70000}}

			e := echo.New()
			NewAPI(e, NewService(r))
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			req := httptest.NewRequest("POST", "/api/v1/epic/"+op, strings.NewReader(`{"id":7,"project_id":7,"name":" Name "}`)).WithContext(ctx)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			require.Equal(t, []string{op}, r.calls)
			require.Same(t, ctx, r.ctx)
			switch op {
			case "create":
				require.Equal(t, 201, rec.Code)
				require.JSONEq(t, `{"id":7}`, rec.Body.String())
				require.Equal(t, domain.EpicCreateRequest{ProjectID: 7, Name: "Name"}, r.req)
			case "update", "delete":
				require.Equal(t, 204, rec.Code)
				require.Zero(t, rec.Body.Len())
				if op == "update" {
					require.Equal(t, domain.EpicUpdateRequest{ID: 7, Name: "Name"}, r.req)
				}
			default:
				require.Equal(t, 200, rec.Code)
				expected := `{"id":7,"project_id":7,"name":"Name","active":true,"done_tc":1,"total_tc":2,"completed":50,"change_count":70000,"created_at":"2026-09-28T00:00:00Z","updated_at":"2026-09-28T00:00:00Z"}`
				if op == "list" || op == "list-inactive" {
					expected = "[" + expected + "]"
				}
				require.JSONEq(t, expected, rec.Body.String())
			}
		})
	}
}

func TestAPIRejectsMalformedAndInvalidRequests(t *testing.T) {
	for _, op := range []string{"details", "create", "update", "delete", "list", "list-inactive"} {
		bodies := []struct{ body, message string }{
			{"{", "invalid epic " + strings.ReplaceAll(op, "list-inactive", "inactive list") + " payload"},
			{`{"id":"bad","project_id":"bad","name":3}`, "invalid epic " + strings.ReplaceAll(op, "list-inactive", "inactive list") + " payload"},
			{`{}`, "invalid epic payload"},
			{`{"id":0,"project_id":0,"name":""}`, "invalid epic payload"},
			{`{"id":-1,"project_id":-1,"name":""}`, "invalid epic payload"},
		}
		if op == "create" || op == "update" {
			bodies = append(bodies, struct{ body, message string }{`{"id":7,"project_id":7,"name":" \t "}`, "invalid epic payload"})
		}
		for _, tc := range bodies {
			t.Run(op+tc.body, func(t *testing.T) {
				r := &fakeEpicRepository{}
				e := echo.New()
				NewAPI(e, NewService(r))
				rec := httptest.NewRecorder()
				req := httptest.NewRequest("POST", "/api/v1/epic/"+op, strings.NewReader(tc.body))
				req.Header.Set("Content-Type", "application/json")
				e.ServeHTTP(rec, req)
				require.Equal(t, 400, rec.Code)
				require.JSONEq(t, `{"message":"`+tc.message+`"}`, rec.Body.String())
				require.Empty(t, r.calls)
			})
		}
	}
}

func TestAPIEmptyList(t *testing.T) {
	p := newBoundary(t)
	rows := &valueRows{}
	p.rows = rows
	p.args = []any{7}
	e := echo.New()
	NewAPI(e, NewService(&Repo{pool: p}))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/epic/list", strings.NewReader(`{"project_id":7}`)).WithContext(p.ctx)
	req.Header.Set("Content-Type", "application/json")
	e.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)
	require.JSONEq(t, `[]`, rec.Body.String())
	require.True(t, rows.closed)
}
