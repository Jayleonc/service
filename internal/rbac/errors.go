package rbac

import "errors"

var (
	// ErrInvalidPayload indicates the request body failed validation.
	ErrInvalidPayload = errors.New("invalid payload")
	// ErrResourceNotFound indicates the target role or permission does not exist.
	ErrResourceNotFound = errors.New("resource not found")
	// ErrPermissionDenied indicates the user lacks the required permission.
	ErrPermissionDenied = errors.New("permission denied")
	// ErrPermissionServiceUnavailable indicates that the permission checker has not been wired.
	ErrPermissionServiceUnavailable = errors.New("permission service unavailable")
)
