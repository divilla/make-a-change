package testcases

// DetailCommands returns slash commands for test case details.
func DetailCommands() []string {
	return []string{"/new-testcase", "/edit", "/delete", "/save", "/cancel", "/return"}
}

// EditCommands returns slash commands for test case edit/create screens.
func EditCommands() []string {
	return []string{"/save", "/cancel"}
}
