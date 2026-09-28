package dto

import "time"

// Change contains only fields returned by change list/details, without documents.
type Change struct {
	ID            int       `json:"id"`
	ProjectID     int       `json:"project_id"`
	RefUUID       string    `json:"ref_uuid"`
	Ref           *int32    `json:"ref"`
	Slug          *string   `json:"slug"`
	EpicID        *int      `json:"epic_id"`
	EpicName      *string   `json:"epic_name"`
	ChangePhase   string    `json:"change_phase"`
	ChangeTypes   []string  `json:"change_types"`
	Title         string    `json:"title"`
	Open          bool      `json:"open"`
	DoneTC        int64     `json:"done_tc"`
	TotalTC       int64     `json:"total_tc"`
	Completed     int64     `json:"completed"`
	UpdatedAt     time.Time `json:"updated_at"`
	AfterChangeID *int      `json:"after_change_id"`
	PRUrl         string    `json:"pr_url"`
	CreatedAt     time.Time `json:"created_at"`
}

// Document is one version returned by the document API.
type Document struct {
	ID        int       `json:"id"`
	RefID     int       `json:"ref_id"`
	RefTable  string    `json:"ref_table"`
	DocType   string    `json:"doc_type"`
	Body      string    `json:"body"`
	AgentEdit bool      `json:"agent_edit"`
	Current   bool      `json:"current"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	HTML      string    `json:"html"`
}

// DocumentInput appends a version without changing earlier versions.
type DocumentInput struct {
	RefID     int    `json:"ref_id"`
	RefTable  string `json:"ref_table"`
	DocType   string `json:"doc_type"`
	Body      string `json:"body"`
	AgentEdit bool   `json:"agent_edit"`
}
