// Package testcases owns testcase validation, mutations, refreshes and feedback.
package testcases

import (
	"cli/internal/dto"
	"context"
)

// API is the small backend capability needed by testcase operations.
type API interface {
	ListTestCases(context.Context, int) ([]dto.TestCase, error)
	CreateTestCase(context.Context, int, string) (int, error)
	UpdateTestCase(context.Context, int, string) error
	UpdateTestCaseDone(context.Context, int, bool) error
	DeleteTestCase(context.Context, int) error
	GetChange(context.Context, int) (dto.Change, error)
}
