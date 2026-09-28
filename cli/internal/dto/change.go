package dto

// ChangeView is the change row and detail data used by mch.
type ChangeView struct {
	ID            string
	RefUUID       string
	Ref           string
	Slug          string
	ProjectID     string
	EpicID        string
	EpicName      string
	ChangePhase   string
	ChangeTypes   []string
	Title         string
	Brief         string
	Spec          string
	PR            string
	AfterChangeID string
	PRUrl         string
	Open          bool
	Done          int64
	Total         int64
	Completed     int64
	Documents     []Document
	TestCases     []TestCase
	Created       string
	Modified      string
}

// TestCase is the test case row data shown on Change details.
type TestCase struct {
	ID       string
	Scenario string
	Done     bool
	ChangeID string
}

// ChangeCreateInput is the backend payload for creating a change.
type ChangeCreateInput struct {
	ProjectID int    `json:"project_id"`
	RefUUID   string `json:"ref_uuid,omitempty"`
	Title     string `json:"title"`
	Brief     string `json:"brief"`
}
