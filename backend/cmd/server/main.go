package main

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"os"

	"aipm/internal/change"
	"aipm/internal/epic"
	"aipm/internal/health"
	"aipm/internal/options"
	"aipm/internal/project"
	"aipm/internal/testcase"
	"aipm/pkg/config"
	"aipm/pkg/db"
	"aipm/pkg/markdown"

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

	ctx := context.Background()
	pool := db.Pool(ctx, cfg.ConnectionString)
	defer pool.Close()

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
		LogValuesFunc: func(c *echo.Context, v middleware.RequestLoggerValues) error {
			logger.Info().
				Str("URI", v.URI).
				Int("status", v.Status).
				Msg("request")
			return nil
		},
	}))

	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(defaultCORSConfig))

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

	if err := e.Start(cfg.Addr()); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error().Err(err).Msg("server stopped")
	}
}

type errorResponse struct {
	Message string `json:"message"`
}

func jsonErrorHandler(c *echo.Context, err error) {
	code := http.StatusInternalServerError
	message := http.StatusText(code)

	if status := echo.StatusCode(err); status != 0 {
		code = status
		message = http.StatusText(code)
	}

	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
		if he.Message != "" {
			message = he.Message
		}
	}

	if code >= http.StatusInternalServerError {
		log.Error().Err(err).Msg("request failed")
	}

	if writeErr := c.JSON(code, errorResponse{Message: message}); writeErr != nil {
		log.Error().Err(writeErr).Msg("failed to write error response")
	}
}
