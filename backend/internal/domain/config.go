package domain

// ConfigSlugRequest identifies a configuration by its immutable slug.
type ConfigSlugRequest struct {
	Slug string `json:"slug" validate:"required"`
}

// ConfigWriteRequest requires all six arrays; empty arrays are allowed, omitted/null arrays are not.
type ConfigWriteRequest struct {
	Slug         string   `json:"slug" validate:"required"`
	ProjectDocs  []string `json:"project_docs"`
	EpicDocs     []string `json:"epic_docs"`
	ChangeDocs   []string `json:"change_docs"`
	ChangePhases []string `json:"change_phases"`
	ChangeColors []string `json:"change_colors"`
	ChangeTypes  []string `json:"change_types"`
}
