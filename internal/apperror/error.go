package apperror

import "net/http"

type Error struct {
	Code    string
	Message string
	Status  int
}

func (e *Error) Error() string { return e.Message }

func New(status int, code, message string) *Error {
	return &Error{Code: code, Message: message, Status: status}
}

func Validation(message string) *Error {
	return New(http.StatusBadRequest, "validation_error", message)
}

func UnsupportedMediaType(message string) *Error {
	return New(http.StatusUnsupportedMediaType, "unsupported_media_type", message)
}

func PayloadTooLarge(code, message string) *Error {
	return New(http.StatusRequestEntityTooLarge, code, message)
}

func Forbidden(message string) *Error {
	return New(http.StatusForbidden, "forbidden", message)
}

func NotFound(code, message string) *Error {
	return New(http.StatusNotFound, code, message)
}

func Conflict(code, message string) *Error {
	return New(http.StatusConflict, code, message)
}

func Internal(message string) *Error {
	return New(http.StatusInternalServerError, "internal_error", message)
}
