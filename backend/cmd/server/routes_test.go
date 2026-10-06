package main

import (
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestAPIConstructorRouteInventory(t *testing.T) {
	e, err := newRouter(nil, []string{"https://allowed.example"}, zerolog.Nop())
	require.NoError(t, err)

	var registered []string
	for _, route := range e.Router().Routes() {
		registered = append(registered, route.Method+" "+route.Path)
	}
	require.ElementsMatch(t, []string{
		"GET /api/health",
		"GET /api/v1/health",
		"POST /api/v1/project/list",
		"POST /api/v1/project/details",
		"POST /api/v1/project/config",
		"POST /api/v1/project/create",
		"POST /api/v1/project/update",
		"POST /api/v1/project/delete",
		"POST /api/v1/epic/list",
		"POST /api/v1/epic/details",
		"POST /api/v1/epic/create",
		"POST /api/v1/epic/update",
		"POST /api/v1/epic/update-active",
		"POST /api/v1/epic/delete",
		"POST /api/v1/change/update-after-change",
		"POST /api/v1/change/list",
		"POST /api/v1/change/details",
		"POST /api/v1/change/create",
		"POST /api/v1/change/update-epic",
		"POST /api/v1/change/update-phase",
		"POST /api/v1/change/update-active",
		"POST /api/v1/change/update-types",
		"POST /api/v1/change/update-title",
		"POST /api/v1/change/update-slug",
		"POST /api/v1/change/update-pr-url",
		"POST /api/v1/change/delete",
		"POST /api/v1/test-case/list",
		"POST /api/v1/test-case/create",
		"POST /api/v1/test-case/update",
		"POST /api/v1/test-case/update-done",
		"POST /api/v1/test-case/delete",
		"POST /api/v1/doc/list",
		"POST /api/v1/doc/list-active",
		"POST /api/v1/doc/active-set",
		"POST /api/v1/doc/details",
		"POST /api/v1/doc/insert",
		"POST /api/v1/doc/comment-list",
		"POST /api/v1/doc/comment-insert",
		"POST /api/v1/doc/comment-update",
		"POST /api/v1/doc/comment-undelete",
		"POST /api/v1/doc/delete",
		"POST /api/v1/config/list",
		"POST /api/v1/config/details",
		"POST /api/v1/config/insert",
		"POST /api/v1/config/update",
		"POST /api/v1/config/delete",
	}, registered)
}
