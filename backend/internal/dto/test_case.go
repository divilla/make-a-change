package dto

import "time"

type (
	// TestCase defines TestCase values.
	TestCase struct {
		ID       int       `json:"id"`
		Version  int16     `json:"version"`
		Scenario string    `json:"scenario"`
		Done     bool      `json:"done"`
		ChangeID int       `json:"change_id"`
		Created  time.Time `json:"created"`
		Modified time.Time `json:"modified"`
	}

	// TestCaseListRequest defines TestCaseListRequest values.
	TestCaseListRequest struct {
		ChangeID int `json:"change_id"`
	}

	// TestCaseIDRequest defines TestCaseIDRequest values.
	TestCaseIDRequest struct {
		ID int `json:"id"`
	}

	// TestCaseCreateRequest defines TestCaseCreateRequest values.
	TestCaseCreateRequest struct {
		Scenario string `json:"scenario"`
		ChangeID int    `json:"change_id"`
	}

	// TestCaseUpdateRequest defines TestCaseUpdateRequest values.
	TestCaseUpdateRequest struct {
		ID       int    `json:"id"`
		Scenario string `json:"scenario"`
	}

	// TestCaseUpdateDoneRequest defines TestCaseUpdateDoneRequest values.
	TestCaseUpdateDoneRequest struct {
		ID   int  `json:"id"`
		Done bool `json:"done"`
	}

	// TestCaseUpdateChangeRequest defines TestCaseUpdateChangeRequest values.
	TestCaseUpdateChangeRequest struct {
		ID       int `json:"id"`
		ChangeID int `json:"change_id"`
	}

	// TestCaseMutationResponse defines TestCaseMutationResponse values.
	TestCaseMutationResponse struct {
		TestCase  *TestCase  `json:"test_case,omitempty"`
		Change    Change     `json:"change"`
		TestCases []TestCase `json:"test_cases"`
	}
)
