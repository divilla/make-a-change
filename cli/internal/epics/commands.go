package epics

// ListCommands returns slash commands for the epics list screen.
func ListCommands() []string {
	return []string{"/new-epic", "/help", "/find", "/return"}
}

// DetailCommands returns slash commands for epic details.
func DetailCommands() []string {
	return []string{"/edit", "/delete", "/help", "/find", "/return"}
}
