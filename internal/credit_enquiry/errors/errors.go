package errors

// ValidationError represents a validation error with detailed information
type ValidationError struct {
	Code    string                 // Error code for client handling
	Message string                 // Human-readable message
	Details map[string]interface{} // Additional error details
}

// ValidationResponse represents the response from validation
type ValidationResponse struct {
	Success bool
	Error   *ValidationError
	Data    interface{} // Optional success data
}

// Error codes
const (
	ErrInvalidUUID      = "INVALID_UUID"
	ErrInvalidLength    = "INVALID_LENGTH"
	ErrInvalidNumeric   = "INVALID_NUMERIC"
	ErrRequiredField    = "REQUIRED_FIELD"
	ErrInternalError    = "INTERNAL_ERROR"
	ErrDuplicateRequest = "DUPLICATE_REQUEST"
	ErrStorageError     = "STORAGE_ERROR"
	ErrPublishError     = "PUBLISH_ERROR"
	ErrBadRequest       = "BAD_REQUEST"
	ErrServiceError     = "SERVICE_ERROR"
)

// NewValidationError creates a new validation error
func NewValidationError(code, message string, details map[string]interface{}) *ValidationError {
	return &ValidationError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// NewValidationResponse creates a new validation response
func NewValidationResponse(success bool, err *ValidationError, data interface{}) *ValidationResponse {
	return &ValidationResponse{
		Success: success,
		Error:   err,
		Data:    data,
	}
}
