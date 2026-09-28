package main

import (
	"context"
	"flag"
	"mch_api/internal/change"
	"mch_api/internal/epic"
	apperror "mch_api/internal/error"
	"mch_api/internal/health"
	"mch_api/internal/options"
	"mch_api/internal/project"
	"mch_api/internal/testcase"
	"mch_api/pkg/config"
	"mch_api/pkg/markdown"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	config.New()
	cfg := config.Get()

	portFlag := flag.String("port", "", "server port")
	dbFlag := flag.String("db", "", "database connection string")
	flag.Parse()
	if *portFlag != "" {
		cfg.Port = *portFlag
	}
	if *dbFlag != "" {
		cfg.ConnectionString = *dbFlag
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, func() (application, error) { return start(ctx, cfg) }); err != nil {
		log.Error().Err(err).Msg("server stopped")
		os.Exit(1)
	}
}

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

	defaultCORSConfig := middleware.CORSConfig{
		Skipper:      middleware.DefaultSkipper,
		AllowOrigins: cfg.AllowedOrigins(),
		AllowMethods: []string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch, http.MethodPost, http.MethodDelete, http.MethodOptions},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAccessControlAllowOrigin, echo.HeaderAccessControlAllowMethods},
	}
	e := echo.New()
	e.HTTPErrorHandler = jsonErrorHandler
	e.Pre(middleware.RemoveTrailingSlash())

	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	log.Logger = logger
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:    true,
		LogStatus: true,
		LogValuesFunc: func(_ *echo.Context, v middleware.RequestLoggerValues) error {
			logger.Info().
				Str("URI", v.URI).
				Int("status", v.Status).
				Msg("request")
			return nil
		},
	}))

	e.Use(middleware.Recover())
	cors, err := defaultCORSConfig.ToMiddleware()
	if err != nil {
		_ = listener.Close()
		pool.Close()
		return application{}, apperror.Wrap(err, "configure CORS")
	}
	e.Use(cors)

	markdownParser := markdown.NewGoldmarkParser()
	htmlSanitizer := markdown.NewBluemondaySanitizer()
	changeRenderer := change.NewRenderer(markdownParser, htmlSanitizer)

	healthRepository := health.NewRepo(pool)
	healthService := health.NewService(healthRepository)
	health.NewAPI(e, healthService)

	projectRepository := project.NewRepo(pool)
	projectService := project.NewService(projectRepository)
	project.NewAPI(e, projectService)

	epicRepository := epic.NewRepo(pool)
	epicService := epic.NewService(epicRepository)
	epic.NewAPI(e, epicService)

	optionsRepository := options.NewRepo(pool)
	optionsService := options.NewService(optionsRepository)
	options.NewAPI(e, optionsService)

	changeRepository := change.NewRepo(pool)
	changeService := change.NewService(changeRepository, changeRenderer)
	change.NewAPI(e, changeService)

	testCaseRepository := testcase.NewRepo(pool)
	testCaseService := testcase.NewService(testCaseRepository, changeRenderer)
	testcase.NewAPI(e, testCaseService)

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

type errorResponse struct {
	Message string `json:"message"`
}

func jsonErrorHandler(c *echo.Context, err error) {
	code, message := apperror.Interpret(err)

	if code >= http.StatusInternalServerError {
		log.Error().Err(err).Msg("request failed")
	}

	if writeErr := c.JSON(code, errorResponse{Message: message}); writeErr != nil {
		log.Error().Err(apperror.Wrap(writeErr, "write error response")).Msg("failed to write error response")
	}
}
