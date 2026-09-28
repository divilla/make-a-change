package dto

// BriefQuestion carries the affected brief context and an optional user answer.
type BriefQuestion struct {
	ID      string `json:"id"`
	Text    string `json:"text"`
	Context string `json:"context"`
	Answer  string `json:"-"`
}

// BriefOutput is the structured result of one clarification step.
type BriefOutput struct {
	InputRevision  uint64          `json:"input_revision"`
	RewrittenBrief string          `json:"rewritten_brief"`
	Questions      []BriefQuestion `json:"questions"`
	Unresolved     []string        `json:"unresolved"`
	ReadyForSpec   bool            `json:"ready_for_spec"`
}

// BriefRequest passes explicit input and output context to the process adapter.
type BriefRequest struct {
	Root          string
	OriginalPath  string
	InputPath     string
	ContextPath   string
	QuestionsPath string
	AnswersPath   string
	OutputPath    string
	Prompt        string
	Revision      uint64
	ProjectID     int
	ChangeID      int
	DocumentID    int
	Original      string
	Brief         string
	Questions     []BriefQuestion
	Progress      chan<- string
}
