package domain

import "time"

type (
	// Epic defines Epic values.
	Epic struct {
		ID          int       `json:"id"`
		ProjectID   int       `json:"project_id"`
		Name        string    `json:"name"`
		Active      bool      `json:"active"`
		DoneTC      int64     `json:"done_tc"`
		TotalTC     int64     `json:"total_tc"`
		Completed   int64     `json:"completed"`
		ChangeCount int       `json:"change_count"`
		CreatedAt   time.Time `json:"created_at"`
		UpdatedAt   time.Time `json:"updated_at"`
	}

	// EpicListRequest defines EpicListRequest values.
	EpicListRequest struct {
		ProjectID int `json:"project_id" validate:"required|min:1"`
	}

	// EpicIDRequest defines EpicIDRequest values.
	EpicIDRequest struct {
		ID int `json:"id" validate:"required|min:1"`
	}

	// EpicCreateRequest defines EpicCreateRequest values.
	EpicCreateRequest struct {
		ProjectID int    `json:"project_id" validate:"required|min:1"`
		Name      string `json:"name" validate:"required"`
	}

	// EpicUpdateRequest defines EpicUpdateRequest values.
	EpicUpdateRequest struct {
		ID   int    `json:"id" validate:"required|min:1"`
		Name string `json:"name" validate:"required"`
	}
)
