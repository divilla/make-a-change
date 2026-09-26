package dto

type (
	// ChangePhase defines ChangePhase values.
	ChangePhase struct {
		Slug     string `json:"slug"`
		Priority int    `json:"priority"`
		Color    string `json:"color,omitempty"`
	}

	// ChangeType defines ChangeType values.
	ChangeType struct {
		Slug     string `json:"slug"`
		Priority int    `json:"priority"`
	}

	// Config defines global change options.
	Config struct {
		Slug         string   `json:"slug"`
		ChangePhases []string `json:"change_phases"`
		ChangeTypes  []string `json:"change_types"`
		ChangeDocs   []string `json:"change_docs"`
	}
)
