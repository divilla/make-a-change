package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	gconfig "github.com/gookit/config/v2"
	"github.com/stretchr/testify/require"
)

func TestConfigurationPanicCausesAndPrecedence(t *testing.T) {
	original := cfg
	t.Cleanup(func() { cfg = original; gconfig.Reset() })
	for _, tc := range []struct {
		name, source    string
		missing, panics bool
	}{
		{name: "missing file", missing: true, panics: true},
		{name: "invalid YAML", source: "[", panics: true},
		{name: "decode failure", source: "port: [one, two]\n", panics: true},
		{name: "valid", source: "port: '1234'\nconnection_string: file-db\ncors_origins: http://file\n"},
		{name: "defaults", source: "{}\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gconfig.Reset()
			cfg = Config{}
			dir := t.TempDir()
			t.Chdir(dir)
			require.NoError(t, os.Mkdir(filepath.Join(dir, "config"), 0o700))
			if !tc.missing {
				require.NoError(t, os.WriteFile(filepath.Join(dir, "config/dev.yaml"), []byte(tc.source), 0o600))
			}
			t.Setenv("DATABASE_URL", "")
			t.Setenv("PORT", "")
			t.Setenv("CORS_ORIGINS", "")
			if tc.panics {
				var recovered any
				func() { defer func() { recovered = recover() }(); New() }()
				err, ok := recovered.(error)
				require.True(t, ok, "panic must retain an error: %v", recovered)
				require.Contains(t, err.Error(), "configuration: ")
				require.NotNil(t, errors.Unwrap(err))
				if tc.missing {
					var pathErr *os.PathError
					require.ErrorAs(t, err, &pathErr)
				}
				return
			}
			New()
			if tc.name == "defaults" {
				require.Equal(t, "8080", Get().Port)
				require.Equal(t, "postgresql://localhost:5432/postgres", Get().ConnectionString)
			} else {
				require.Equal(t, "1234", Get().Port)
				require.Equal(t, "file-db", Get().ConnectionString)
			}
			t.Setenv("PORT", ":4321")
			t.Setenv("DATABASE_URL", "env-db")
			t.Setenv("CORS_ORIGINS", " http://one, ,http://two ")
			gconfig.Reset()
			cfg = Config{}
			New()
			require.Equal(t, ":4321", Get().Addr())
			require.Equal(t, "env-db", Get().ConnectionString)
			require.Equal(t, []string{"http://one", "http://two"}, Get().AllowedOrigins())
		})
	}
}
