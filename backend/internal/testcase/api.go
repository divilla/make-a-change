package testcase

import (
	"errors"
	"net/http"

	"aipm/internal/dto"

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
	a.g.POST("/update-change", a.updateTestCaseChange)
	a.g.POST("/delete", a.deleteTestCase)

	return a
}

func (a *API) listTestCases(c *echo.Context) error {
	var req dto.TestCaseListRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid test case list payload")
	}
	res, err := a.s.ListTestCases(c.Request().Context(), req)
	if err != nil {
		return testCaseError(err)
	}
	return c.JSON(http.StatusOK, &res)
}

func (a *API) createTestCase(c *echo.Context) error {
	var req dto.TestCaseCreateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid test case create payload")
	}
	res, err := a.s.CreateTestCase(c.Request().Context(), req)
	if err != nil {
		return testCaseError(err)
	}
	return c.JSON(http.StatusCreated, &res)
}

func (a *API) updateTestCase(c *echo.Context) error {
	var req dto.TestCaseUpdateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid test case update payload")
	}
	res, err := a.s.UpdateTestCase(c.Request().Context(), req)
	if err != nil {
		return testCaseError(err)
	}
	return c.JSON(http.StatusOK, &res)
}

func (a *API) updateTestCaseDone(c *echo.Context) error {
	var req dto.TestCaseUpdateDoneRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid test case done payload")
	}
	res, err := a.s.UpdateTestCaseDone(c.Request().Context(), req)
	if err != nil {
		return testCaseError(err)
	}
	return c.JSON(http.StatusOK, &res)
}

func (a *API) updateTestCaseChange(c *echo.Context) error {
	var req dto.TestCaseUpdateChangeRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid test case change payload")
	}
	res, err := a.s.UpdateTestCaseChange(c.Request().Context(), req)
	if err != nil {
		return testCaseError(err)
	}
	return c.JSON(http.StatusOK, &res)
}

func (a *API) deleteTestCase(c *echo.Context) error {
	var req dto.TestCaseIDRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid test case delete payload")
	}
	res, err := a.s.DeleteTestCase(c.Request().Context(), req)
	if err != nil {
		return testCaseError(err)
	}
	return c.JSON(http.StatusOK, &res)
}

func testCaseError(err error) error {
	switch {
	case errors.Is(err, ErrInvalidInput):
		return echo.NewHTTPError(http.StatusBadRequest, "invalid test case payload")
	case errors.Is(err, ErrNotFound):
		return echo.NewHTTPError(http.StatusNotFound, "test case not found")
	default:
		return err
	}
}
