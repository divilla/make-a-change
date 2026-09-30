package app

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/require"
)

func TestWrapAndShutdownCauses(t *testing.T) {
	cause := &pgconn.PgError{Code: "XX000", Message: "private detail"}
	require.Nil(t, Wrap(nil, "context"))
	require.Same(t, cause, Wrap(cause, ""))
	wrapped := Wrap(Wrap(cause, "scan"), "read")
	require.EqualError(t, wrapped, "read: scan: "+cause.Error())
	require.ErrorIs(t, wrapped, cause)
	var pgErr *pgconn.PgError
	require.ErrorAs(t, wrapped, &pgErr)
	require.Same(t, cause, pgErr)
	require.Same(t, wrapped, Wrap(wrapped, ""))
	require.Nil(t, ServerShutdown(nil))
	require.Nil(t, ServerShutdown(http.ErrServerClosed))
	require.Nil(t, ServerShutdown(Wrap(http.ErrServerClosed, "server")))
	require.ErrorIs(t, ServerShutdown(cause), cause)
}

func TestDatabaseMappingsAndCauses(t *testing.T) {
	fk := &pgconn.PgError{Code: "23503", Message: "private constraint"}
	other := &pgconn.PgError{Code: "23505"}
	for _, tc := range []struct {
		name                                 string
		cause, missing, foreignKey, semantic error
	}{
		{"nil", nil, ErrChangeNotFound, nil, nil},
		{"no rows", pgx.ErrNoRows, ErrChangeNotFound, nil, ErrChangeNotFound},
		{"wrapped no rows", Wrap(pgx.ErrNoRows, "query"), ErrProjectNotFound, nil, ErrProjectNotFound},
		{"epic missing", pgx.ErrNoRows, ErrEpicNotFound, nil, ErrEpicNotFound},
		{"testcase missing", pgx.ErrNoRows, ErrTestCaseNotFound, nil, ErrTestCaseNotFound},
		{"foreign key", fk, nil, ErrTestCaseNotFound, ErrTestCaseNotFound},
		{"nested foreign key", Wrap(Wrap(fk, "insert"), "transaction"), nil, ErrTestCaseNotFound, ErrTestCaseNotFound},
		{"unmapped rows", pgx.ErrNoRows, nil, nil, nil},
		{"unmapped constraint", fk, nil, nil, nil},
		{"other constraint", other, ErrChangeNotFound, ErrTestCaseNotFound, nil},
		{"unknown", errors.New("unknown"), ErrChangeNotFound, ErrTestCaseNotFound, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := DatabaseError(tc.cause, tc.missing, tc.foreignKey)
			require.ErrorIs(t, got, tc.cause)
			if tc.semantic != nil {
				require.ErrorIs(t, got, tc.semantic)
			} else {
				require.Equal(t, tc.cause, got)
			}
			require.Equal(t, got, DatabaseError(got, tc.missing, tc.foreignKey), "no repeated mapping")
			var pgErr *pgconn.PgError
			if errors.As(tc.cause, &pgErr) {
				var actual *pgconn.PgError
				require.ErrorAs(t, got, &actual)
				require.Same(t, pgErr, actual)
			}
		})
	}
}

func TestHTTPContractsAndCauses(t *testing.T) {
	for _, tc := range []struct {
		cause   error
		code    int
		message string
	}{
		{ErrProjectInvalidInput, 400, "invalid project payload"},
		{ErrProjectNotFound, 404, "project not found"},
		{ErrProjectConfigNotFound, 404, "project configuration not found"},
		{ErrProjectHasChanges, 409, "project has dependencies and cannot be deleted"},
		{ErrEpicInvalidInput, 400, "invalid epic payload"},
		{ErrEpicNotFound, 404, "epic not found"},
		{ErrEpicHasChanges, 409, "epic has changes and cannot be deleted"},
		{ErrChangeInvalidInput, 400, "invalid change payload"},
		{ErrChangeInvalidReference, 400, "invalid change reference"},
		{ErrChangeNotFound, 404, "change not found"},
		{ErrTestCaseInvalidInput, 400, "invalid test case payload"},
		{ErrTestCaseNotFound, 404, "test case not found"},
		{errors.New("secret SQL"), 500, "Internal Server Error"},
		{echo.NewHTTPError(500, "secret SQL"), 500, "Internal Server Error"},
		{echo.NewHTTPError(400, "operation message"), 400, "operation message"},
		{echo.NewHTTPError(400, ""), 400, "Bad Request"},
		{echo.ErrNotFound, 404, "Not Found"},
		{echo.ErrMethodNotAllowed, 405, "Method Not Allowed"},
	} {
		t.Run(tc.message+fmt.Sprint(tc.cause), func(t *testing.T) {
			for _, cause := range []error{tc.cause, Wrap(tc.cause, "outer")} {
				code, message := Interpret(cause)
				require.Equal(t, tc.code, code)
				require.Equal(t, tc.message, message)
				translated := HTTP(cause)
				var he *echo.HTTPError
				require.ErrorAs(t, translated, &he)
				require.Equal(t, tc.code, he.Code)
				require.Equal(t, tc.message, he.Message)
				require.ErrorIs(t, translated, cause)
				require.Same(t, translated, HTTP(translated))
			}
		})
	}
	code, message := Interpret(nil)
	require.Zero(t, code)
	require.Empty(t, message)
	require.Nil(t, HTTP(nil))
	require.NotErrorIs(t, ErrProjectNotFound, ErrChangeNotFound)
	external := &pgconn.PgError{Code: "23503"}
	translated := HTTP(DatabaseError(Wrap(external, "insert"), nil, ErrTestCaseNotFound))
	var actual *pgconn.PgError
	require.ErrorAs(t, translated, &actual)
	require.Same(t, external, actual)
}

func TestInvalidPayload(t *testing.T) {
	cause := errors.New("decoder details")
	for _, err := range []error{nil, cause, Wrap(cause, "bind")} {
		got := InvalidPayload(err, "invalid change details payload")
		code, message := Interpret(got)
		require.Equal(t, 400, code)
		require.Equal(t, "invalid change details payload", message)
		if err != nil {
			require.ErrorIs(t, got, err)
		}
	}
}

func TestChangeDatabaseContracts(t *testing.T) {
	for _, tc := range []struct {
		cause   *pgconn.PgError
		want    error
		status  int
		message string
	}{
		{&pgconn.PgError{Code: "23505", ConstraintName: "change_ref_uuid_idx"}, ErrChangeDuplicateUUID, 409, "change reference UUID already exists"},
		{&pgconn.PgError{Code: "23505", ConstraintName: "other"}, nil, 500, "Internal Server Error"},
		{&pgconn.PgError{Code: "23502", TableName: "change", ColumnName: "project_id"}, ErrProjectNotFound, 404, "project not found"},
		{&pgconn.PgError{Code: "23502", TableName: "change", ColumnName: "title"}, nil, 500, "Internal Server Error"},
		{&pgconn.PgError{Code: "23503"}, ErrProjectNotFound, 404, "project not found"},
	} {
		cause := Wrap(tc.cause, "nested")
		err := ChangeCreate(cause)
		require.ErrorIs(t, err, cause)
		require.ErrorIs(t, err, tc.cause)
		if tc.want != nil {
			require.ErrorIs(t, err, tc.want)
		}
		code, msg := Interpret(err)
		require.Equal(t, tc.status, code)
		require.Equal(t, tc.message, msg)
	}
	require.NoError(t, ChangeCreate(nil))
	cause := &pgconn.PgError{Code: "23503"}
	err := DatabaseError(cause, nil, ErrChangeHasTestCases)
	require.ErrorIs(t, err, cause)
	code, msg := Interpret(err)
	require.Equal(t, 409, code)
	require.Equal(t, "change has testcases and cannot be deleted", msg)
}

func TestValidationCauses(t *testing.T) {
	for _, input := range []string{"https://host/%zz", "https://[host"} {
		t.Run(input, func(t *testing.T) {
			_, cause := url.Parse(input)
			require.Error(t, cause)
			wrapped := Wrap(cause, "parser")
			for _, tc := range []struct {
				name            string
				cause, semantic error
			}{
				{"direct", cause, ErrChangeInvalidInput},
				{"wrapped cause", wrapped, ErrChangeInvalidInput},
				{"wrapped semantic", cause, Wrap(ErrChangeInvalidInput, "input")},
				{"nil cause", nil, ErrChangeInvalidInput},
				{"nil semantic", cause, nil},
				{"nil semantic wrapped", wrapped, nil},
				{"both nil", nil, nil},
			} {
				t.Run(tc.name, func(t *testing.T) {
					got := Validation(tc.cause, tc.semantic)
					var httpErr *echo.HTTPError
					require.False(t, errors.As(got, &httpErr))
					if tc.semantic != nil {
						require.ErrorIs(t, got, tc.semantic)
						require.ErrorIs(t, got, ErrChangeInvalidInput)
					}
					if tc.cause != nil {
						require.ErrorIs(t, got, tc.cause)
						require.ErrorIs(t, got, cause)
						var parser *url.Error
						require.ErrorAs(t, got, &parser)
						require.Same(t, cause, parser)
					}
					if tc.cause == nil {
						require.Equal(t, tc.semantic, got)
					}
					if tc.semantic == nil {
						require.Equal(t, tc.cause, got)
					}
				})
			}
		})
	}
}
