// Package dto contains shared business values crossing feature boundaries.
package dto

import "time"

// Project is the backend project value, independent of display formatting.
type Project struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Config      string    `json:"config"`
	LastRef     int32     `json:"last_ref"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	ChangeCount int       `json:"change_count"`
}

// ProjectConfig contains the selected project's ordered catalogs.
type ProjectConfig struct {
	Slug         string   `json:"slug"`
	ProjectDocs  []string `json:"project_docs"`
	EpicDocs     []string `json:"epic_docs"`
	ChangeDocs   []string `json:"change_docs"`
	ChangePhases []string `json:"change_phases"`
	ChangeColors []string `json:"change_colors"`
	ChangeTypes  []string `json:"change_types"`
}
