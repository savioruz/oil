package response

import (
	"encoding/json"
	"errors"
	"net/http"
	"oil/shared/constant"
	"oil/shared/failure"
	"oil/shared/logger"
)

type Data[T any] struct {
	Data *T `json:"data,omitempty"`
}

type Error struct {
	Error *string `json:"error,omitempty"`
}

// FieldError represents a single field validation error
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationErrors represents validation errors response
type ValidationErrors struct {
	Errors []FieldError `json:"errors"`
}

type Message struct {
	Message *string `json:"message,omitempty"`
}

// WithMessage sends a response with a simple text message
func WithMessage(writer http.ResponseWriter, code int, message string) {
	response(writer, code, Message{Message: &message})
}

// WithJSON sends a response containing a JSON object
func WithJSON(writer http.ResponseWriter, code int, jsonPayload interface{}) {
	response(writer, code, Data[any]{Data: &jsonPayload})
}

// WithError sends a response with an error message
// For validation errors (422), it returns an array of field errors with format:
//
//	{"errors": [{"field": "title", "message": "Title is required"}]}
//
// For other errors, it returns a single error key:
//
//	{"error": "gallery.not_found"}
//
// See Error.md for complete error documentation
func WithError(writer http.ResponseWriter, err error) {
	code := failure.GetCode(err)

	// Check if this is a validation error with field details
	var valErr *failure.ValidationError
	if errors.As(err, &valErr) && len(valErr.Fields) > 0 {
		// Build array of field errors
		fieldErrors := make([]FieldError, len(valErr.Fields))
		for i, fieldErr := range valErr.Fields {
			fieldErrors[i] = FieldError{
				Field:   fieldErr.Field,
				Message: fieldErr.Message, // Human-readable message, not the error key
			}
		}

		response(writer, code, ValidationErrors{Errors: fieldErrors})
		return
	}

	// For non-validation errors, return the general error key
	key := failure.GetKey(err)
	errorValue := string(key)
	response(writer, code, Error{Error: &errorValue})
}

// WithRequestLimitExceeded sends a default response for when the request limit is exceeded
func WithRequestLimitExceeded(writer http.ResponseWriter) {
	WithMessage(writer, http.StatusTooManyRequests, constant.ResponseErrorRequestLimitExceeded)
}

// WithPreparingShutdown sends a default response for when the server is preparing to shut down
func WithPreparingShutdown(writer http.ResponseWriter) {
	WithMessage(writer, http.StatusServiceUnavailable, constant.ResponseErrorPrepareShutdown)
}

// WithUnhealthy sends a default response for when the server is unhealthy
func WithUnhealthy(writer http.ResponseWriter) {
	WithMessage(writer, http.StatusServiceUnavailable, constant.ResponseErrorUnhealthy)
}

func response(writer http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		logger.ErrorWithStack(err)

		return
	}

	writer.Header().Set(constant.RequestHeaderContentType, constant.ContentTypeJSON)
	writer.WriteHeader(code)
	_, err = writer.Write(response)

	if err != nil {
		logger.ErrorWithStack(err)
	}
}
