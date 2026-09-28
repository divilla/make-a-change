// Package changes provides change list, detail, and editing presentation helpers.
package changes

// ListCommands returns slash commands for the changes list screen.
func ListCommands() []string {
	return []string{"/new-change", "/phase-filter", "/epic-filter", "/type-filter", "/find-filter", "/clear-filters", "/help", "/return"}
}

// DetailCommands returns slash commands for change details.
func DetailCommands() []string {
	return []string{"/new-testcase", "/phase", "/epic", "/types", "/edit-spec", "/delete", "/return"}
}
