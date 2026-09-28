// Package projects provides project list and detail presentation helpers.
package projects

// ListCommands returns slash commands for the projects list screen.
func ListCommands() []string {
	return []string{"/new-project", "/help", "/find", "/return"}
}

// DetailCommands returns slash commands for project details.
func DetailCommands() []string {
	return []string{"/edit", "/help", "/find", "/return"}
}
