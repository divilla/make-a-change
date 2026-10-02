// Package config provides configuration management by slug.
package config

import (
	"mch_api/internal/app"
	"mch_api/internal/domain"
	"net/http"

	"github.com/gookit/validate/v2"
	"github.com/labstack/echo/v5"
)

// API handles config operations.
type API struct {
	g *echo.Group
	s *Service
}

// NewAPI registers configuration endpoints.
func NewAPI(e *echo.Echo, s *Service) *API {
	a := &API{g: e.Group("/api").Group("/v1").Group("/config"), s: s}
	a.g.POST("/list", a.list)
	a.g.POST("/details", a.details)
	a.g.POST("/insert", a.insert)
	a.g.POST("/update", a.update)
	a.g.POST("/delete", a.delete)
	return a
}

func (a *API) list(c *echo.Context) error {
	result, err := a.s.List(c.Request().Context())
	if err != nil {
		return app.HTTPError(err)
	}
	return c.JSON(http.StatusOK, result)
}

func (a *API) details(c *echo.Context) error {
	var req domain.ConfigSlugRequest
	if err := c.Bind(&req); err != nil {
		return app.PayloadError(err, "invalid config details payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return app.HTTPError(app.ValidationError(v.Errors, app.ErrConfigInvalidInput))
	}
	result, err := a.s.Details(c.Request().Context(), req)
	if err != nil {
		return app.HTTPError(err)
	}
	return c.JSON(http.StatusOK, result)
}

func (a *API) insert(c *echo.Context) error {
	var req domain.ConfigWriteRequest
	if err := c.Bind(&req); err != nil {
		return app.PayloadError(err, "invalid config insert payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return app.HTTPError(app.ValidationError(v.Errors, app.ErrConfigInvalidInput))
	}
	result, err := a.s.Insert(c.Request().Context(), req)
	if err != nil {
		return app.HTTPError(err)
	}
	return c.JSON(http.StatusCreated, result)
}

func (a *API) update(c *echo.Context) error {
	var req domain.ConfigWriteRequest
	if err := c.Bind(&req); err != nil {
		return app.PayloadError(err, "invalid config update payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return app.HTTPError(app.ValidationError(v.Errors, app.ErrConfigInvalidInput))
	}
	if err := a.s.Update(c.Request().Context(), req); err != nil {
		return app.HTTPError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (a *API) delete(c *echo.Context) error {
	var req domain.ConfigSlugRequest
	if err := c.Bind(&req); err != nil {
		return app.PayloadError(err, "invalid config delete payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return app.HTTPError(app.ValidationError(v.Errors, app.ErrConfigInvalidInput))
	}
	if err := a.s.Delete(c.Request().Context(), req); err != nil {
		return app.HTTPError(err)
	}
	return c.NoContent(http.StatusNoContent)
}
