// Package doc provides shared docs for projects, epics and changes.
package doc

import (
	"mch_api/internal/app"
	"mch_api/internal/domain"
	"net/http"

	"github.com/gookit/validate/v2"
	"github.com/labstack/echo/v5"
)

// API handles doc requests.
type API struct {
	g *echo.Group
	s *Service
}

// NewAPI registers explicit doc operations.
func NewAPI(e *echo.Echo, s *Service) *API {
	a := &API{g: e.Group("/api").Group("/v1").Group("/doc"), s: s}
	a.g.POST("/list", a.list)
	a.g.POST("/current", a.current)
	a.g.POST("/details", a.details)
	a.g.POST("/insert", a.insert)
	return a
}

func (a *API) list(c *echo.Context) error {
	var req domain.DocListRequest
	if err := c.Bind(&req); err != nil {
		return app.InvalidPayload(err, "invalid doc list payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return app.HTTP(app.Validation(v.Errors, app.ErrDocInvalidInput))
	}
	result, err := a.s.List(c.Request().Context(), req)
	if err != nil {
		return app.HTTP(err)
	}
	return c.JSON(http.StatusOK, result)
}

func (a *API) current(c *echo.Context) error {
	var req domain.DocListRequest
	if err := c.Bind(&req); err != nil {
		return app.InvalidPayload(err, "invalid doc current payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return app.HTTP(app.Validation(v.Errors, app.ErrDocInvalidInput))
	}
	result, err := a.s.Current(c.Request().Context(), req)
	if err != nil {
		return app.HTTP(err)
	}
	return c.JSON(http.StatusOK, result)
}

func (a *API) details(c *echo.Context) error {
	var req domain.DocIDRequest
	if err := c.Bind(&req); err != nil {
		return app.InvalidPayload(err, "invalid doc details payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return app.HTTP(app.Validation(v.Errors, app.ErrDocInvalidInput))
	}
	result, err := a.s.Details(c.Request().Context(), req)
	if err != nil {
		return app.HTTP(err)
	}
	return c.JSON(http.StatusOK, result)
}

func (a *API) insert(c *echo.Context) error {
	var req domain.DocInsertRequest
	if err := c.Bind(&req); err != nil {
		return app.InvalidPayload(err, "invalid doc insert payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return app.HTTP(app.Validation(v.Errors, app.ErrDocInvalidInput))
	}
	result, err := a.s.Insert(c.Request().Context(), req)
	if err != nil {
		return app.HTTP(err)
	}
	return c.JSON(http.StatusCreated, result)
}
