package config

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func configurationDirectory(t *testing.T) {
	t.Helper()
	t.Chdir(t.TempDir())
	require.NoError(t, os.Mkdir("config", 0o700))
	for _, key := range []string{"DATABASE_URL", "PORT", "CORS_ORIGINS"} {
		t.Setenv(key, "")
	}
}

func writeConfiguration(t *testing.T, source string) {
	t.Helper()
	require.NoError(t, os.WriteFile("config/dev.yaml", []byte(source), 0o600))
}

func TestConfigurationIndependentLoads(t *testing.T) {
	configurationDirectory(t)
	writeConfiguration(t, "connection_string: file-db\nport: '1234'\ncors_origins: http://file\n")
	first := New()
	require.Equal(t, Config{"file-db", "1234", "http://file"}, *first, "mapstructure fields load from the fixed YAML path")

	writeConfiguration(t, "port: '5678'\n")
	second := New()
	require.NotSame(t, first, second, "each load returns a new owner")
	require.Equal(t, Config{"postgresql://localhost:5432/postgres", "5678", "http://localhost:8000"}, *second, "omitted fields default instead of retaining file A")
	require.Equal(t, Config{"file-db", "1234", "http://file"}, *first, "file B cannot change file A's object")

	*first = Config{"owner-db", ":9000", "http://owner"}
	require.Equal(t, Config{"postgresql://localhost:5432/postgres", "5678", "http://localhost:8000"}, *second, "owner mutation cannot affect another object")
	third := New()
	require.NotSame(t, second, third, "identical values still have distinct ownership")
	require.Equal(t, *second, *third, "owner mutation cannot affect future loads")

	writeConfiguration(t, "{}\n")
	fourth := New()
	require.Equal(t, Config{"postgresql://localhost:5432/postgres", "8080", "http://localhost:8000"}, *fourth, "empty file defaults every field after earlier loads")
}

func TestConfigurationPanicCausesAndRecovery(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		pathError    bool
	}{
		{name: "missing file", pathError: true},
		{name: "unreadable file is a directory", pathError: true},
		{name: "malformed YAML", source: "["},
		{name: "decode failure after other fields", source: "connection_string: partial-db\ncors_origins: http://partial\nport: [one, two]\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			configurationDirectory(t)
			writeConfiguration(t, "connection_string: earlier-db\nport: '1234'\ncors_origins: http://earlier\n")
			earlier := New()
			before := *earlier
			require.NoError(t, os.Remove("config/dev.yaml"))
			switch tc.name {
			case "missing file":
			case "unreadable file is a directory":
				require.NoError(t, os.Mkdir("config/dev.yaml", 0o700))
			default:
				writeConfiguration(t, tc.source)
			}
			var recovered any
			func() { defer func() { recovered = recover() }(); _ = New() }()
			err, ok := recovered.(error)
			require.True(t, ok, "panic must retain an error: %v", recovered)
			cause := errors.Unwrap(err)
			require.NotNil(t, cause, "configuration panic preserves its cause")
			require.Equal(t, "configuration: "+cause.Error(), err.Error(), "context wraps the underlying error")
			if tc.pathError {
				var pathErr *os.PathError
				require.ErrorAs(t, err, &pathErr)
				require.Equal(t, "config/dev.yaml", pathErr.Path)
				if tc.name == "missing file" {
					require.ErrorIs(t, err, os.ErrNotExist)
				}
			}
			require.Equal(t, before, *earlier, "failed load cannot corrupt a returned object")
			if tc.name == "unreadable file is a directory" {
				require.NoError(t, os.Remove("config/dev.yaml"))
			}
			writeConfiguration(t, "{}\n")
			after := New()
			require.NotSame(t, earlier, after)
			require.Equal(t, Config{"postgresql://localhost:5432/postgres", "8080", "http://localhost:8000"}, *after, "failed decode cannot leak partial fields into the next load")
			require.Equal(t, before, *earlier, "successful recovery preserves earlier values")
		})
	}
}

func TestConfigurationEnvironmentPrecedence(t *testing.T) {
	for _, source := range []struct{ name, yaml string }{
		{"file", "connection_string: file-db\nport: '1234'\ncors_origins: http://file\n"},
		{"defaults", "{}\n"},
	} {
		t.Run(source.name, func(t *testing.T) {
			configurationDirectory(t)
			writeConfiguration(t, source.yaml)
			for _, key := range []string{"DATABASE_URL", "PORT", "CORS_ORIGINS"} {
				require.NoError(t, os.Unsetenv(key)) // t.Setenv above restores original process state.
			}
			want := Config{"file-db", "1234", "http://file"}
			if source.name == "defaults" {
				want = Config{"postgresql://localhost:5432/postgres", "8080", "http://localhost:8000"}
			}
			require.Equal(t, want, *New(), "unset overrides retain file/default values")
			for _, key := range []string{"DATABASE_URL", "PORT", "CORS_ORIGINS"} {
				t.Setenv(key, "")
			}
			require.Equal(t, want, *New(), "empty overrides retain file/default values")
			t.Setenv("DATABASE_URL", " env-db ")
			t.Setenv("PORT", " :4321 ")
			t.Setenv("CORS_ORIGINS", " http://one, ,http://two ")
			require.Equal(t, Config{" env-db ", " :4321 ", " http://one, ,http://two "}, *New(), "nonempty overrides apply last without trimming")
		})
	}
}

func TestConfigurationEnvironmentInterpolation(t *testing.T) {
	configurationDirectory(t)
	t.Setenv("MCH_R4_DB", "interpolated-db")
	t.Setenv("MCH_R4_PORT", "")
	t.Setenv("MCH_R4_ORIGIN", "http://interpolated")
	writeConfiguration(t, "connection_string: '${MCH_R4_DB}'\nport: '${MCH_R4_PORT|4567}'\ncors_origins: '${MCH_R4_ORIGIN|http://fallback}'\n")
	require.Equal(t, Config{"interpolated-db", "4567", "http://interpolated"}, *New(), "ParseEnv expands variables and fallback before defaults")
	t.Setenv("MCH_R4_DB", "")
	t.Setenv("MCH_R4_ORIGIN", "")
	require.Equal(t, Config{"postgresql://localhost:5432/postgres", "4567", "http://fallback"}, *New(), "empty interpolation receives application defaults; fallback remains a file value")
	t.Setenv("DATABASE_URL", "explicit-db")
	t.Setenv("PORT", "7890")
	t.Setenv("CORS_ORIGINS", "http://explicit")
	require.Equal(t, Config{"explicit-db", "7890", "http://explicit"}, *New(), "explicit environment overrides win over interpolation and fallback")
}

func TestConfigurationAccessors(t *testing.T) {
	var cfg Config
	for _, tc := range []struct{ port, want string }{
		{"8080", ":8080"}, {":9000", ":9000"}, {"", ":"}, {" 80 ", ": 80 "}, {"::80", "::80"},
	} {
		t.Run("address "+tc.port, func(t *testing.T) {
			cfg.Port = tc.port
			require.Equal(t, tc.want, cfg.Addr(), "owner mutation is reflected without validation or trimming")
		})
	}
	for _, tc := range []struct {
		name, source string
		want         []string
	}{
		{"empty", "", nil},
		{"all blank", " , \t,\n ", nil},
		{"ordered duplicates", " http://two, ,http://one, http://two ", []string{"http://two", "http://one", "http://two"}},
	} {
		t.Run("origins "+tc.name, func(t *testing.T) {
			cfg.CORSOrigins = tc.source
			require.Equal(t, tc.want, cfg.AllowedOrigins(), "trim and omit blanks while retaining order, duplicates and nil")
		})
	}
	other := Config{Port: "1234", CORSOrigins: "http://other"}
	require.Equal(t, ":1234", other.Addr(), "accessors read only their receiver")
	require.Equal(t, []string{"http://other"}, other.AllowedOrigins())
	origins := other.AllowedOrigins()
	origins[0] = "http://caller"
	require.Equal(t, []string{"http://other"}, other.AllowedOrigins(), "derived slices do not mutate the receiver")
}

func TestConfigurationConcurrentIndependence(t *testing.T) {
	configurationDirectory(t)
	writeConfiguration(t, "connection_string: concurrent-db\nport: '1234'\ncors_origins: http://concurrent\n")
	t.Setenv("PORT", "5678")
	const count = 32
	type result struct {
		cfg    *Config
		before Config
		owner  string
	}
	results := make(chan result, count)
	for i := range count {
		go func() {
			cfg := New()
			before := *cfg
			owner := fmt.Sprintf("owner-%d", i)
			*cfg = Config{owner, owner, owner}
			results <- result{cfg, before, owner}
		}()
	}
	// Collect every result before assertions, keeping cwd and environment stable
	// until all constructors and owner mutations have finished.
	collected := make([]result, count)
	for i := range collected {
		collected[i] = <-results
	}
	seen := make(map[*Config]bool)
	for _, result := range collected {
		require.Equal(t, Config{"concurrent-db", "5678", "http://concurrent"}, result.before, "concurrent loads see only the immutable fixture and environment")
		require.False(t, seen[result.cfg], "each goroutine owns a distinct config")
		seen[result.cfg] = true
		require.Equal(t, Config{result.owner, result.owner, result.owner}, *result.cfg, "other goroutines cannot change an owner's fields")
	}
	require.Equal(t, Config{"concurrent-db", "5678", "http://concurrent"}, *New(), "concurrent mutations cannot leak into later loads")
}
