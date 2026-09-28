// Package epic manages project epics and their change completion summaries.
package epic

import (
	"mch_api/internal/app"
	"mch_api/internal/domain"
	"net/http"

	"github.com/gookit/validate/v2"
	"github.com/labstack/echo/v5"
)

// API defines API values.
type API struct {
	g *echo.Group
	s *Service
}

// NewAPI initializes or executes NewAPI behavior.
func NewAPI(e *echo.Echo, s *Service) *API {
	a := &API{
		g: e.Group("/api").Group("/v1").Group("/epic"),
		s: s,
	}

	a.g.POST("/list", a.list)
	a.g.POST("/details", a.details)
	a.g.POST("/create", a.create)
	a.g.POST("/update", a.updateEpic)
	a.g.POST("/delete", a.delete)

	return a
}

func (a *API) list(c *echo.Context) error {
	var req domain.EpicListRequest
	if err := c.Bind(&req); err != nil {
		return app.InvalidPayload(err, "invalid epic list payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return app.HTTP(app.Validation(v.Errors, app.ErrEpicInvalidInput))
	}
	res, err := a.s.List(c.Request().Context(), req)
	if err != nil {
		return app.HTTP(err)
	}
	return c.JSON(http.StatusOK, &res)
}

func (a *API) details(c *echo.Context) error {
	var req domain.EpicIDRequest
	if err := c.Bind(&req); err != nil {
		return app.InvalidPayload(err, "invalid epic details payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return app.HTTP(app.Validation(v.Errors, app.ErrEpicInvalidInput))
	}
	res, err := a.s.Details(c.Request().Context(), req)
	if err != nil {
		return app.HTTP(err)
	}
	return c.JSON(http.StatusOK, &res)
}

func (a *API) create(c *echo.Context) error {
	var req domain.EpicCreateRequest
	if err := c.Bind(&req); err != nil {
		return app.InvalidPayload(err, "invalid epic create payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return app.HTTP(app.Validation(v.Errors, app.ErrEpicInvalidInput))
	}
	res, err := a.s.Create(c.Request().Context(), req)
	if err != nil {
		return app.HTTP(err)
	}
	return c.JSON(http.StatusCreated, &res)
}

func (a *API) updateEpic(c *echo.Context) error {
	var req domain.EpicUpdateRequest
	if err := c.Bind(&req); err != nil {
		return app.InvalidPayload(err, "invalid epic update payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return app.HTTP(app.Validation(v.Errors, app.ErrEpicInvalidInput))
	}
	if err := a.s.UpdateEpic(c.Request().Context(), req); err != nil {
		return app.HTTP(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (a *API) delete(c *echo.Context) error {
	var req domain.EpicIDRequest
	if err := c.Bind(&req); err != nil {
		return app.InvalidPayload(err, "invalid epic delete payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return app.HTTP(app.Validation(v.Errors, app.ErrEpicInvalidInput))
	}
	if err := a.s.Delete(c.Request().Context(), req); err != nil {
		return app.HTTP(err)
	}
	return c.NoContent(http.StatusNoContent)
}
