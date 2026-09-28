// Package change implements change operations and current document reads and writes.
package change

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
	a.g.POST("/documents", a.documents)
	a.g.POST("/set-document", a.setDocument)

	return a
}

func (a *API) listChanges(c *echo.Context) error {
	var req domain.ChangeListRequest
	if err := c.Bind(&req); err != nil {
		return apperror.InvalidPayload(err, "invalid change list payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return apperror.HTTP(apperror.Validation(v.Errors, apperror.ErrChangeInvalidInput))
	}
	res, err := a.s.ListChanges(c.Request().Context(), req)
	if err != nil {
		return apperror.HTTP(err)
	}
	return c.JSON(http.StatusOK, &res)
}

func (a *API) getChange(c *echo.Context) error {
	var req domain.ChangeIDRequest
	if err := c.Bind(&req); err != nil {
		return apperror.InvalidPayload(err, "invalid change get payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return apperror.HTTP(apperror.Validation(v.Errors, apperror.ErrChangeInvalidInput))
	}
	res, err := a.s.GetChange(c.Request().Context(), req)
	if err != nil {
		return apperror.HTTP(err)
	}
	return c.JSON(http.StatusOK, &res)
}

func (a *API) renderedArtifacts(c *echo.Context) error {
	var req domain.ChangeRenderedArtifactsRequest
	if err := c.Bind(&req); err != nil {
		return apperror.InvalidPayload(err, "invalid change rendered artifacts payload")
	}
	res, err := a.s.RenderedArtifacts(c.Request().Context(), req)
	if err != nil {
		return apperror.HTTP(err)
	}
	return c.JSON(http.StatusOK, &res)
}

func (a *API) createChange(c *echo.Context) error {
	var req domain.ChangeCreateRequest
	if err := c.Bind(&req); err != nil {
		return apperror.InvalidPayload(err, "invalid change create payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return apperror.HTTP(apperror.Validation(v.Errors, apperror.ErrChangeInvalidInput))
	}
	res, err := a.s.CreateChange(c.Request().Context(), req)
	if err != nil {
		return apperror.HTTP(err)
	}
	return c.JSON(http.StatusCreated, &res)
}

func (a *API) updateEpic(c *echo.Context) error {
	var req domain.ChangeUpdateEpicRequest
	if err := c.Bind(&req); err != nil {
		return apperror.InvalidPayload(err, "invalid change epic payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return apperror.HTTP(apperror.Validation(v.Errors, apperror.ErrChangeInvalidInput))
	}
	err := a.s.UpdateEpic(c.Request().Context(), req)
	if err != nil {
		return apperror.HTTP(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (a *API) updateChangeTypes(c *echo.Context) error {
	var req domain.ChangeUpdateChangeTypesRequest
	if err := c.Bind(&req); err != nil {
		return apperror.InvalidPayload(err, "invalid change types payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return apperror.HTTP(apperror.Validation(v.Errors, apperror.ErrChangeInvalidInput))
	}
	err := a.s.UpdateChangeTypes(c.Request().Context(), req)
	if err != nil {
		return apperror.HTTP(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (a *API) updateTitle(c *echo.Context) error {
	var req domain.ChangeUpdateTitleRequest
	if err := c.Bind(&req); err != nil {
		return apperror.InvalidPayload(err, "invalid change title payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return apperror.HTTP(apperror.Validation(v.Errors, apperror.ErrChangeInvalidInput))
	}
	err := a.s.UpdateTitle(c.Request().Context(), req)
	if err != nil {
		return apperror.HTTP(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (a *API) updateBrief(c *echo.Context) error {
	var req domain.ChangeUpdateBriefRequest
	if err := c.Bind(&req); err != nil {
		return apperror.InvalidPayload(err, "invalid change brief payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return apperror.HTTP(apperror.Validation(v.Errors, apperror.ErrChangeInvalidInput))
	}
	err := a.s.UpdateBrief(c.Request().Context(), req)
	if err != nil {
		return apperror.HTTP(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (a *API) updateSpec(c *echo.Context) error {
	var req domain.ChangeUpdateSpecRequest
	if err := c.Bind(&req); err != nil {
		return apperror.InvalidPayload(err, "invalid change spec payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return apperror.HTTP(apperror.Validation(v.Errors, apperror.ErrChangeInvalidInput))
	}
	err := a.s.UpdateSpec(c.Request().Context(), req)
	if err != nil {
		return apperror.HTTP(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (a *API) updatePR(c *echo.Context) error {
	var req domain.ChangeUpdatePRRequest
	if err := c.Bind(&req); err != nil {
		return apperror.InvalidPayload(err, "invalid change pr payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return apperror.HTTP(apperror.Validation(v.Errors, apperror.ErrChangeInvalidInput))
	}
	err := a.s.UpdatePR(c.Request().Context(), req)
	if err != nil {
		return apperror.HTTP(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (a *API) updatePRUrl(c *echo.Context) error {
	var req domain.ChangeUpdatePRUrlRequest
	if err := c.Bind(&req); err != nil {
		return apperror.InvalidPayload(err, "invalid change pr url payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return apperror.HTTP(apperror.Validation(v.Errors, apperror.ErrChangeInvalidInput))
	}
	err := a.s.UpdatePRUrl(c.Request().Context(), req)
	if err != nil {
		return apperror.HTTP(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (a *API) updatePhase(c *echo.Context) error {
	var req domain.ChangeUpdatePhaseRequest
	if err := c.Bind(&req); err != nil {
		return apperror.InvalidPayload(err, "invalid change phase payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return apperror.HTTP(apperror.Validation(v.Errors, apperror.ErrChangeInvalidInput))
	}
	err := a.s.UpdatePhase(c.Request().Context(), req)
	if err != nil {
		return apperror.HTTP(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (a *API) updateOpen(c *echo.Context) error {
	var req domain.ChangeUpdateOpenRequest
	if err := c.Bind(&req); err != nil {
		return apperror.InvalidPayload(err, "invalid change open payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return apperror.HTTP(apperror.Validation(v.Errors, apperror.ErrChangeInvalidInput))
	}
	err := a.s.UpdateOpen(c.Request().Context(), req)
	if err != nil {
		return apperror.HTTP(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (a *API) deleteChange(c *echo.Context) error {
	var req domain.ChangeIDRequest
	if err := c.Bind(&req); err != nil {
		return apperror.InvalidPayload(err, "invalid change delete payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return apperror.HTTP(apperror.Validation(v.Errors, apperror.ErrChangeInvalidInput))
	}
	if err := a.s.DeleteChange(c.Request().Context(), req); err != nil {
		return apperror.HTTP(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (a *API) documents(c *echo.Context) error {
	var req domain.ChangeIDRequest
	if err := c.Bind(&req); err != nil {
		return apperror.InvalidPayload(err, "invalid change documents payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return apperror.HTTP(apperror.Validation(v.Errors, apperror.ErrChangeInvalidInput))
	}
	res, err := a.s.Documents(c.Request().Context(), req)
	if err != nil {
		return apperror.HTTP(err)
	}
	return c.JSON(http.StatusOK, &res)
}

func (a *API) setDocument(c *echo.Context) error {
	var req domain.ChangeDocumentSetRequest
	if err := c.Bind(&req); err != nil {
		return apperror.InvalidPayload(err, "invalid change document payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return apperror.HTTP(apperror.Validation(v.Errors, apperror.ErrChangeInvalidInput))
	}
	if err := a.s.SetDocument(c.Request().Context(), req); err != nil {
		return apperror.HTTP(err)
	}
	return c.NoContent(http.StatusNoContent)
}
