// Package domain defines business models and requests shared across backend layers.
package domain

import (
	"time"

	"github.com/gofrs/uuid/v5"
)

type (
	// ChangeListItem defines ChangeListItem values.
	ChangeListItem struct {
		ID          int       `json:"id"`
		RefUUID     string    `json:"ref_uuid"`
		RefSlug     *string   `json:"ref_slug"`
		ProjectID   int       `json:"project_id"`
		ChangePhase string    `json:"change_phase"`
		ChangeTypes []string  `json:"change_types"`
		EpicID      *int      `json:"epic_id"`
		EpicName    *string   `json:"epic_name"`
		Title       string    `json:"title"`
		Open        bool      `json:"open"`
		DoneTC      int64     `json:"done_tc"`
		TotalTC     int64     `json:"total_tc"`
		Completed   int64     `json:"completed"`
		UpdatedAt   time.Time `json:"updated_at"`
	}

	// ChangeDetails contains fields exposed by the current change details view.
	ChangeDetails struct {
		ChangeListItem
		AfterChangeID   *int      `json:"after_change_id"`
		AfterChangeName *string   `json:"after_change_name"`
		PRUrl           string    `json:"pr_url"`
		CreatedAt       time.Time `json:"created_at"`
	}

	// ChangeListRequest defines ChangeListRequest values.
	ChangeListRequest struct {
		ProjectID int `json:"project_id" validate:"required|min:1"`
	}

	// ChangeIDRequest defines ChangeIDRequest values.
	ChangeIDRequest struct {
		ID int `json:"id" validate:"required|min:1"`
	}

	// ChangeCreateRequest defines ChangeCreateRequest values.
	ChangeCreateRequest struct {
		ProjectID int        `json:"project_id" validate:"required|min:1"`
		RefUUID   *uuid.UUID `json:"ref_uuid"`
		Title     string     `json:"title"`
		Brief     string     `json:"brief"`
	}

	// ChangeUpdatePhaseRequest defines ChangeUpdatePhaseRequest values.
	ChangeUpdatePhaseRequest struct {
		ID          int    `json:"id" validate:"required|min:1"`
		ChangePhase string `json:"change_phase"`
	}

	// ChangeUpdateTypesRequest defines ChangeUpdateTypesRequest values.
	ChangeUpdateTypesRequest struct {
		ID          int      `json:"id" validate:"required|min:1"`
		ChangeTypes []string `json:"change_types"`
	}

	// ChangeUpdateEpicRequest defines ChangeUpdateEpicRequest values.
	ChangeUpdateEpicRequest struct {
		ID     int  `json:"id" validate:"required|min:1"`
		EpicID *int `json:"epic_id"`
	}

	// ChangeUpdateTitleRequest defines ChangeUpdateTitleRequest values.
	ChangeUpdateTitleRequest struct {
		ID    int    `json:"id" validate:"required|min:1"`
		Title string `json:"title"`
	}

	// ChangeUpdateSlugRequest updates the editable suffix of a change slug.
	ChangeUpdateSlugRequest struct {
		ID   int    `json:"id" validate:"required|min:1"`
		Slug string `json:"slug"`
	}

	// ChangeUpdatePRUrlRequest defines ChangeUpdatePRUrlRequest values.
	ChangeUpdatePRUrlRequest struct {
		ID    int    `json:"id" validate:"required|min:1"`
		PRUrl string `json:"pr_url"`
	}
	// ChangeUpdateOpenRequest defines ChangeUpdateOpenRequest values.
	ChangeUpdateOpenRequest struct {
		ID   int   `json:"id" validate:"required|min:1"`
		Open *bool `json:"open"`
	}
)

// ChangeUpdateAfterChangeRequest sets or clears the prerequisite change.
type ChangeUpdateAfterChangeRequest struct {
	ID            int  `json:"id" validate:"required|min:1"`
	AfterChangeID *int `json:"after_change_id"`
}
