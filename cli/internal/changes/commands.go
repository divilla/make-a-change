// Package changes provides change list, detail, and editing presentation helpers.
package changes

// ListCommands returns slash commands for the changes list screen.
func ListCommands() []string {
	return []string{"/new-change", "/phase-filter", "/types-filter", "/epic-filter", "/find-filter", "/clear-filters", "/help", "/return"}
}

// DetailCommands returns slash commands for change details.
func DetailCommands() []string {
	return []string{"/find", "/document", "/title", "/brief", "/pr-url", "/after-change", "/open", "/retry", "/help", "/new-testcase", "/phase", "/epic", "/types", "/edit-spec", "/delete", "/documents", "/return"}
}

// HelpView describes ordinary change operations and safe recovery.
func HelpView() string {
	return "Changes: /new-change, /phase-filter, /types-filter, /epic-filter, /find-filter, /clear-filters\nDetails: /brief-clarify, /document (configured types), /title, /phase, /types, /epic, /after-change, /open, /pr-url, /brief, /edit-spec, /delete\nCreate: Ctrl+T title, Ctrl+U optional UUID, Ctrl+E brief editor; Enter saves.\nEpic and prerequisite can clear to null; toggling all types off clears types.\nPR URL requires HTTP(S) and cannot be cleared. /retry reads only after a committed write."
}
