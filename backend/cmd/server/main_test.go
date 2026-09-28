package main

import (
	"context"
	"errors"
	"fmt"
	"mch_api/internal/app"
	"mch_api/pkg/config"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/labstack/echo/v5"
	"github.com/rs/zerolog"
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
						deadline, ok := ctx.Deadline()
						require.True(t, ok)
						require.InDelta(t, 10, time.Until(deadline).Seconds(), 1)
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
	captureErrorLog(t)
	_, err := start(context.Background(), &config.Config{Port: "0"})
	require.ErrorContains(t, err, "configure CORS:")
	_, err = start(context.Background(), &config.Config{ConnectionString: ":invalid"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "connect database:")
	var parseErr *pgconn.ParseConfigError
	require.ErrorAs(t, err, &parseErr)
	require.NotNil(t, errors.Unwrap(err))
	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	defer func() { require.NoError(t, listener.Close()) }()
	_, port, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err)
	_, err = start(context.Background(), &config.Config{Port: port})
	require.ErrorContains(t, err, "listen:")
	var listenErr *net.OpError
	require.ErrorAs(t, err, &listenErr)
}

func TestStartCancelRealServer(t *testing.T) {
	captureErrorLog(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.NoError(t, run(ctx, func() (application, error) {
		return start(ctx, &config.Config{Port: "0", CORSOrigins: "http://localhost"})
	}))
}

func TestInstalledJSONErrorContracts(t *testing.T) {
	captureErrorLog(t)
	for _, tc := range []struct {
		name string
		err  error
		code int
		body string
	}{
		{"invalid", app.ErrChangeInvalidInput, 400, `{"message":"invalid change payload"}`},
		{"missing", app.ErrProjectNotFound, 404, `{"message":"project not found"}`},
		{"conflict", app.ErrEpicHasChanges, 409, `{"message":"epic has changes and cannot be deleted"}`},
		{"unknown", errors.New("private database details"), 500, `{"message":"Internal Server Error"}`},
		{"wrapped", fmt.Errorf("outer: %w", app.ErrTestCaseNotFound), 404, `{"message":"test case not found"}`},
		{"bind", app.InvalidPayload(errors.New("decode"), "invalid project details payload"), 400, `{"message":"invalid project details payload"}`},
		{"echo internal", echo.NewHTTPError(500, "secret"), 500, `{"message":"Internal Server Error"}`},
		{"wrapped echo", fmt.Errorf("outer: %w", echo.NewHTTPError(400, "safe message")), 400, `{"message":"safe message"}`},
		{"router", echo.ErrNotFound, 404, `{"message":"Not Found"}`},
		{"method", echo.ErrMethodNotAllowed, 405, `{"message":"Method Not Allowed"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e, err := newRouter(nil, []string{"https://allowed.example"}, zerolog.Nop())
			require.NoError(t, err)
			e.GET("/test", func(_ *echo.Context) error { return tc.err })
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/test", nil))
			require.Equal(t, tc.code, rec.Code)
			require.Equal(t, tc.body+"\n", rec.Body.String())
			require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
		})
	}
}

func TestStartCORSFailureReleasesListener(t *testing.T) {
	captureErrorLog(t)
	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	addr := listener.Addr().String()
	_, port, err := net.SplitHostPort(addr)
	require.NoError(t, err)
	require.NoError(t, listener.Close())
	// A valid but unreachable database preserves lazy startup and CORS precedence.
	_, err = start(context.Background(), &config.Config{Port: port, ConnectionString: "postgres://localhost:1/postgres?sslmode=disable"})
	require.ErrorContains(t, err, "configure CORS:")
	require.NotNil(t, errors.Unwrap(err))
	listener, err = net.Listen("tcp", addr)
	require.NoError(t, err, "failed router setup must release its listener")
	require.NoError(t, listener.Close())
}

func TestLifecycleFailedShutdownClosesBeforeWaiting(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	failure := errors.New("shutdown failed")
	closed := make(chan struct{})
	result := make(chan error, 1)
	var events []string
	go func() {
		result <- run(ctx, func() (application, error) {
			return application{
				serve:    func() error { cancel(); <-closed; return http.ErrServerClosed },
				shutdown: func(context.Context) error { events = append(events, "shutdown"); return failure },
				close:    func() { events = append(events, "close"); close(closed) },
			}, nil
		})
	}()
	select {
	case err := <-result:
		require.ErrorIs(t, err, failure)
		require.EqualError(t, err, "shutdown: shutdown failed")
		require.Equal(t, []string{"shutdown", "close"}, events)
	case <-time.After(2 * time.Second):
		t.Fatal("failed shutdown did not close resources to unblock serving")
	}
}
