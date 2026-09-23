package project

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
		g: e.Group("/api").Group("/v1").Group("/project"),
		s: s,
	}

	a.g.POST("/list", a.listProjects)
	a.g.POST("/get", a.getProject)
	a.g.POST("/create", a.createProject)
	a.g.POST("/update", a.updateProject)
	a.g.POST("/delete", a.deleteProject)

	return a
}

func (a *API) listProjects(c *echo.Context) error {
	ctx := c.Request().Context()
	res, err := a.s.ListProjects(ctx)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, &res)
}

func (a *API) getProject(c *echo.Context) error {
	ctx := c.Request().Context()
	var req dto.ProjectIDRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid project get payload")
	}

	res, err := a.s.GetProject(ctx, req)
	if err != nil {
		return projectError(err)
	}

	return c.JSON(http.StatusOK, &res)
}

func (a *API) createProject(c *echo.Context) error {
	ctx := c.Request().Context()
	var req dto.ProjectCreateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid project create payload")
	}

	res, err := a.s.CreateProject(ctx, req)
	if err != nil {
		return projectError(err)
	}

	return c.JSON(http.StatusCreated, &res)
}

func (a *API) updateProject(c *echo.Context) error {
	ctx := c.Request().Context()
	var req dto.ProjectUpdateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid project update payload")
	}

	res, err := a.s.UpdateProject(ctx, req)
	if err != nil {
		return projectError(err)
	}

	return c.JSON(http.StatusOK, &res)
}

func (a *API) deleteProject(c *echo.Context) error {
	ctx := c.Request().Context()
	var req dto.ProjectIDRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid project delete payload")
	}

	if err := a.s.DeleteProject(ctx, req); err != nil {
		return projectError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

func projectError(err error) error {
	switch {
	case errors.Is(err, ErrInvalidInput):
		return echo.NewHTTPError(http.StatusBadRequest, "invalid project payload")
	case errors.Is(err, ErrNotFound):
		return echo.NewHTTPError(http.StatusNotFound, "project not found")
	case errors.Is(err, ErrProjectHasChanges):
		return echo.NewHTTPError(http.StatusConflict, "project has changes and cannot be deleted")
	default:
		return err
	}
}
