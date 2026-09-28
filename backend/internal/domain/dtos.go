package domain

// Config is the configuration selected by a project's stored slug.
type Config struct {
	Slug         string   `json:"slug"`
	ProjectDocs  []string `json:"project_docs"`
	EpicDocs     []string `json:"epic_docs"`
	ChangeDocs   []string `json:"change_docs"`
	ChangePhases []string `json:"change_phases"`
	ChangeColors []string `json:"change_colors"`
	ChangeTypes  []string `json:"change_types"`
}
