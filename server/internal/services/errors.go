package services

import "net/http"

// statusError is a domain error carrying the HTTP status it maps to, so the
// transport layer can return it as-is instead of translating every case.
type statusError struct {
	status int
	msg    string
}

func (e statusError) Error() string     { return e.msg }
func (e statusError) StatusCode() int   { return e.status }
func (e statusError) DetailMsg() string { return e.msg }

var (
	ErrInvalidCredentials = statusError{http.StatusUnauthorized, "invalid username or password"}
	ErrUsernameTaken      = statusError{http.StatusConflict, "username already taken"}
	ErrSetupComplete      = statusError{http.StatusConflict, "initial setup already complete"}
	ErrForbidden          = statusError{http.StatusForbidden, "admin privileges required"}
	ErrNotFound           = statusError{http.StatusNotFound, "not found"}

	ErrShareNotFound   = statusError{http.StatusNotFound, "share not found"}
	ErrShareExpired    = statusError{http.StatusGone, "this share has expired"}
	ErrPasswordNeeded  = statusError{http.StatusUnauthorized, "this share is password protected"}
	ErrWrongPassword   = statusError{http.StatusUnauthorized, "incorrect password"}
	ErrNotUploadBox    = statusError{http.StatusBadRequest, "this share does not accept uploads"}
	ErrUploadBoxFull   = statusError{http.StatusForbidden, "this box is no longer accepting files"}
	ErrQuotaExceeded   = statusError{http.StatusRequestEntityTooLarge, "storage quota exceeded"}
	ErrFileTooLarge    = statusError{http.StatusRequestEntityTooLarge, "file exceeds the maximum upload size"}
	ErrDiskFull        = statusError{http.StatusInsufficientStorage, "the server is out of disk space"}
	ErrInvalidFilename = statusError{http.StatusBadRequest, "invalid filename"}
	ErrModeImmutable   = statusError{http.StatusBadRequest, "a share's mode cannot be changed after creation"}
)
