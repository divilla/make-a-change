// Package changes provides change list, detail, and editing presentation helpers.
package changes

// ListCommands returns slash commands for the changes list screen.
func ListCommands() []string {
	return []string{"/new-change", "/del-change", "/phase-filter", "/types-filter", "/epic-filter", "/find-filter", "/inactive-filter", "/clear-filters", "/help", "/return"}
}

// DetailCommands returns slash commands for change details.
func DetailCommands() []string {
	return []string{"/new-comment", "/find", "/document", "/title", "/brief", "/pr-url", "/after-change", "/active", "/retry", "/help", "/new-testcase", "/phase", "/epic", "/types", "/edit-spec", "/delete", "/documents", "/return"}
}

// HelpView describes ordinary change operations and safe recovery.
func HelpView() string {
	return "Changes: /new-change, /del-change (confirm deactivation), /undel-change (inactive list restore), /phase-filter, /types-filter, /epic-filter, /find-filter, /inactive-filter, /clear-filters\nDelete confirms deactivation; inactive Space restores. /inactive-filter persists through navigation.\nDetails: /document (configured types), /title, /phase, /types, /epic, /after-change, /active, /pr-url, /brief, /edit-spec, /delete\nCreate: /new-change opens an empty brief in the editor; a first nonblank # Title is required. Saving starts brief rewriting and spec writing.\nEpic and prerequisite can clear to null; toggling all types off clears types.\nPR URL requires HTTP(S) and cannot be cleared. /retry reads only after a committed write."
}
