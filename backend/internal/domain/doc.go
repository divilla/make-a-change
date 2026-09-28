package domain

import "time"

// Doc is an append-only stored doc with its safe HTML representation.
type Doc struct {
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

// DocListRequest identifies the project, epic or change owning the docs.
type DocListRequest struct {
	RefID    int    `json:"ref_id" validate:"required|min:1"`
	RefTable string `json:"ref_table" validate:"required|in:project,epic,change"`
}

// DocIDRequest identifies one stored doc.
type DocIDRequest struct {
	ID int `json:"id" validate:"required|min:1"`
}

// DocInsertRequest appends a new current doc of the configured type.
type DocInsertRequest struct {
	RefID     int    `json:"ref_id" validate:"required|min:1"`
	RefTable  string `json:"ref_table" validate:"required|in:project,epic,change"`
	DocType   string `json:"doc_type" validate:"required"`
	Body      string `json:"body" validate:"required"`
	AgentEdit *bool  `json:"agent_edit"`
}
