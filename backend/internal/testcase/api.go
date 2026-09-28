// Package testcase exposes independent reads and mutations of current testcases.
package testcase

import (
	"mch_api/internal/domain"
	apperror "mch_api/internal/error"
	"net/http"

	"github.com/gookit/validate/v2"
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
		g: e.Group("/api").Group("/v1").Group("/test-case"),
		s: s,
	}

	a.g.POST("/list", a.listTestCases)
	a.g.POST("/create", a.createTestCase)
	a.g.POST("/update", a.updateTestCase)
	a.g.POST("/update-done", a.updateTestCaseDone)
	a.g.POST("/delete", a.deleteTestCase)

	return a
}

func (a *API) listTestCases(c *echo.Context) error {
	var req domain.TestCaseListRequest
	if err := c.Bind(&req); err != nil {
		return apperror.InvalidPayload(err, "invalid test case list payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return apperror.HTTP(apperror.ErrTestCaseInvalidInput)
	}
	res, err := a.s.ListTestCases(c.Request().Context(), req)
	if err != nil {
		return apperror.HTTP(err)
	}
	return c.JSON(http.StatusOK, &res)
}

func (a *API) createTestCase(c *echo.Context) error {
	var req domain.TestCaseCreateRequest
	if err := c.Bind(&req); err != nil {
		return apperror.InvalidPayload(err, "invalid test case create payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return apperror.HTTP(apperror.ErrTestCaseInvalidInput)
	}
	res, err := a.s.CreateTestCase(c.Request().Context(), req)
	if err != nil {
		return apperror.HTTP(err)
	}
	return c.JSON(http.StatusCreated, &res)
}

func (a *API) updateTestCase(c *echo.Context) error {
	var req domain.TestCaseUpdateRequest
	if err := c.Bind(&req); err != nil {
		return apperror.InvalidPayload(err, "invalid test case update payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return apperror.HTTP(apperror.ErrTestCaseInvalidInput)
	}
	err := a.s.UpdateTestCase(c.Request().Context(), req)
	if err != nil {
		return apperror.HTTP(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (a *API) updateTestCaseDone(c *echo.Context) error {
	var req domain.TestCaseUpdateDoneRequest
	if err := c.Bind(&req); err != nil {
		return apperror.InvalidPayload(err, "invalid test case done payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return apperror.HTTP(apperror.ErrTestCaseInvalidInput)
	}
	err := a.s.UpdateTestCaseDone(c.Request().Context(), req)
	if err != nil {
		return apperror.HTTP(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (a *API) deleteTestCase(c *echo.Context) error {
	var req domain.TestCaseIDRequest
	if err := c.Bind(&req); err != nil {
		return apperror.InvalidPayload(err, "invalid test case delete payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return apperror.HTTP(apperror.ErrTestCaseInvalidInput)
	}
	err := a.s.DeleteTestCase(c.Request().Context(), req)
	if err != nil {
		return apperror.HTTP(err)
	}
	return c.NoContent(http.StatusNoContent)
}
