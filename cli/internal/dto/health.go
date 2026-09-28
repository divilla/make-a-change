package dto

// Health preserves the backend status and the route used for a diagnostic.
type Health struct {
	Route      string
	HTTPStatus int
	Status     string
	API        string
	Database   string
	Error      string
}
