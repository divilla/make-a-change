// Package change implements change operations.
package change

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
		g: e.Group("/api").Group("/v1").Group("/change"),
		s: s,
	}

	a.g.POST("/list", a.list)
	a.g.POST("/details", a.details)
	a.g.POST("/create", a.create)
	a.g.POST("/update-epic", a.updateEpic)
	a.g.POST("/update-after-change", a.updateAfterChange)
	a.g.POST("/update-phase", a.updatePhase)
	a.g.POST("/update-open", a.updateOpen)
	a.g.POST("/update-types", a.updateTypes)
	a.g.POST("/update-title", a.updateTitle)
	a.g.POST("/update-pr-url", a.updatePRUrl)
	a.g.POST("/delete", a.delete)

	return a
}

func (a *API) list(c *echo.Context) error {
	var req domain.ChangeListRequest
	if err := c.Bind(&req); err != nil {
		return app.InvalidPayload(err, "invalid change list payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return app.HTTP(app.Validation(v.Errors, app.ErrChangeInvalidInput))
	}
	res, err := a.s.List(c.Request().Context(), req)
	if err != nil {
		return app.HTTP(err)
	}
	return c.JSON(http.StatusOK, &res)
}

func (a *API) details(c *echo.Context) error {
	var req domain.ChangeIDRequest
	if err := c.Bind(&req); err != nil {
		return app.InvalidPayload(err, "invalid change details payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return app.HTTP(app.Validation(v.Errors, app.ErrChangeInvalidInput))
	}
	res, err := a.s.Details(c.Request().Context(), req)
	if err != nil {
		return app.HTTP(err)
	}
	return c.JSON(http.StatusOK, &res)
}

func (a *API) create(c *echo.Context) error {
	var req domain.ChangeCreateRequest
	if err := c.Bind(&req); err != nil {
		return app.InvalidPayload(err, "invalid change create payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return app.HTTP(app.Validation(v.Errors, app.ErrChangeInvalidInput))
	}
	res, err := a.s.Create(c.Request().Context(), req)
	if err != nil {
		return app.HTTP(err)
	}
	return c.JSON(http.StatusCreated, &res)
}

func (a *API) updateEpic(c *echo.Context) error {
	var req domain.ChangeUpdateEpicRequest
	if err := c.Bind(&req); err != nil {
		return app.InvalidPayload(err, "invalid change epic payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return app.HTTP(app.Validation(v.Errors, app.ErrChangeInvalidInput))
	}
	if err := a.s.UpdateEpic(c.Request().Context(), req); err != nil {
		return app.HTTP(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (a *API) updateTypes(c *echo.Context) error {
	var req domain.ChangeUpdateTypesRequest
	if err := c.Bind(&req); err != nil {
		return app.InvalidPayload(err, "invalid change types payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return app.HTTP(app.Validation(v.Errors, app.ErrChangeInvalidInput))
	}
	if err := a.s.UpdateTypes(c.Request().Context(), req); err != nil {
		return app.HTTP(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (a *API) updateTitle(c *echo.Context) error {
	var req domain.ChangeUpdateTitleRequest
	if err := c.Bind(&req); err != nil {
		return app.InvalidPayload(err, "invalid change title payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return app.HTTP(app.Validation(v.Errors, app.ErrChangeInvalidInput))
	}
	if err := a.s.UpdateTitle(c.Request().Context(), req); err != nil {
		return app.HTTP(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (a *API) updatePRUrl(c *echo.Context) error {
	var req domain.ChangeUpdatePRUrlRequest
	if err := c.Bind(&req); err != nil {
		return app.InvalidPayload(err, "invalid change pr url payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return app.HTTP(app.Validation(v.Errors, app.ErrChangeInvalidInput))
	}
	if err := a.s.UpdatePRUrl(c.Request().Context(), req); err != nil {
		return app.HTTP(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (a *API) updatePhase(c *echo.Context) error {
	var req domain.ChangeUpdatePhaseRequest
	if err := c.Bind(&req); err != nil {
		return app.InvalidPayload(err, "invalid change phase payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return app.HTTP(app.Validation(v.Errors, app.ErrChangeInvalidInput))
	}
	if err := a.s.UpdatePhase(c.Request().Context(), req); err != nil {
		return app.HTTP(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (a *API) updateOpen(c *echo.Context) error {
	var req domain.ChangeUpdateOpenRequest
	if err := c.Bind(&req); err != nil {
		return app.InvalidPayload(err, "invalid change open payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return app.HTTP(app.Validation(v.Errors, app.ErrChangeInvalidInput))
	}
	if err := a.s.UpdateOpen(c.Request().Context(), req); err != nil {
		return app.HTTP(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (a *API) delete(c *echo.Context) error {
	var req domain.ChangeIDRequest
	if err := c.Bind(&req); err != nil {
		return app.InvalidPayload(err, "invalid change delete payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return app.HTTP(app.Validation(v.Errors, app.ErrChangeInvalidInput))
	}
	if err := a.s.Delete(c.Request().Context(), req); err != nil {
		return app.HTTP(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (a *API) updateAfterChange(c *echo.Context) error {
	var req domain.ChangeUpdateAfterChangeRequest
	if err := c.Bind(&req); err != nil {
		return app.InvalidPayload(err, "invalid change after change payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return app.HTTP(app.Validation(v.Errors, app.ErrChangeInvalidInput))
	}
	if err := a.s.UpdateAfterChange(c.Request().Context(), req); err != nil {
		return app.HTTP(err)
	}
	return c.NoContent(http.StatusNoContent)
}
