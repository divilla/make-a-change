package integration_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCLIPackageBoundaries(t *testing.T) {
	root := repositoryRoot(t)
	internalRoot := filepath.Join(root, "cli", "internal")
	featurePackages := map[string]bool{
		"agent": true, "changes": true, "epics": true,
		"help": true, "projects": true, "testcases": true,
	}
	sharedPackages := map[string]bool{
		"dto": true, "navigation": true, "styles": true, "ui": true,
	}

	err := filepath.WalkDir(internalRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		require.NoError(t, walkErr)
		if entry.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}

		relative, err := filepath.Rel(internalRoot, path)
		require.NoError(t, err)
		sourcePackage := strings.Split(relative, string(filepath.Separator))[0]
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		require.NoError(t, err)

		for _, importSpec := range parsed.Imports {
			importPath, err := strconv.Unquote(importSpec.Path.Value)
			require.NoError(t, err)
			const prefix = "mch/internal/"
			if !strings.HasPrefix(importPath, prefix) {
				continue
			}
			targetPackage := strings.Split(strings.TrimPrefix(importPath, prefix), "/")[0]

			if featurePackages[sourcePackage] && targetPackage == "app" {
				assert.Failf(t, "feature imports app", "%s imports %s", relative, importPath)
			}
			if sharedPackages[sourcePackage] && (targetPackage == "app" || featurePackages[targetPackage]) {
				assert.Failf(t, "shared package imports an upper layer", "%s imports %s", relative, importPath)
			}
		}
		return nil
	})
	require.NoError(t, err)
}
