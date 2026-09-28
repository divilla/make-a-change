package testcases

import "fmt"

// Form describes the testcase-specific text rendered by the shell's input widget.
type Form struct {
	Placeholder string
	Help        string
	Status      string
}

// CreateForm supplies the new testcase form's visible text.
func CreateForm() Form {
	return Form{
		Placeholder: "Write a Scenario",
		Help:        "<return> save  |  <ctrl+c> delete prompt  |  <esc> cancel",
		Status:      "new test case",
	}
}

// EditForm supplies the existing testcase form's visible text.
func EditForm() Form {
	form := CreateForm()
	form.Status = "editing test case"
	return form
}

// Summary renders lifecycle feedback for the embedded change-detail screen.
func Summary(m Model) string {
	if m.Busy {
		return "loading test cases"
	}
	if !m.Loaded {
		return "test cases need /retry"
	}
	if len(m.Rows) == 0 {
		return "no test cases"
	}
	return fmt.Sprintf("%d test cases", len(m.Rows))
}

// DetailTitle returns the test case details screen title.
func DetailTitle() string {
	return "TestCaseDetailsScreen - Title: Test Case Details"
}

// CreateTitle returns the new test case screen title.
func CreateTitle() string {
	return "TestCaseCreateScreen - Title: New Test Case"
}

// UpdateTitle returns the edit test case screen title.
func UpdateTitle() string {
	return "TestCaseUpdateScreen - Title: Edit Test Case"
}
