package domain

import "time"

type (
	// TestCase defines TestCase values.
	TestCase struct {
		ID        int       `json:"id"`
		Scenario  string    `json:"scenario"`
		Done      bool      `json:"done"`
		ChangeID  int       `json:"change_id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	}

	// TestCaseListRequest defines TestCaseListRequest values.
	TestCaseListRequest struct {
		ChangeID int `json:"change_id" validate:"required|min:1"`
	}

	// TestCaseIDRequest defines TestCaseIDRequest values.
	TestCaseIDRequest struct {
		ID int `json:"id" validate:"required|min:1"`
	}

	// TestCaseCreateRequest defines TestCaseCreateRequest values.
	TestCaseCreateRequest struct {
		Scenario string `json:"scenario" validate:"required"`
		ChangeID int    `json:"change_id" validate:"required|min:1"`
	}

	// TestCaseUpdateRequest defines TestCaseUpdateRequest values.
	TestCaseUpdateRequest struct {
		ID       int    `json:"id" validate:"required|min:1"`
		Scenario string `json:"scenario" validate:"required"`
	}

	// TestCaseUpdateDoneRequest defines TestCaseUpdateDoneRequest values.
	TestCaseUpdateDoneRequest struct {
		ID   int  `json:"id" validate:"required|min:1"`
		Done bool `json:"done"`
	}
)
