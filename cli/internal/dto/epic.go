package dto

import "time"

// Epic is the current backend epic value. Completion is owned by the server.
type Epic struct {
	ID          int
	ProjectID   int
	Active      bool
	Name        string
	DoneTC      int64
	TotalTC     int64
	Completed   int64
	ChangeCount int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
