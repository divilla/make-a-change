package change

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChangeArchitectureAndP4Boundary(t *testing.T) {
	for _, path := range []string{"repo.go", "service.go", "api.go"} {
		source, err := os.ReadFile(path)
		require.NoError(t, err)
		for _, obsolete := range []string{"domain.Change,", "domain.Change{}", "finishMutation", "recalculate", "change_history", "test_case", "AvailableChangeTypes", "Begin("} {
			require.NotContains(t, string(source), obsolete, path)
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, source, 0)
		require.NoError(t, err)
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			for _, name := range []string{"Begin", "BeginTx", "Commit", "Rollback"} {
				require.NotEqual(t, name, sel.Sel.Name, path)
			}
			return true
		})
	}
	source, err := os.ReadFile("../testcase/service.go")
	require.NoError(t, err)
	require.NotContains(t, string(source), "RenderMutation")
	source, err = os.ReadFile("../testcase/repo.go")
	require.NoError(t, err)
	require.NotContains(t, string(source), "domain.Change")
	source, err = os.ReadFile("service.go")
	require.NoError(t, err)
	require.False(t, strings.Contains(string(source), "RenderChange("))
}
