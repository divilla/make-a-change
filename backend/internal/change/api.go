package change

import (
	"errors"
	"net/http"

	"mch_api/internal/domain"

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
		g: e.Group("/api").Group("/v1").Group("/change"),
		s: s,
	}

	a.g.POST("/list", a.listChanges)
	a.g.POST("/get", a.getChange)
	a.g.POST("/rendered-artifacts", a.renderedArtifacts)
	a.g.POST("/create", a.createChange)
	a.g.POST("/update-epic", a.updateEpic)
	a.g.POST("/update-phase", a.updatePhase)
	a.g.POST("/update-open", a.updateOpen)
	a.g.POST("/update-change-types", a.updateChangeTypes)
	a.g.POST("/update-title", a.updateTitle)
	a.g.POST("/update-brief", a.updateBrief)
	a.g.POST("/update-spec", a.updateSpec)
	a.g.POST("/update-pr", a.updatePR)
	a.g.POST("/update-pr-url", a.updatePRUrl)
	a.g.POST("/delete", a.deleteChange)

	return a
}

func (a *API) listChanges(c *echo.Context) error {
	var req domain.ChangeListRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid change list payload")
	}
	res, err := a.s.ListChanges(c.Request().Context(), req)
	if err != nil {
		return changeError(err)
	}
	return c.JSON(http.StatusOK, &res)
}

func (a *API) getChange(c *echo.Context) error {
	var req domain.ChangeIDRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid change get payload")
	}
	res, err := a.s.GetChange(c.Request().Context(), req)
	if err != nil {
		return changeError(err)
	}
	return c.JSON(http.StatusOK, &res)
}

func (a *API) renderedArtifacts(c *echo.Context) error {
	var req domain.ChangeRenderedArtifactsRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid change rendered artifacts payload")
	}
	res, err := a.s.RenderedArtifacts(c.Request().Context(), req)
	if err != nil {
		return changeError(err)
	}
	return c.JSON(http.StatusOK, &res)
}

func (a *API) createChange(c *echo.Context) error {
	var req domain.ChangeCreateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid change create payload")
	}
	res, err := a.s.CreateChange(c.Request().Context(), req)
	if err != nil {
		return changeError(err)
	}
	return c.JSON(http.StatusCreated, &res)
}

func (a *API) updateEpic(c *echo.Context) error {
	var req domain.ChangeUpdateEpicRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid change epic payload")
	}
	res, err := a.s.UpdateEpic(c.Request().Context(), req)
	if err != nil {
		return changeError(err)
	}
	return c.JSON(http.StatusOK, &res)
}

func (a *API) updateChangeTypes(c *echo.Context) error {
	var req domain.ChangeUpdateChangeTypesRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid change types payload")
	}
	res, err := a.s.UpdateChangeTypes(c.Request().Context(), req)
	if err != nil {
		return changeError(err)
	}
	return c.JSON(http.StatusOK, &res)
}

func (a *API) updateTitle(c *echo.Context) error {
	var req domain.ChangeUpdateTitleRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid change title payload")
	}
	res, err := a.s.UpdateTitle(c.Request().Context(), req)
	if err != nil {
		return changeError(err)
	}
	return c.JSON(http.StatusOK, &res)
}

func (a *API) updateBrief(c *echo.Context) error {
	var req domain.ChangeUpdateBriefRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid change brief payload")
	}
	res, err := a.s.UpdateBrief(c.Request().Context(), req)
	if err != nil {
		return changeError(err)
	}
	return c.JSON(http.StatusOK, &res)
}

func (a *API) updateSpec(c *echo.Context) error {
	var req domain.ChangeUpdateSpecRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid change spec payload")
	}
	res, err := a.s.UpdateSpec(c.Request().Context(), req)
	if err != nil {
		return changeError(err)
	}
	return c.JSON(http.StatusOK, &res)
}

func (a *API) updatePR(c *echo.Context) error {
	var req domain.ChangeUpdatePRRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid change pr payload")
	}
	res, err := a.s.UpdatePR(c.Request().Context(), req)
	if err != nil {
		return changeError(err)
	}
	return c.JSON(http.StatusOK, &res)
}

func (a *API) updatePRUrl(c *echo.Context) error {
	var req domain.ChangeUpdatePRUrlRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid change pr url payload")
	}
	res, err := a.s.UpdatePRUrl(c.Request().Context(), req)
	if err != nil {
		return changeError(err)
	}
	return c.JSON(http.StatusOK, &res)
}

func (a *API) updatePhase(c *echo.Context) error {
	var req domain.ChangeUpdatePhaseRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid change phase payload")
	}
	res, err := a.s.UpdatePhase(c.Request().Context(), req)
	if err != nil {
		return changeError(err)
	}
	return c.JSON(http.StatusOK, &res)
}

func (a *API) updateOpen(c *echo.Context) error {
	var req domain.ChangeUpdateOpenRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid change open payload")
	}
	res, err := a.s.UpdateOpen(c.Request().Context(), req)
	if err != nil {
		return changeError(err)
	}
	return c.JSON(http.StatusOK, &res)
}

func (a *API) deleteChange(c *echo.Context) error {
	var req domain.ChangeIDRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid change delete payload")
	}
	if err := a.s.DeleteChange(c.Request().Context(), req); err != nil {
		return changeError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func changeError(err error) error {
	switch {
	case errors.Is(err, ErrInvalidInput):
		return echo.NewHTTPError(http.StatusBadRequest, "invalid change payload")
	case errors.Is(err, ErrInvalidReference):
		return echo.NewHTTPError(http.StatusBadRequest, "invalid change reference")
	case errors.Is(err, ErrNotFound):
		return echo.NewHTTPError(http.StatusNotFound, "change not found")
	default:
		return err
	}
}
