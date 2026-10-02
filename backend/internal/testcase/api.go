// Package testcase exposes independent reads and mutations of current testcases.
package testcase

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
		g: e.Group("/api").Group("/v1").Group("/test-case"),
		s: s,
	}

	a.g.POST("/list", a.list)
	a.g.POST("/create", a.create)
	a.g.POST("/update", a.updateTestCase)
	a.g.POST("/update-done", a.updateTestCaseDone)
	a.g.POST("/delete", a.delete)

	return a
}

func (a *API) list(c *echo.Context) error {
	var req domain.TestCaseListRequest
	if err := c.Bind(&req); err != nil {
		return app.PayloadError(err, "invalid test case list payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return app.HTTPError(app.ValidationError(v.Errors, app.ErrTestCaseInvalidInput))
	}
	res, err := a.s.List(c.Request().Context(), req)
	if err != nil {
		return app.HTTPError(err)
	}
	return c.JSON(http.StatusOK, &res)
}

func (a *API) create(c *echo.Context) error {
	var req domain.TestCaseCreateRequest
	if err := c.Bind(&req); err != nil {
		return app.PayloadError(err, "invalid test case create payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return app.HTTPError(app.ValidationError(v.Errors, app.ErrTestCaseInvalidInput))
	}
	res, err := a.s.Create(c.Request().Context(), req)
	if err != nil {
		return app.HTTPError(err)
	}
	return c.JSON(http.StatusCreated, &res)
}

func (a *API) updateTestCase(c *echo.Context) error {
	var req domain.TestCaseUpdateRequest
	if err := c.Bind(&req); err != nil {
		return app.PayloadError(err, "invalid test case update payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return app.HTTPError(app.ValidationError(v.Errors, app.ErrTestCaseInvalidInput))
	}
	if err := a.s.UpdateTestCase(c.Request().Context(), req); err != nil {
		return app.HTTPError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (a *API) updateTestCaseDone(c *echo.Context) error {
	var req domain.TestCaseUpdateDoneRequest
	if err := c.Bind(&req); err != nil {
		return app.PayloadError(err, "invalid test case done payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return app.HTTPError(app.ValidationError(v.Errors, app.ErrTestCaseInvalidInput))
	}
	if err := a.s.UpdateTestCaseDone(c.Request().Context(), req); err != nil {
		return app.HTTPError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (a *API) delete(c *echo.Context) error {
	var req domain.TestCaseIDRequest
	if err := c.Bind(&req); err != nil {
		return app.PayloadError(err, "invalid test case delete payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return app.HTTPError(app.ValidationError(v.Errors, app.ErrTestCaseInvalidInput))
	}
	if err := a.s.Delete(c.Request().Context(), req); err != nil {
		return app.HTTPError(err)
	}
	return c.NoContent(http.StatusNoContent)
}
