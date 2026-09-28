// Package epics owns epic forms, operations and presentation.
package epics

// ListCommands returns discoverable epic list actions.
func ListCommands() []string { return []string{"/new-epic", "/retry", "/help", "/find", "/return"} }

// DetailCommands returns discoverable epic detail actions.
func DetailCommands() []string {
	return []string{"/edit", "/delete", "/documents", "/retry", "/help", "/find", "/return"}
}
