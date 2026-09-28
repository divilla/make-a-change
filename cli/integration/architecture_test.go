package integration_test

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func boundaryViolations(root string) ([]string, error) {
	module, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return nil, err
	}
	var prefix string
	for _, line := range strings.Split(string(module), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "module" {
			prefix = strings.Trim(fields[1], `"`) + "/"
		}
	}
	if prefix == "" {
		return nil, fmt.Errorf("missing module directive")
	}
	features := map[string]bool{
		"agent": true, "changes": true, "epics": true, "help": true,
		"projects": true, "testcases": true, "documents": true, "configurations": true, "configs": true, "health": true,
	}
	shared := map[string]bool{"dto": true, "navigation": true, "styles": true, "ui": true}
	var violations []string
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if strings.HasPrefix(entry.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		parts := strings.Split(filepath.ToSlash(relative), "/")
		if len(parts) < 3 || (parts[0] != "internal" && parts[0] != "pkg") || filepath.Ext(path) != ".go" {
			return nil
		}
		source := parts[1]
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, spec := range parsed.Imports {
			name, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				return err
			}
			if !strings.HasPrefix(name, prefix) {
				continue
			}
			target := strings.Split(strings.TrimPrefix(name, prefix), "/")
			if len(target) < 2 {
				continue
			}
			upper := target[0] == "internal" && (target[1] == "app" || features[target[1]])
			adapter := target[0] == "pkg" || (target[0] == "internal" && !shared[target[1]] && !features[target[1]] && target[1] != "app")
			forbidden := false
			switch {
			case parts[0] == "pkg":
				forbidden = upper
			case features[source]:
				forbidden = adapter || (upper && target[1] != source)
			case shared[source]:
				forbidden = upper || adapter
			case source != "app": // internal adapters
				forbidden = upper
			}
			if forbidden {
				violations = append(violations, relative+" imports "+name)
			}
		}
		return nil
	})
	return violations, err
}

func TestCLIPackageBoundaries(t *testing.T) {
	violations, err := boundaryViolations(filepath.Join(repositoryRoot(t), "cli"))
	require.NoError(t, err)
	require.Empty(t, violations, strings.Join(violations, "\n"))
}

func TestCLIPackageBoundariesFixtures(t *testing.T) {
	for _, tt := range []struct {
		source, target string
		forbidden      bool
	}{
		{"internal/changes", "internal/app", true},
		{"internal/changes", "internal/projects", true},
		{"internal/changes", "internal/testcases", true},
		{"internal/testcases", "internal/app", true},
		{"internal/documents", "internal/app", true},
		{"internal/documents", "pkg/client", true},
		{"internal/documents", "internal/dto", false},
		{"internal/testcases", "pkg/client", true},
		{"internal/changes", "pkg/client", true},
		{"internal/changes", "internal/config", true},
		{"internal/dto", "internal/changes", true},
		{"internal/ui", "internal/app", true},
		{"internal/navigation", "pkg/client", true},
		{"pkg/client", "internal/app", true},
		{"pkg/client", "internal/projects", true},
		{"internal/editor", "internal/changes", true},
		{"internal/app", "pkg/client", false},
		{"internal/changes", "internal/dto", false},
		{"internal/changes", "internal/changes/detail", false},
		{"internal/ui", "internal/styles", false},
		{"pkg/client", "internal/dto", false},
	} {
		t.Run(tt.source+"_"+tt.target, func(t *testing.T) {
			root := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("module fixture.example/cli\n"), 0o600))
			dir := filepath.Join(root, tt.source)
			require.NoError(t, os.MkdirAll(dir, 0o700))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "edge_test.go"), []byte("package fixture\nimport _ \"fixture.example/cli/"+tt.target+"\"\n"), 0o600))
			violations, err := boundaryViolations(root)
			require.NoError(t, err)
			require.Equal(t, tt.forbidden, len(violations) > 0)
		})
	}
}
