package dto

// Project is the project row data used by mch selectors, tables, and details.
type Project struct {
	ID          string
	Name        string
	LastRef     int
	ChangeCount int
	Created     string
	Modified    string
}
