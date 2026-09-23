package health_test

import (
	"aipm/api-tests/shared"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

type healthResponse struct {
	Status   string `json:"status"`
	API      string `json:"api"`
	Database string `json:"database"`
}

func TestHealthEndpointsAreGet(t *testing.T) {
	client := shared.NewClient(t)

	for _, path := range []string{"/api/v1/health", "/api/health"} {
		t.Run(path, func(t *testing.T) {
			var res healthResponse
			status := client.Get(t, path, &res)

			assert.Equal(t, http.StatusOK, status)
			assert.Equal(t, "ok", res.API)
			assert.NotEmpty(t, res.Database)
		})
	}
}
