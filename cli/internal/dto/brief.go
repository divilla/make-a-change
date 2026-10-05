package dto

import "time"

// AgentOutput identifies the final message and persisted session of one exec run.
type AgentOutput struct {
	Final, SessionID string
}

// FileStamp detects creation and modification without interpreting agent responses.
type FileStamp struct {
	Exists bool
	Time   time.Time
}
