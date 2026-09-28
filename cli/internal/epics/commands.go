// Package epics provides the retained epic navigation shell.
package epics

// ListCommands returns slash commands for the epics list screen.
func ListCommands() []string {
	return []string{"/help", "/find", "/return"}
}

// DetailCommands returns slash commands for epic details.
func DetailCommands() []string {
	return []string{"/help", "/find", "/return"}
}
