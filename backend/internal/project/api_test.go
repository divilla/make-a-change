package project

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
	for _, op := range []string{"list", "details", "create", "update", "delete", "config"} {
		t.Run(op, func(t *testing.T) {
			r := &fakeProjectRepository{item: domain.Project{ID: 7, Name: "Name", Active: true, CreatedAt: now, UpdatedAt: now, ConfigSlug: "custom", LastRef: 42, ChangeCount: 70000}}
			r.config = domain.Config{Slug: "custom", ProjectDocs: []string{"p2", "p1"}, EpicDocs: []string{"e"}, ChangeDocs: []string{"c"}, ChangePhases: []string{"phase"}, ChangeColors: []string{"blue"}, ChangeTypes: []string{"type"}}
			e := echo.New()
			NewAPI(e, NewService(r))
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			req := httptest.NewRequest("POST", "/api/v1/project/"+op, strings.NewReader(`{"id":7,"project_id":7,"name":" Name "}`)).WithContext(ctx)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			require.Equal(t, []string{op}, r.calls)
			require.Same(t, ctx, r.ctx)
			switch op {
			case "create":
				require.Equal(t, 201, rec.Code)
				require.JSONEq(t, `{"id":7}`, rec.Body.String())
				require.Equal(t, domain.ProjectCreateRequest{Name: "Name"}, r.req)
			case "update", "delete":
				require.Equal(t, 204, rec.Code)
				require.Zero(t, rec.Body.Len())
				if op == "update" {
					require.Equal(t, domain.ProjectUpdateRequest{ID: 7, Name: "Name"}, r.req)
				}
			default:
				require.Equal(t, 200, rec.Code)
				expected := `{"id":7,"name":"Name","active":true,"config_slug":"custom","last_ref":42,"change_count":70000,"created_at":"2026-09-28T00:00:00Z","updated_at":"2026-09-28T00:00:00Z"}`
				if op == "config" {
					expected = `{"slug":"custom","project_docs":["p2","p1"],"epic_docs":["e"],"change_docs":["c"],"change_phases":["phase"],"change_colors":["blue"],"change_types":["type"]}`
				}
				if op == "list" {
					expected = "[" + expected + "]"
				}
				require.JSONEq(t, expected, rec.Body.String())
			}
		})
	}
}

func TestAPIRejectsMalformedAndInvalidRequests(t *testing.T) {
	for _, op := range []string{"details", "create", "update", "delete", "config"} {
		bodies := []struct{ body, message string }{
			{"{", "invalid project " + op + " payload"},
			{`{"id":"bad","project_id":"bad","name":3}`, "invalid project " + op + " payload"},
			{`{}`, "invalid project payload"},
			{`{"id":0,"project_id":0,"name":""}`, "invalid project payload"},
			{`{"id":-1,"project_id":-1,"name":""}`, "invalid project payload"},
		}
		if op == "create" || op == "update" {
			bodies = append(bodies, struct{ body, message string }{`{"id":7,"project_id":7,"name":" \t "}`, "invalid project payload"})
		}
		for _, tc := range bodies {
			t.Run(op+tc.body, func(t *testing.T) {
				r := &fakeProjectRepository{}
				e := echo.New()
				NewAPI(e, NewService(r))
				rec := httptest.NewRecorder()
				req := httptest.NewRequest("POST", "/api/v1/project/"+op, strings.NewReader(tc.body))
				req.Header.Set("Content-Type", "application/json")
				e.ServeHTTP(rec, req)
				require.Equal(t, 400, rec.Code)
				require.JSONEq(t, `{"message":"`+tc.message+`"}`, rec.Body.String())
				require.Empty(t, r.calls)
			})
		}
	}
}

func TestOptionsRoutesUnregistered(t *testing.T) {
	e := echo.New()
	NewAPI(e, NewService(nil))
	for _, path := range []string{"/api/v1/options/change-phases-list", "/api/v1/options/change-types-list"} {
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, httptest.NewRequest("POST", path, strings.NewReader(`{}`)))
		require.Equal(t, 404, rec.Code)
		require.JSONEq(t, `{"message":"Not Found"}`, rec.Body.String())
	}
}

func TestAPIEmptyList(t *testing.T) {
	p := newBoundary(t)
	rows := &valueRows{}
	p.rows = rows

	e := echo.New()
	NewAPI(e, NewService(&Repo{pool: p}))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/project/list", strings.NewReader(`{"project_id":7}`)).WithContext(p.ctx)
	req.Header.Set("Content-Type", "application/json")
	e.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)
	require.JSONEq(t, `[]`, rec.Body.String())
	require.True(t, rows.closed)
}
