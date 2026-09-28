package project

import (
	"mch_api/internal/domain"
	apperror "mch_api/internal/error"
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
		g: e.Group("/api").Group("/v1").Group("/project"),
		s: s,
	}

	a.g.POST("/list", a.listProjects)
	a.g.POST("/config", a.config)
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
		return apperror.HTTP(err)
	}

	return c.JSON(http.StatusOK, &res)
}

func (a *API) getProject(c *echo.Context) error {
	ctx := c.Request().Context()
	var req domain.ProjectIDRequest
	if err := c.Bind(&req); err != nil {
		return apperror.InvalidPayload(err, "invalid project get payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return apperror.HTTP(apperror.ErrProjectInvalidInput)
	}

	res, err := a.s.GetProject(ctx, req)
	if err != nil {
		return apperror.HTTP(err)
	}

	return c.JSON(http.StatusOK, &res)
}

func (a *API) createProject(c *echo.Context) error {
	ctx := c.Request().Context()
	var req domain.ProjectCreateRequest
	if err := c.Bind(&req); err != nil {
		return apperror.InvalidPayload(err, "invalid project create payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return apperror.HTTP(apperror.ErrProjectInvalidInput)
	}

	res, err := a.s.CreateProject(ctx, req)
	if err != nil {
		return apperror.HTTP(err)
	}

	return c.JSON(http.StatusCreated, &res)
}

func (a *API) updateProject(c *echo.Context) error {
	ctx := c.Request().Context()
	var req domain.ProjectUpdateRequest
	if err := c.Bind(&req); err != nil {
		return apperror.InvalidPayload(err, "invalid project update payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return apperror.HTTP(apperror.ErrProjectInvalidInput)
	}

	err := a.s.UpdateProject(ctx, req)
	if err != nil {
		return apperror.HTTP(err)
	}

	return c.NoContent(http.StatusNoContent)
}

func (a *API) deleteProject(c *echo.Context) error {
	ctx := c.Request().Context()
	var req domain.ProjectIDRequest
	if err := c.Bind(&req); err != nil {
		return apperror.InvalidPayload(err, "invalid project delete payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return apperror.HTTP(apperror.ErrProjectInvalidInput)
	}

	if err := a.s.DeleteProject(ctx, req); err != nil {
		return apperror.HTTP(err)
	}

	return c.NoContent(http.StatusNoContent)
}

func (a *API) config(c *echo.Context) error {
	var req domain.ProjectIDRequest
	if err := c.Bind(&req); err != nil {
		return apperror.InvalidPayload(err, "invalid project config payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return apperror.HTTP(apperror.ErrProjectInvalidInput)
	}
	res, err := a.s.Config(c.Request().Context(), req)
	if err != nil {
		return apperror.HTTP(err)
	}
	return c.JSON(http.StatusOK, &res)
}
