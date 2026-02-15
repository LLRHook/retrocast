package service

import "errors"

var (
	ErrNotFound       = errors.New("not found")
	ErrForbidden      = errors.New("forbidden")
	ErrConflict       = errors.New("conflict")
	ErrBadRequest     = errors.New("bad request")
	ErrUnauthorized   = errors.New("unauthorized")
	ErrInternal       = errors.New("internal")
	ErrRoleHierarchy  = errors.New("role hierarchy")
	ErrGone           = errors.New("gone")
)

// ServiceError wraps a sentinel error with a specific code and message for the handler to use.
type ServiceError struct {
	Err     error
	Code    string
	Message string
}

func (e *ServiceError) Error() string { return e.Message }
func (e *ServiceError) Unwrap() error { return e.Err }

// NewError creates a ServiceError wrapping the given sentinel.
func NewError(sentinel error, code, message string) *ServiceError {
	return &ServiceError{Err: sentinel, Code: code, Message: message}
}

// Convenience constructors for common error types.

func NotFound(code, message string) *ServiceError {
	return NewError(ErrNotFound, code, message)
}

func Forbidden(code, message string) *ServiceError {
	return NewError(ErrForbidden, code, message)
}

func BadRequest(code, message string) *ServiceError {
	return NewError(ErrBadRequest, code, message)
}

func Conflict(code, message string) *ServiceError {
	return NewError(ErrConflict, code, message)
}

func Unauthorized(code, message string) *ServiceError {
	return NewError(ErrUnauthorized, code, message)
}

func Internal(code, message string) *ServiceError {
	return NewError(ErrInternal, code, message)
}

func Gone(code, message string) *ServiceError {
	return NewError(ErrGone, code, message)
}

func RoleHierarchyError(message string) *ServiceError {
	return NewError(ErrRoleHierarchy, "ROLE_HIERARCHY", message)
}
