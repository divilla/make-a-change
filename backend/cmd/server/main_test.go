package main

import (
	"context"
	"errors"
	"fmt"
	apperror "mch_api/internal/error"
	"mch_api/pkg/config"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/require"
)

func TestHTTPServerPreservesReadTimeout(t *testing.T) {
	handler := echo.New()
	server := newHTTPServer(handler)
	require.Same(t, handler, server.Handler)
	require.Equal(t, 30*time.Second, server.ReadTimeout, "bound reads of the entire request, including its body")
	require.Equal(t, 10*time.Second, server.ReadHeaderTimeout)
	require.Zero(t, server.IdleTimeout, "idle connections inherit ReadTimeout")
}

func TestLifecycle(t *testing.T) {
	failure := errors.New("failure")
	for _, name := range []string{"cancel", "startup", "serve", "shutdown"} {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			events := []string{}
			stopped := make(chan struct{})
			err := run(ctx, func() (application, error) {
				events = append(events, "startup")
				if name == "startup" {
					return application{}, failure
				}
				if name != "serve" {
					cancel()
				}
				return application{
					serve: func() error {
						if name == "serve" {
							return failure
						}
						<-stopped
						return http.ErrServerClosed
					},
					shutdown: func(ctx context.Context) error {
						require.NoError(t, ctx.Err())
						_, ok := ctx.Deadline()
						require.True(t, ok)
						events = append(events, "shutdown")
						close(stopped)
						if name == "shutdown" {
							return failure
						}
						return nil
					},
					close: func() { events = append(events, "close") },
				}, nil
			})
			if name == "cancel" {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, failure)
			}
			switch name {
			case "startup":
				require.Equal(t, []string{"startup"}, events)
			case "serve":
				require.Equal(t, []string{"startup", "close"}, events)
			default:
				require.Equal(t, []string{"startup", "shutdown", "close"}, events)
			}
		})
	}
}

func TestStartFailures(t *testing.T) {
	_, err := start(context.Background(), &config.Config{Port: "0"})
	require.Error(t, err)
	_, err = start(context.Background(), &config.Config{ConnectionString: ":invalid"})
	require.Error(t, err)
	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	defer func() { require.NoError(t, listener.Close()) }()
	_, port, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err)
	_, err = start(context.Background(), &config.Config{Port: port})
	require.Error(t, err)
}

func TestStartCancelRealServer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.NoError(t, run(ctx, func() (application, error) {
		return start(ctx, &config.Config{Port: "0", CORSOrigins: "http://localhost"})
	}))
}

func TestInstalledJSONErrorContracts(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		code int
		body string
	}{
		{"invalid", apperror.ErrChangeInvalidInput, 400, `{"message":"invalid change payload"}`},
		{"missing", apperror.ErrProjectNotFound, 404, `{"message":"project not found"}`},
		{"conflict", apperror.ErrEpicHasChanges, 409, `{"message":"epic has changes and cannot be deleted"}`},
		{"unknown", errors.New("private database details"), 500, `{"message":"Internal Server Error"}`},
		{"wrapped", fmt.Errorf("outer: %w", apperror.ErrTestCaseNotFound), 404, `{"message":"test case not found"}`},
		{"bind", apperror.InvalidPayload(errors.New("decode"), "invalid project get payload"), 400, `{"message":"invalid project get payload"}`},
		{"echo internal", echo.NewHTTPError(500, "secret"), 500, `{"message":"Internal Server Error"}`},
		{"wrapped echo", fmt.Errorf("outer: %w", echo.NewHTTPError(400, "safe message")), 400, `{"message":"safe message"}`},
		{"router", echo.ErrNotFound, 404, `{"message":"Not Found"}`},
		{"method", echo.ErrMethodNotAllowed, 405, `{"message":"Method Not Allowed"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := echo.New()
			e.HTTPErrorHandler = jsonErrorHandler
			e.GET("/test", func(_ *echo.Context) error { return tc.err })
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/test", nil))
			require.Equal(t, tc.code, rec.Code)
			require.Equal(t, tc.body+"\n", rec.Body.String())
			require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
		})
	}
	// Exercise actual router-produced errors using the installed handler.
	e := echo.New()
	e.HTTPErrorHandler = jsonErrorHandler
	e.GET("/test", func(_ *echo.Context) error { return nil })
	for _, tc := range []struct {
		method, path string
		code         int
		body         string
	}{
		{http.MethodGet, "/missing", 404, `{"message":"Not Found"}`},
		{http.MethodPost, "/test", 405, `{"message":"Method Not Allowed"}`},
	} {
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, nil))
		require.Equal(t, tc.code, rec.Code)
		require.Equal(t, tc.body+"\n", rec.Body.String())
	}
}
