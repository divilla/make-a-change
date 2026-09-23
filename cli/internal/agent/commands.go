package agent

// MainCommands returns slash commands for the main screen.
func MainCommands() []string {
	return []string{"/changes", "/epics", "/projects", "/select-project", "/config", "/help", "/quit"}
}
