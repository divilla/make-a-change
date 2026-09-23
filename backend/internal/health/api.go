package health

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

// API defines API values.
type API struct {
	e *echo.Echo
	g *echo.Group
	s *Service
}

// NewAPI initializes or executes NewAPI behavior.
func NewAPI(e *echo.Echo, s *Service) *API {
	a := &API{
		e: e,
		g: e.Group("/api").Group("/v1").Group("/health"),
		s: s,
	}

	a.g.GET("", a.check)
	e.GET("/api/health", a.check)

	return a
}

func (a *API) check(c *echo.Context) error {
	ctx := c.Request().Context()

	res := a.s.Check(ctx)
	if res.Status != "ok" {
		return c.JSON(http.StatusServiceUnavailable, &res)
	}

	return c.JSON(http.StatusOK, &res)
}
