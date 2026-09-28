package domain

import "time"

type (
	// Project defines Project values.
	Project struct {
		ID          int       `json:"id"`
		Name        string    `json:"name"`
		Config      string    `json:"config"`
		LastRef     int32     `json:"last_ref"`
		CreatedAt   time.Time `json:"created_at"`
		UpdatedAt   time.Time `json:"updated_at"`
		ChangeCount int       `json:"change_count"`
	}

	// ProjectIDRequest defines ProjectIDRequest values.
	ProjectIDRequest struct {
		ID int `json:"id" validate:"required|min:1"`
	}

	// ProjectCreateRequest defines ProjectCreateRequest values.
	ProjectCreateRequest struct {
		Name string `json:"name" validate:"required"`
	}

	// ProjectUpdateRequest defines ProjectUpdateRequest values.
	ProjectUpdateRequest struct {
		ID   int    `json:"id" validate:"required|min:1"`
		Name string `json:"name" validate:"required"`
	}
)
