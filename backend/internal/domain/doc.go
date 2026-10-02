package domain

import "time"

// Doc is a stored document, including retained history and nullable deletion time.
type Doc struct {
	ID        int        `json:"id"`
	RefTable  string     `json:"ref_table"`
	RefID     int        `json:"ref_id"`
	DocType   string     `json:"doc_type"`
	Body      string     `json:"body"`
	HTML      string     `json:"html"`
	AgentEdit bool       `json:"agent_edit"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at"`
}

// DocIDRequest identifies one stored doc.
type DocIDRequest struct {
	ID int `json:"id" validate:"required|min:1"`
}

// DocListRequest identifies the project, epic or change owning the docs.
type DocListRequest struct {
	RefTable string `json:"ref_table" validate:"required|in:project,epic,change"`
	RefID    int    `json:"ref_id" validate:"required|min:1"`
}

// DocInsertRequest appends a new current doc of the configured type.
type DocInsertRequest struct {
	RefTable  string `json:"ref_table" validate:"required|in:project,epic,change"`
	RefID     int    `json:"ref_id" validate:"required|min:1"`
	DocType   string `json:"doc_type" validate:"required"`
	Body      string `json:"body" validate:"required"`
	AgentEdit *bool  `json:"agent_edit"`
}

// DocCommentInsertRequest appends a comment without selecting an active document.
type DocCommentInsertRequest struct {
	RefTable  string `json:"ref_table" validate:"required|in:project,epic,change"`
	RefID     int    `json:"ref_id" validate:"required|min:1"`
	Body      string `json:"body" validate:"required"`
	AgentEdit *bool  `json:"agent_edit"`
}

// DocCommentUpdateRequest edits an existing comment; an explicit empty body is allowed.
type DocCommentUpdateRequest struct {
	ID   int     `json:"id" validate:"required|min:1"`
	Body *string `json:"body" validate:"required"`
}
