// Package doc provides shared docs for projects, epics and changes.
package doc

import (
	"mch_api/internal/app"
	"mch_api/internal/domain"
	"net/http"

	"github.com/gookit/validate/v2"
	"github.com/labstack/echo/v5"
)

// API handles doc requests.
type API struct {
	g *echo.Group
	s *Service
}

// NewAPI registers explicit doc operations.
func NewAPI(e *echo.Echo, s *Service) *API {
	a := &API{g: e.Group("/api").Group("/v1").Group("/doc"), s: s}
	a.g.POST("/list", a.list)
	a.g.POST("/list-active", a.listActive)
	a.g.POST("/details", a.details)
	a.g.POST("/insert", a.insert)
	a.g.POST("/comment-list", a.commentList)
	a.g.POST("/comment-insert", a.commentInsert)
	a.g.POST("/comment-update", a.commentUpdate)
	a.g.POST("/delete", a.delete)
	return a
}

func (a *API) list(c *echo.Context) error {
	var req domain.DocListRequest
	if err := c.Bind(&req); err != nil {
		return app.PayloadError(err, "invalid doc list payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return app.HTTPError(app.ValidationError(v.Errors, app.ErrDocInvalidInput))
	}
	result, err := a.s.List(c.Request().Context(), req)
	if err != nil {
		return app.HTTPError(err)
	}
	return c.JSON(http.StatusOK, result)
}

func (a *API) listActive(c *echo.Context) error {
	var req domain.DocListRequest
	if err := c.Bind(&req); err != nil {
		return app.PayloadError(err, "invalid doc active list payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return app.HTTPError(app.ValidationError(v.Errors, app.ErrDocInvalidInput))
	}
	result, err := a.s.ListActive(c.Request().Context(), req)
	if err != nil {
		return app.HTTPError(err)
	}
	return c.JSON(http.StatusOK, result)
}

func (a *API) details(c *echo.Context) error {
	var req domain.DocIDRequest
	if err := c.Bind(&req); err != nil {
		return app.PayloadError(err, "invalid doc details payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return app.HTTPError(app.ValidationError(v.Errors, app.ErrDocInvalidInput))
	}
	result, err := a.s.Details(c.Request().Context(), req)
	if err != nil {
		return app.HTTPError(err)
	}
	return c.JSON(http.StatusOK, result)
}

func (a *API) insert(c *echo.Context) error {
	var req domain.DocInsertRequest
	if err := c.Bind(&req); err != nil {
		return app.PayloadError(err, "invalid doc insert payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return app.HTTPError(app.ValidationError(v.Errors, app.ErrDocInvalidInput))
	}
	result, err := a.s.Insert(c.Request().Context(), req)
	if err != nil {
		return app.HTTPError(err)
	}
	return c.JSON(http.StatusCreated, result)
}

func (a *API) commentList(c *echo.Context) error {
	var req domain.DocListRequest
	if err := c.Bind(&req); err != nil {
		return app.PayloadError(err, "invalid doc comment list payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return app.HTTPError(app.ValidationError(v.Errors, app.ErrDocInvalidInput))
	}
	result, err := a.s.CommentList(c.Request().Context(), req)
	if err != nil {
		return app.HTTPError(err)
	}
	return c.JSON(200, result)
}

func (a *API) commentInsert(c *echo.Context) error {
	var req domain.DocCommentInsertRequest
	if err := c.Bind(&req); err != nil {
		return app.PayloadError(err, "invalid doc comment insert payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return app.HTTPError(app.ValidationError(v.Errors, app.ErrDocInvalidInput))
	}
	result, err := a.s.CommentInsert(c.Request().Context(), req)
	if err != nil {
		return app.HTTPError(err)
	}
	return c.JSON(201, result)
}

func (a *API) commentUpdate(c *echo.Context) error {
	var req domain.DocCommentUpdateRequest
	if err := c.Bind(&req); err != nil {
		return app.PayloadError(err, "invalid doc comment update payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return app.HTTPError(app.ValidationError(v.Errors, app.ErrDocInvalidInput))
	}
	if err := a.s.CommentUpdate(c.Request().Context(), req); err != nil {
		return app.HTTPError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (a *API) delete(c *echo.Context) error {
	var req domain.DocIDRequest
	if err := c.Bind(&req); err != nil {
		return app.PayloadError(err, "invalid doc delete payload")
	}
	if v := validate.Struct(req); !v.Validate() {
		return app.HTTPError(app.ValidationError(v.Errors, app.ErrDocInvalidInput))
	}
	if err := a.s.Delete(c.Request().Context(), req); err != nil {
		return app.HTTPError(err)
	}
	return c.NoContent(http.StatusNoContent)
}
