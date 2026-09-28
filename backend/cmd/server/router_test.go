package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mch_api/pkg/config"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v5"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
)

// Tests that replace the process logger must remain serial and restore it.
func captureErrorLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	previous := log.Logger
	var output bytes.Buffer
	log.Logger = zerolog.New(&output)
	t.Cleanup(func() { log.Logger = previous })
	return &output
}

func TestRouterMiddlewareParity(t *testing.T) {
	for _, tc := range []struct {
		name, method, path, origin, allowOrigin, body, allow string
		code                                                 int
		preflight                                            bool
	}{
		{name: "allowed trailing slash", method: "POST", path: "/api/v1/project/details/", origin: "https://allowed.example", allowOrigin: "https://allowed.example", code: 400, body: `{"message":"invalid project payload"}`},
		{name: "denied origin reaches handler", method: "POST", path: "/api/v1/project/details/", origin: "https://denied.example", code: 400, body: `{"message":"invalid project payload"}`},
		{name: "second configured origin", method: "POST", path: "/api/v1/project/details", origin: "https://second.example", allowOrigin: "https://second.example", code: 400, body: `{"message":"invalid project payload"}`},
		{name: "allowed preflight", method: "OPTIONS", path: "/api/v1/project/details", origin: "https://allowed.example", allowOrigin: "https://allowed.example", code: 204, allow: "OPTIONS, POST", preflight: true},
		{name: "denied preflight", method: "OPTIONS", path: "/api/v1/project/details", origin: "https://denied.example", code: 204, allow: "OPTIONS, POST", preflight: true},
		{name: "router missing", method: "GET", path: "/missing", code: 404, body: `{"message":"Not Found"}`},
		{name: "router method", method: "GET", path: "/api/v1/project/details", code: 405, body: `{"message":"Method Not Allowed"}`, allow: "OPTIONS, POST"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			cfg := config.Config{CORSOrigins: " https://allowed.example, https://second.example "}
			// Nil pool makes these invalid-input/router/preflight cases independent of PostgreSQL.
			e, err := newRouter(nil, cfg.AllowedOrigins(), zerolog.New(&output))
			require.NoError(t, err)
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(`{}`))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			req.Header.Set(echo.HeaderOrigin, tc.origin)
			if tc.preflight {
				req.Header.Set(echo.HeaderAccessControlRequestMethod, "POST")
				req.Header.Set(echo.HeaderAccessControlRequestHeaders, "Content-Type")
			}
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			require.Equal(t, tc.code, rec.Code)
			require.Empty(t, rec.Header().Get(echo.HeaderLocation), "trailing slash removal does not redirect")
			require.Equal(t, tc.allowOrigin, rec.Header().Get(echo.HeaderAccessControlAllowOrigin))
			require.Equal(t, tc.allow, rec.Header().Get(echo.HeaderAllow))
			require.Empty(t, rec.Header().Get(echo.HeaderAccessControlAllowCredentials))
			if tc.body != "" {
				require.Equal(t, tc.body+"\n", rec.Body.String())
				require.Equal(t, echo.MIMEApplicationJSON, rec.Header().Get(echo.HeaderContentType))
			} else {
				require.Empty(t, rec.Body.String())
			}
			if tc.preflight && tc.allowOrigin != "" {
				require.Equal(t, "GET,HEAD,PUT,PATCH,POST,DELETE,OPTIONS", rec.Header().Get(echo.HeaderAccessControlAllowMethods))
				require.Equal(t, "Origin,Content-Type,Accept,Access-Control-Allow-Origin,Access-Control-Allow-Methods", rec.Header().Get(echo.HeaderAccessControlAllowHeaders))
				require.Equal(t, []string{"Origin", "Access-Control-Request-Method", "Access-Control-Request-Headers"}, rec.Header().Values(echo.HeaderVary))
			} else {
				require.Empty(t, rec.Header().Get(echo.HeaderAccessControlAllowMethods))
				require.Empty(t, rec.Header().Get(echo.HeaderAccessControlAllowHeaders))
				require.Equal(t, []string{"Origin"}, rec.Header().Values(echo.HeaderVary))
			}
			var event map[string]any
			require.NoError(t, json.Unmarshal(output.Bytes(), &event))
			require.Equal(t, "request", event["message"])
			require.Equal(t, strings.TrimSuffix(tc.path, "/"), event["URI"])
			require.Equal(t, float64(tc.code), event["status"])
		})
	}
}

func TestRouterRecoveryPreservesContextAndLogs(t *testing.T) {
	errorLog := captureErrorLog(t)
	var requestLog bytes.Buffer
	e, err := newRouter(nil, []string{"https://allowed.example"}, zerolog.New(&requestLog))
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	e.GET("/panic", func(c *echo.Context) error {
		require.Same(t, ctx, c.Request().Context())
		panic(errors.New("private panic details"))
	})
	req := httptest.NewRequest(http.MethodGet, "/panic/", nil).WithContext(ctx)
	req.Header.Set(echo.HeaderOrigin, "https://allowed.example")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	require.Equal(t, 500, rec.Code)
	require.Equal(t, "{\"message\":\"Internal Server Error\"}\n", rec.Body.String())
	require.Equal(t, "https://allowed.example", rec.Header().Get(echo.HeaderAccessControlAllowOrigin))
	require.Contains(t, errorLog.String(), `"message":"request failed"`)
	require.Contains(t, errorLog.String(), "private panic details")
	var event map[string]any
	require.NoError(t, json.Unmarshal(requestLog.Bytes(), &event))
	require.Equal(t, float64(500), event["status"])
	require.Equal(t, "/panic", event["URI"])
}

func TestRouterDoesNotAcquireOrClosePool(t *testing.T) {
	for _, origins := range [][]string{{"https://allowed.example"}, nil} {
		cfg, err := pgxpool.ParseConfig("postgres://localhost:1/postgres?sslmode=disable")
		require.NoError(t, err)
		var attempts atomic.Int32
		failure := errors.New("connection deliberately unavailable")
		cfg.BeforeConnect = func(context.Context, *pgx.ConnConfig) error {
			attempts.Add(1)
			return failure
		}
		pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
		require.NoError(t, err)
		t.Cleanup(pool.Close)
		e, err := newRouter(pool, origins, zerolog.Nop())
		if len(origins) == 0 {
			require.ErrorContains(t, err, "configure CORS:")
			require.NotNil(t, errors.Unwrap(err))
			require.Nil(t, e)
		} else {
			require.NoError(t, err)
			require.NotNil(t, e)
		}
		require.Zero(t, attempts.Load(), "route construction must not connect or ping")
		_, err = pool.Acquire(context.Background())
		require.ErrorIs(t, err, failure, "caller still owns an open pool, including after router failure")
		require.Equal(t, int32(1), attempts.Load())
	}
}

type failingResponseWriter struct {
	*httptest.ResponseRecorder
	attempted bytes.Buffer
	failure   error
	writes    int
}

func (w *failingResponseWriter) Write(body []byte) (int, error) {
	w.writes++
	_, _ = w.attempted.Write(body)
	return 0, w.failure
}

func TestJSONErrorWriteFailure(t *testing.T) {
	output := captureErrorLog(t)
	e, err := newRouter(nil, []string{"https://allowed.example"}, zerolog.Nop())
	require.NoError(t, err)
	e.GET("/failure", func(*echo.Context) error { return errors.New("private database details") })
	writer := &failingResponseWriter{ResponseRecorder: httptest.NewRecorder(), failure: errors.New("broken response transport")}
	e.ServeHTTP(writer, httptest.NewRequest(http.MethodGet, "/failure", nil))
	require.Equal(t, 500, writer.Code)
	require.Equal(t, echo.MIMEApplicationJSON, writer.Header().Get(echo.HeaderContentType))
	require.Equal(t, 1, writer.writes, "do not retry with raw internal details")
	require.Equal(t, "{\"message\":\"Internal Server Error\"}\n", writer.attempted.String())
	require.Empty(t, writer.Body.String(), "transport rejected the safe body")
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	require.Len(t, lines, 2)
	var original, writeFailure map[string]any
	require.NoError(t, json.Unmarshal([]byte(lines[0]), &original))
	require.NoError(t, json.Unmarshal([]byte(lines[1]), &writeFailure))
	require.Equal(t, "request failed", original["message"])
	require.Equal(t, "private database details", original["error"])
	require.Equal(t, "failed to write error response", writeFailure["message"])
	require.Equal(t, "write error response: broken response transport", writeFailure["error"])
}
