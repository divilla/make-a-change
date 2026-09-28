package main

import (
	"context"
	apperror "mch_api/internal/error"
	"mch_api/pkg/config"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type application struct {
	serve    func() error
	shutdown func(context.Context) error
	close    func()
}

func run(ctx context.Context, startup func() (application, error)) error {
	app, err := startup()
	if err != nil {
		return apperror.Wrap(err, "startup")
	}
	closeApp := sync.OnceFunc(app.close)
	defer closeApp()
	served := make(chan error, 1)
	go func() { served <- app.serve() }()
	var serveErr error
	select {
	case serveErr = <-served:
		return apperror.Wrap(serveErr, "serve")
	case <-ctx.Done():
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	shutdownErr := app.shutdown(shutdownCtx)
	// A failed Shutdown may leave Serve blocked. Close owned resources first.
	if shutdownErr != nil {
		closeApp()
		<-served
		return apperror.Wrap(shutdownErr, "shutdown")
	}
	serveErr = <-served
	return apperror.ServerShutdown(serveErr)
}

func start(ctx context.Context, cfg *config.Config) (application, error) {
	pool, err := pgxpool.New(ctx, cfg.ConnectionString)
	if err != nil {
		return application{}, apperror.Wrap(err, "connect database")
	}
	listener, err := net.Listen("tcp", cfg.Addr())
	if err != nil {
		pool.Close()
		return application{}, apperror.Wrap(err, "listen")
	}

	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	log.Logger = logger
	e, err := newRouter(pool, cfg.AllowedOrigins(), logger)
	if err != nil {
		_ = listener.Close()
		pool.Close()
		return application{}, err
	}

	server := newHTTPServer(e)
	return application{
		serve:    func() error { return server.Serve(listener) },
		shutdown: server.Shutdown,
		close:    func() { _ = server.Close(); _ = listener.Close(); pool.Close() },
	}, nil
}

func newHTTPServer(handler http.Handler) *http.Server {
	return &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		// Preserve Echo's request-body deadline and implicit idle timeout.
		ReadTimeout: 30 * time.Second,
	}
}
