package testcase

import (
	"mch_api/internal/domain"
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCurrentTestcaseArchitecture(t *testing.T) {
	typ := reflect.TypeOf(domain.TestCase{})
	require.Equal(t, 6, typ.NumField())
	fields := []string{}
	for i := 0; i < typ.NumField(); i++ {
		fields = append(fields, typ.Field(i).Tag.Get("json"))
	}
	require.ElementsMatch(t, []string{"id", "change_id", "scenario", "done", "created_at", "updated_at"}, fields)
	for _, path := range []string{"repo.go", "service.go", "api.go", "../doc/render.go", "../domain/test_case.go", "../domain/change.go", "../../cmd/server/main.go"} {
		source, err := os.ReadFile(path)
		require.NoError(t, err)
		for _, old := range []string{"public.test_case", "testcase_history", "test_case_history", "sp_testcase", "sp_test_case", "fn_testcase", "fn_test_case", "Begin(", "BeginTx(", "Commit(", "Rollback(", "finishMutation", "details(", "scanChange(", "RenderMutation", "RenderChange(", "TestCaseMutationResponse", "domain.Change{"} {
			require.NotContains(t, string(source), old, path)
		}
	}
	source, err := os.ReadFile("service.go")
	require.NoError(t, err)
	for _, dependency := range []string{"internal/change", "pgx", "echo", "validate"} {
		require.NotContains(t, string(source), dependency)
	}
}
