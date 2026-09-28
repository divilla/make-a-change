// Package domain defines business models and requests shared across backend layers.
package domain

import (
	"time"

	"github.com/gofrs/uuid/v5"
)

type (
	// Change is the legacy testcase mutation DTO. Remove with the P4 testcase migration.
	Change struct {
		ID          int       `json:"id"`
		Version     int16     `json:"version"`
		RefUUID     string    `json:"ref_uuid"`
		Ref         *int32    `json:"ref"`
		Slug        *string   `json:"slug"`
		ProjectID   int       `json:"project_id"`
		EpicID      *int      `json:"epic_id"`
		EpicName    *string   `json:"epic_name"`
		ChangePhase string    `json:"change_phase"`
		ChangeTypes []string  `json:"change_types"`
		Title       string    `json:"title"`
		Brief       string    `json:"brief"`
		BriefHTML   string    `json:"brief_html"`
		Spec        string    `json:"spec"`
		SpecHTML    string    `json:"spec_html"`
		PR          string    `json:"pr"`
		PRHtml      string    `json:"pr_html"`
		PRUrl       string    `json:"pr_url"`
		Open        bool      `json:"open"`
		DoneTC      int16     `json:"done_tc"`
		TotalTC     int16     `json:"total_tc"`
		Completed   int16     `json:"completed"`
		Created     time.Time `json:"created"`
		Modified    time.Time `json:"modified"`
	}

	// ChangeListItem defines ChangeListItem values.
	ChangeListItem struct {
		ID          int       `json:"id"`
		RefUUID     string    `json:"ref_uuid"`
		Ref         *int32    `json:"ref"`
		Slug        *string   `json:"slug"`
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
		Modified    time.Time `json:"modified"`
	}

	// ChangeDetails contains fields exposed by the current change details view.
	ChangeDetails struct {
		ChangeListItem
		PRUrl   string    `json:"pr_url"`
		Created time.Time `json:"created"`
	}

	// ChangeRenderedArtifactsRequest defines ChangeRenderedArtifactsRequest values.
	ChangeRenderedArtifactsRequest struct {
		IDs []int `json:"ids"`
	}

	// ChangeRenderedArtifact defines ChangeRenderedArtifact values.
	ChangeRenderedArtifact struct {
		ID       int    `json:"id"`
		SpecHTML string `json:"spec_html"`
		PRHtml   string `json:"pr_html"`
	}

	// ChangeRenderedArtifactsResponse defines ChangeRenderedArtifactsResponse values.
	ChangeRenderedArtifactsResponse struct {
		Artifacts []ChangeRenderedArtifact `json:"artifacts"`
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

	// ChangeUpdateChangeTypesRequest defines ChangeUpdateChangeTypesRequest values.
	ChangeUpdateChangeTypesRequest struct {
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

	// ChangeUpdateBriefRequest defines ChangeUpdateBriefRequest values.
	ChangeUpdateBriefRequest struct {
		ID        int    `json:"id" validate:"required|min:1"`
		Brief     string `json:"brief"`
		AgentEdit *bool  `json:"agent_edit"`
	}

	// ChangeUpdateSpecRequest defines ChangeUpdateSpecRequest values.
	ChangeUpdateSpecRequest struct {
		ID        int    `json:"id" validate:"required|min:1"`
		Spec      string `json:"spec"`
		AgentEdit *bool  `json:"agent_edit"`
	}

	// ChangeUpdatePRRequest defines ChangeUpdatePRRequest values.
	ChangeUpdatePRRequest struct {
		ID        int    `json:"id" validate:"required|min:1"`
		PR        string `json:"pr"`
		AgentEdit *bool  `json:"agent_edit"`
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

// ChangeDocumentSetRequest writes one configured document kind.
type ChangeDocumentSetRequest struct {
	ID        int    `json:"id" validate:"required|min:1"`
	DocType   string `json:"doc_type" validate:"required"`
	Body      string `json:"body" validate:"required"`
	AgentEdit *bool  `json:"agent_edit"`
}

// ChangeDocument is a current stored document and its safe rendered representation.
type ChangeDocument struct {
	ID        int       `json:"id"`
	DocType   string    `json:"doc_type"`
	Body      string    `json:"body"`
	AgentEdit bool      `json:"agent_edit"`
	Created   time.Time `json:"created"`
	HTML      string    `json:"html"`
}

// ChangeArtifactSource contains only the raw documents needed by the bulk read.
type ChangeArtifactSource struct {
	ID   int
	Spec string
	PR   string
}
