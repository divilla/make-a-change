// Package error owns backend error semantics and external error interpretation.
package error

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/labstack/echo/v5"
)

// Module errors remain distinct even where their HTTP statuses coincide.
var (
	ErrProjectInvalidInput    = errors.New("invalid project input")
	ErrProjectNotFound        = errors.New("project not found")
	ErrProjectHasChanges      = errors.New("project has dependencies")
	ErrProjectConfigNotFound  = errors.New("project configuration not found")
	ErrEpicInvalidInput       = errors.New("invalid epic input")
	ErrEpicNotFound           = errors.New("epic not found")
	ErrEpicHasChanges         = errors.New("epic has changes")
	ErrChangeInvalidInput     = errors.New("invalid change input")
	ErrChangeInvalidReference = errors.New("invalid change reference")
	ErrChangeNotFound         = errors.New("change not found")
	ErrTestCaseInvalidInput   = errors.New("invalid test case input")
	ErrTestCaseNotFound       = errors.New("test case not found")
)

// Wrap adds diagnostic context without losing the cause. No context means no wrapper.
func Wrap(err error, context string) error {
	if err == nil || context == "" {
		return err
	}
	return fmt.Errorf("%s: %w", context, err)
}

// Database maps only the caller's existing no-row and foreign-key contracts.
// A nil mapping leaves that database condition uninterpreted.
func Database(err, missing, foreignKey error) error {
	if err == nil {
		return nil
	}
	var semantic error
	var pgErr *pgconn.PgError
	switch {
	case missing != nil && errors.Is(err, pgx.ErrNoRows):
		semantic = missing
	case foreignKey != nil && errors.As(err, &pgErr) && pgErr.Code == "23503":
		semantic = foreignKey
	}
	if semantic == nil || errors.Is(err, semantic) {
		return err
	}
	return fmt.Errorf("%w: %w", semantic, err)
}

// InvalidPayload retains binding failures while exposing the operation's safe message.
func InvalidPayload(err error, message string) error {
	return echo.NewHTTPError(http.StatusBadRequest, message).Wrap(err)
}

// Interpret returns the public status and message, masking unexpected internal details.
// A nil error has no error response.
func Interpret(err error) (int, string) {
	if err == nil {
		return 0, ""
	}
	for _, contract := range []struct {
		err     error
		status  int
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
	} {
		if errors.Is(err, contract.err) {
			return contract.status, contract.message
		}
	}
	status := echo.StatusCode(err)
	if status < 400 || status >= 500 {
		return 500, http.StatusText(500)
	}
	var he *echo.HTTPError
	if errors.As(err, &he) && he.Message != "" {
		return status, he.Message
	}
	return status, http.StatusText(status)
}

// HTTP translates a handler error, preserving its complete cause chain.
func HTTP(err error) error {
	if err == nil {
		return nil
	}
	code, message := Interpret(err)
	var he *echo.HTTPError
	if errors.As(err, &he) && he.Code == code && he.Message == message {
		return err
	}
	return echo.NewHTTPError(code, message).Wrap(err)
}

// ServerShutdown accepts the normal net/http closure after an initiated shutdown.
func ServerShutdown(err error) error {
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return Wrap(err, "serve")
}
