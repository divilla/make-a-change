package epic

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
		g: e.Group("/api").Group("/v1").Group("/epic"),
		s: s,
	}

	a.g.POST("/list", a.listEpics)
	a.g.POST("/get", a.getEpic)
	a.g.POST("/create", a.createEpic)
	a.g.POST("/update", a.updateEpic)
	a.g.POST("/delete", a.deleteEpic)

	return a
}

func (a *API) listEpics(c *echo.Context) error {
	var req dto.EpicListRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid epic list payload")
	}
	res, err := a.s.ListEpics(c.Request().Context(), req)
	if err != nil {
		return epicError(err)
	}
	return c.JSON(http.StatusOK, &res)
}

func (a *API) getEpic(c *echo.Context) error {
	var req dto.EpicIDRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid epic get payload")
	}
	res, err := a.s.GetEpic(c.Request().Context(), req)
	if err != nil {
		return epicError(err)
	}
	return c.JSON(http.StatusOK, &res)
}

func (a *API) createEpic(c *echo.Context) error {
	var req dto.EpicCreateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid epic create payload")
	}
	res, err := a.s.CreateEpic(c.Request().Context(), req)
	if err != nil {
		return epicError(err)
	}
	return c.JSON(http.StatusCreated, &res)
}

func (a *API) updateEpic(c *echo.Context) error {
	var req dto.EpicUpdateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid epic update payload")
	}
	res, err := a.s.UpdateEpic(c.Request().Context(), req)
	if err != nil {
		return epicError(err)
	}
	return c.JSON(http.StatusOK, &res)
}

func (a *API) deleteEpic(c *echo.Context) error {
	var req dto.EpicIDRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid epic delete payload")
	}
	if err := a.s.DeleteEpic(c.Request().Context(), req); err != nil {
		return epicError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func epicError(err error) error {
	switch {
	case errors.Is(err, ErrInvalidInput):
		return echo.NewHTTPError(http.StatusBadRequest, "invalid epic payload")
	case errors.Is(err, ErrNotFound):
		return echo.NewHTTPError(http.StatusNotFound, "epic not found")
	case errors.Is(err, ErrEpicHasChanges):
		return echo.NewHTTPError(http.StatusConflict, "epic has changes and cannot be deleted")
	default:
		return err
	}
}
