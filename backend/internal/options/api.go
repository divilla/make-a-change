package options

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

// API defines API values.
type API struct {
	g *echo.Group
	s *Service
}

// NewAPI initializes API.
func NewAPI(e *echo.Echo, s *Service) *API {
	a := &API{
		g: e.Group("/api").Group("/v1").Group("/options"),
		s: s,
	}

	a.g.POST("/change-phases-list", a.changePhases)
	a.g.POST("/change-types-list", a.changeTypes)

	return a
}

func (a *API) changePhases(c *echo.Context) error {
	res, err := a.s.ChangePhases(c.Request().Context())
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, &res)
}

func (a *API) changeTypes(c *echo.Context) error {
	res, err := a.s.ChangeTypes(c.Request().Context())
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, &res)
}
