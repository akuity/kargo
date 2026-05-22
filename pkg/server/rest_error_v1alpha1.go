package server

// ErrorResponse represents the standard REST error response body.
type ErrorResponse struct {
	Error string `json:"error,omitempty"`
} // @name ErrorResponse
