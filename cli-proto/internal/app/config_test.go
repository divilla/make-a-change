package app

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfigInitializesMissingConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cli-proto", ".config", "config.json")
	cfg, saved, err := loadConfig(path)
	require.NoError(t, err)
	assert.True(t, saved)
	assert.Equal(t, defaultBackendURL, cfg.BackendURL)

	loaded, saved, err := loadConfig(path)
	require.NoError(t, err)
	assert.False(t, saved)
	assert.Equal(t, defaultBackendURL, loaded.BackendURL)
}
