package epic

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
	var req domain.EpicListRequest
	if err := c.Bind(&req); err != nil {
		return apperror.InvalidPayload(err, "invalid epic list payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return apperror.HTTP(apperror.Validation(v.Errors, apperror.ErrEpicInvalidInput))
	}
	res, err := a.s.ListEpics(c.Request().Context(), req)
	if err != nil {
		return apperror.HTTP(err)
	}
	return c.JSON(http.StatusOK, &res)
}

func (a *API) getEpic(c *echo.Context) error {
	var req domain.EpicIDRequest
	if err := c.Bind(&req); err != nil {
		return apperror.InvalidPayload(err, "invalid epic get payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return apperror.HTTP(apperror.Validation(v.Errors, apperror.ErrEpicInvalidInput))
	}
	res, err := a.s.GetEpic(c.Request().Context(), req)
	if err != nil {
		return apperror.HTTP(err)
	}
	return c.JSON(http.StatusOK, &res)
}

func (a *API) createEpic(c *echo.Context) error {
	var req domain.EpicCreateRequest
	if err := c.Bind(&req); err != nil {
		return apperror.InvalidPayload(err, "invalid epic create payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return apperror.HTTP(apperror.Validation(v.Errors, apperror.ErrEpicInvalidInput))
	}
	res, err := a.s.CreateEpic(c.Request().Context(), req)
	if err != nil {
		return apperror.HTTP(err)
	}
	return c.JSON(http.StatusCreated, &res)
}

func (a *API) updateEpic(c *echo.Context) error {
	var req domain.EpicUpdateRequest
	if err := c.Bind(&req); err != nil {
		return apperror.InvalidPayload(err, "invalid epic update payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return apperror.HTTP(apperror.Validation(v.Errors, apperror.ErrEpicInvalidInput))
	}
	err := a.s.UpdateEpic(c.Request().Context(), req)
	if err != nil {
		return apperror.HTTP(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (a *API) deleteEpic(c *echo.Context) error {
	var req domain.EpicIDRequest
	if err := c.Bind(&req); err != nil {
		return apperror.InvalidPayload(err, "invalid epic delete payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return apperror.HTTP(apperror.Validation(v.Errors, apperror.ErrEpicInvalidInput))
	}
	if err := a.s.DeleteEpic(c.Request().Context(), req); err != nil {
		return apperror.HTTP(err)
	}
	return c.NoContent(http.StatusNoContent)
}
