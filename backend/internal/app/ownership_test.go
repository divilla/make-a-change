package app

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBackendErrorOwnership(t *testing.T) {
	root := filepath.Join("..", "..")
	for _, dir := range []string{"cmd", "internal", "pkg"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") || filepath.Dir(path) == filepath.Join(root, "internal", "app") {
				return nil
			}
			require.Empty(t, errorOwnershipViolations(t, path, nil))
			return nil
		})
		require.NoError(t, err)
	}
}

func errorOwnershipViolations(t *testing.T, path string, source any) []string {
	t.Helper()
	var violations []string
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, source, 0)
	require.NoError(t, err)
	imports := map[string]string{}
	for _, imp := range file.Imports {
		source, err := strconv.Unquote(imp.Path.Value)
		require.NoError(t, err)
		name := filepath.Base(source)
		// Echo's declared package name differs from its versioned path basename.
		if source == "github.com/labstack/echo/v5" {
			name = "echo"
		}
		if imp.Name != nil {
			name = imp.Name.Name
		}
		imports[name] = source
		if filepath.Base(path) == "service.go" {
			for _, forbidden := range []string{"github.com/jackc/pgx", "github.com/labstack/echo", "github.com/gookit/validate"} {
				if strings.HasPrefix(source, forbidden) {
					violations = append(violations, "service import in "+path)
				}
			}
		}
	}
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		pkg, ok := selector.X.(*ast.Ident)
		if !ok {
			return true
		}
		source := imports[pkg.Name]
		forbidden := source == "errors" || source == "fmt" && selector.Sel.Name == "Errorf" || strings.HasPrefix(source, "github.com/labstack/echo/") && (selector.Sel.Name == "NewHTTPError" || selector.Sel.Name == "StatusCode")
		if forbidden {
			violations = append(violations, "independent error handling at "+fset.Position(call.Pos()).String())
		}
		return true
	})
	return violations
}

func TestErrorOwnershipEchoCalls(t *testing.T) {
	for _, importName := range []string{"", "web"} {
		name := importName
		if name == "" {
			name = "echo"
		}
		for _, call := range []string{`NewHTTPError(400, "bad request")`, `StatusCode(err)`} {
			t.Run(name+"/"+call, func(t *testing.T) {
				source := "package fixture\nimport " + importName + ` "github.com/labstack/echo/v5"` +
					"\nfunc handler(err error) { _ = " + name + "." + call + " }"
				require.Equal(t, []string{"independent error handling at api.go:3:31"},
					errorOwnershipViolations(t, "api.go", source))
			})
		}
	}
}

func TestErrorOwnershipAllowsEchoSetupAndCentralErrors(t *testing.T) {
	source := `package fixture
import (
	"github.com/labstack/echo/v5"
	"mch_api/internal/app"
)
func handler() {
	_ = echo.New()
	_ = app.InvalidPayload(nil, "bad request")
}`
	require.Empty(t, errorOwnershipViolations(t, "api.go", source))
}
