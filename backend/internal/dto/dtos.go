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

	// Config defines global Flow and Task option values.
	Config struct {
		Slug                  string   `json:"slug"`
		FlowStages            []string `json:"flow_stages"`
		FlowStagesHelp        []string `json:"flow_stages_help"`
		FlowStageModesDefault []string `json:"flow_stage_modes_default"`
		FlowStageEntryScripts []string `json:"flow_stage_entry_scripts"`
		FlowStagePrompts      []string `json:"flow_stage_prompts"`
		FlowStageExitScripts  []string `json:"flow_stage_exit_scripts"`
		StageModes            []string `json:"stage_modes"`
		StageModesHelp        []string `json:"stage_modes_help"`
		TaskStatuses          []string `json:"task_statuses"`
		TaskStatusesHelp      []string `json:"task_statuses_help"`
		TaskSteps             []string `json:"task_steps"`
		TaskStepsHelp         []string `json:"task_steps_help"`
	}
)
