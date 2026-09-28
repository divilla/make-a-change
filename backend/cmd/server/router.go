package main

import (
	"mch_api/internal/change"
	"mch_api/internal/epic"
	apperror "mch_api/internal/error"
	"mch_api/internal/health"
	"mch_api/internal/project"
	"mch_api/internal/testcase"
	"mch_api/pkg/markdown"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// newRouter composes HTTP handling without acquiring or closing resources.
func newRouter(pool *pgxpool.Pool, allowedOrigins []string, logger zerolog.Logger) (*echo.Echo, error) {
	defaultCORSConfig := middleware.CORSConfig{
		Skipper:      middleware.DefaultSkipper,
		AllowOrigins: allowedOrigins,
		AllowMethods: []string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch, http.MethodPost, http.MethodDelete, http.MethodOptions},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAccessControlAllowOrigin, echo.HeaderAccessControlAllowMethods},
	}
	e := echo.New()
	e.HTTPErrorHandler = jsonErrorHandler
	e.Pre(middleware.RemoveTrailingSlash())

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
		return nil, apperror.Wrap(err, "configure CORS")
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

	changeRepository := change.NewRepo(pool)
	changeService := change.NewService(changeRepository, changeRenderer, projectService)
	change.NewAPI(e, changeService)

	testCaseRepository := testcase.NewRepo(pool)
	testCaseService := testcase.NewService(testCaseRepository)
	testcase.NewAPI(e, testCaseService)

	return e, nil
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
