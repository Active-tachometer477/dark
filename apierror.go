package dark

import "fmt"

// APIError represents an error with an HTTP status code for API responses.
type APIError struct {
	Status  int
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%d: %s", e.Status, e.Message)
}

// NewAPIError creates an APIError with the given status code and message.
func NewAPIError(status int, message string) *APIError {
	return &APIError{Status: status, Message: message}
}
