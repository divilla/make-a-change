package dto

import "time"

type (
	// Epic defines Epic values.
	Epic struct {
		ID          int       `json:"id"`
		Version     int16     `json:"version"`
		ProjectID   int       `json:"project_id"`
		Name        string    `json:"name"`
		DoneTC      int16     `json:"done_tc"`
		TotalTC     int16     `json:"total_tc"`
		Completed   int16     `json:"completed"`
		ChangeCount int       `json:"change_count"`
		Created     time.Time `json:"created"`
		Modified    time.Time `json:"modified"`
	}

	// EpicListRequest defines EpicListRequest values.
	EpicListRequest struct {
		ProjectID int `json:"project_id"`
	}

	// EpicIDRequest defines EpicIDRequest values.
	EpicIDRequest struct {
		ID int `json:"id"`
	}

	// EpicCreateRequest defines EpicCreateRequest values.
	EpicCreateRequest struct {
		ProjectID int    `json:"project_id"`
		Name      string `json:"name"`
	}

	// EpicUpdateRequest defines EpicUpdateRequest values.
	EpicUpdateRequest struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
)
