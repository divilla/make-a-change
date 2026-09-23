package dto

import "time"

type (
	// Project defines Project values.
	Project struct {
		ID          int       `json:"id"`
		Name        string    `json:"name"`
		LastRef     int32     `json:"last_ref"`
		Created     time.Time `json:"created"`
		Modified    time.Time `json:"modified"`
		ChangeCount int       `json:"change_count"`
	}

	// ProjectIDRequest defines ProjectIDRequest values.
	ProjectIDRequest struct {
		ID int `json:"id"`
	}

	// ProjectCreateRequest defines ProjectCreateRequest values.
	ProjectCreateRequest struct {
		Name string `json:"name"`
	}

	// ProjectUpdateRequest defines ProjectUpdateRequest values.
	ProjectUpdateRequest struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
)
