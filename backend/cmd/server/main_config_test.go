package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMainConfigurationAndFlagOrdering(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "server")
	buildCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	build := exec.CommandContext(buildCtx, "go", "build", "-o", binary, ".")
	output, err := build.CombinedOutput()
	require.NoError(t, err, "%s", output)

	for _, tc := range []struct {
		name, source, dbEnv, portEnv string
		args                         []string
		want                         string
		status                       int
	}{
		{name: "missing config before help", args: []string{"-h"}, want: "panic: configuration:", status: 2},
		{name: "missing config before invalid flag", args: []string{"-unknown"}, want: "panic: configuration:", status: 2},
		{name: "valid config allows help", source: "{}", args: []string{"-h"}, want: "Usage of ", status: 0},
		{name: "valid config allows flag error", source: "{}", args: []string{"-unknown"}, want: "flag provided but not defined", status: 2},
		{name: "omitted db flag retains file", source: "connection_string: ':file-db'", want: "`:file-db`", status: 1},
		{name: "empty db flag retains environment", source: "connection_string: ':file-db'", dbEnv: ":env-db", args: []string{"-db="}, want: "`:env-db`", status: 1},
		{name: "empty db flag retains file", source: "connection_string: ':file-db'", args: []string{"-db="}, want: "`:file-db`", status: 1},
		{name: "omitted port flag retains file", source: "port: file-port", want: "file-port", status: 1},
		{name: "empty port flag retains environment", source: "port: file-port", portEnv: "env-port", args: []string{"-port="}, want: "env-port", status: 1},
		{name: "empty port flag retains file", source: "port: file-port", args: []string{"-port="}, want: "file-port", status: 1},
		{
			name:   "nonempty flags override file and environment",
			source: "connection_string: ':file-db'\nport: file-port", dbEnv: ":env-db", portEnv: "env-port",
			args: []string{"-db=postgresql://localhost:1/postgres?sslmode=disable", "-port=flag-port"},
			want: "flag-port", status: 1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if tc.source != "" {
				require.NoError(t, os.Mkdir(filepath.Join(dir, "config"), 0o700))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "config/dev.yaml"), []byte(tc.source), 0o600))
			}
			t.Setenv("DATABASE_URL", tc.dbEnv)
			t.Setenv("PORT", tc.portEnv)
			t.Setenv("CORS_ORIGINS", "")
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, binary, tc.args...)
			command.Dir = dir
			var stdout, stderr bytes.Buffer
			command.Stdout, command.Stderr = &stdout, &stderr
			err := command.Run()
			require.NoError(t, ctx.Err(), "ordinary startup must terminate without a database connection")
			if tc.status == 0 {
				require.NoError(t, err)
			} else {
				var exit *exec.ExitError
				require.ErrorAs(t, err, &exit)
				require.Equal(t, tc.status, exit.ExitCode())
			}
			require.Empty(t, stdout.String(), "early startup diagnostics stay on stderr")
			require.Contains(t, stderr.String(), tc.want)
			if tc.source == "" {
				require.NotContains(t, stderr.String(), "Usage of ", "config failure precedes help")
				require.NotContains(t, stderr.String(), "flag provided but not defined", "config failure precedes flag validation")
			}
		})
	}
}
