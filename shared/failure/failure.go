package failure

import (
	"errors"
	"net/http"
	"oil/shared/errkey"
)

// Failure is a wrapper for error messages and codes using standard HTTP response codes.
type Failure struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Key     errkey.ErrorKey `json:"key,omitempty"` // Error key for frontend translation
}

var InvalidPageParam = &Failure{Code: http.StatusBadRequest, Message: "invalid page parameter", Key: errkey.ErrInvalidPageParam}
var InvalidLimitParam = &Failure{Code: http.StatusBadRequest, Message: "invalid limit parameter", Key: errkey.ErrInvalidLimitParam}
var ForbiddenError = &Failure{Code: http.StatusForbidden, Message: "You don't have the required permissions", Key: errkey.ErrForbidden}
var ResourceRestrictedError = &Failure{Code: http.StatusForbidden, Message: "You don't have permission to access this resource", Key: errkey.ErrResourceRestricted}

// Error returns the error code and message in a formatted string.
func (e *Failure) Error() string {
	return e.Message
}

// BadRequest returns a new Failure with code for bad requests.
func BadRequest(err error) error {
	if err != nil {
		return &Failure{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		}
	}

	return nil
}

// BadRequestFromString returns a new Failure with code for bad requests with message set from string.
func BadRequestFromString(msg string) error {
	return &Failure{
		Code:    http.StatusBadRequest,
		Message: msg,
	}
}

// UnprocessableEntity returns a new Failure with code for unprocessable entity and message derived from an error interface.
func UnprocessableEntity(err error) error {
	if err != nil {
		return &Failure{
			Code:    http.StatusUnprocessableEntity,
			Message: err.Error(),
		}
	}

	return nil
}

// UnprocessableEntityFromString returns a new Failure with code for unprocessable entity with message set from string.
func UnprocessableEntityFromString(msg string) error {
	return &Failure{
		Code:    http.StatusUnprocessableEntity,
		Message: msg,
	}
}

// Unauthorized returns a new Failure with code for unauthorized requests.
func Unauthorized(msg string) error {
	return &Failure{
		Code:    http.StatusUnauthorized,
		Message: msg,
	}
}

// InternalError returns a new Failure with code for internal error and message derived from an error interface.
func InternalError(err error) error {
	if err != nil {
		return &Failure{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		}
	}

	return nil
}

// Unimplemented returns a new Failure with code for unimplemented method.
func Unimplemented(methodName string) error {
	return &Failure{
		Code:    http.StatusNotImplemented,
		Message: methodName,
	}
}

// NotFound returns a new Failure with code for entity not found.
func NotFound(entityName string) error {
	return &Failure{
		Code:    http.StatusNotFound,
		Message: entityName,
	}
}

// Conflict returns a new Failure with code for conflict situations.
func Conflict(message string) error {
	return &Failure{
		Code:    http.StatusConflict,
		Message: message,
	}
}

func Forbidden(msg string) error {
	return &Failure{
		Code:    http.StatusForbidden,
		Message: msg,
	}
}

// GetCode returns the error code of an error interface.
func GetCode(err error) int {
	var fail *Failure
	if errors.As(err, &fail) {
		return fail.Code
	}

	return http.StatusInternalServerError
}

// GetKey returns the error key of an error interface.
func GetKey(err error) errkey.ErrorKey {
	var fail *Failure
	if errors.As(err, &fail) {
		// If the key is empty, return a default based on the status code
		if fail.Key == "" {
			// Return appropriate default key based on status code
			switch {
			case fail.Code == http.StatusUnprocessableEntity:
				return errkey.ErrValidationFailed
			case fail.Code == http.StatusBadRequest:
				return errkey.ErrValidationFailed
			case fail.Code == http.StatusNotFound:
				return errkey.ErrNotFound
			case fail.Code == http.StatusUnauthorized:
				return errkey.ErrUnauthorized
			case fail.Code == http.StatusForbidden:
				return errkey.ErrForbidden
			case fail.Code >= 500:
				return errkey.ErrInternalServer
			default:
				return errkey.ErrInternalServer
			}
		}
		return fail.Key
	}

	return errkey.ErrInternalServer
}

// NewWithKey creates a new Failure with a key, code, and message.
func NewWithKey(key errkey.ErrorKey, code int, message string) error {
	return &Failure{
		Code:    code,
		Message: message,
		Key:     key,
	}
}

// BadRequestWithKey returns a new Failure with bad request code and error key.
func BadRequestWithKey(key errkey.ErrorKey, message string) error {
	return &Failure{
		Code:    http.StatusBadRequest,
		Message: message,
		Key:     key,
	}
}

// NotFoundWithKey returns a new Failure with not found code and error key.
func NotFoundWithKey(key errkey.ErrorKey, message string) error {
	return &Failure{
		Code:    http.StatusNotFound,
		Message: message,
		Key:     key,
	}
}

// InternalErrorWithKey returns a new Failure with internal error code and error key.
func InternalErrorWithKey(key errkey.ErrorKey, message string) error {
	return &Failure{
		Code:    http.StatusInternalServerError,
		Message: message,
		Key:     key,
	}
}

// UnauthorizedWithKey returns a new Failure with unauthorized code and error key.
func UnauthorizedWithKey(key errkey.ErrorKey, message string) error {
	return &Failure{
		Code:    http.StatusUnauthorized,
		Message: message,
		Key:     key,
	}
}

// ForbiddenWithKey returns a new Failure with forbidden code and error key.
func ForbiddenWithKey(key errkey.ErrorKey, message string) error {
	return &Failure{
		Code:    http.StatusForbidden,
		Message: message,
		Key:     key,
	}
}

// ConflictWithKey returns a new Failure with conflict code and error key.
func ConflictWithKey(key errkey.ErrorKey, message string) error {
	return &Failure{
		Code:    http.StatusConflict,
		Message: message,
		Key:     key,
	}
}

// ServiceUnavailableWithKey returns a new Failure with service unavailable code and error key.
func ServiceUnavailableWithKey(key errkey.ErrorKey, message string) error {
	return &Failure{
		Code:    http.StatusServiceUnavailable,
		Message: message,
		Key:     key,
	}
}
