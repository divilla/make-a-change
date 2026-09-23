package dto

// Change is the change row and detail data used by mch.
type Change struct {
	ID          string
	RefUUID     string
	Ref         string
	Slug        string
	ProjectID   string
	EpicID      string
	EpicName    string
	ChangePhase string
	ChangeTypes []string
	Title       string
	Def         string
	Spec        string
	PR          string
	PRUrl       string
	AgentEdit   bool
	Open        bool
	Done        int
	Total       int
	Completed   int
	TestCases   []TestCase
	Created     string
	Modified    string
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
	ProjectID int
	RefUUID   string
	Title     string
	Def       string
}
