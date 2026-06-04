package apperror

import (
	"errors"
	"net/http"
)

type Error struct {
	Code    string
	Message string
	Status  int
	Err     error
}

func (e *Error) Error() string {
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Err
}

func Convert(err error) *Error {
	if appErr, ok := errors.AsType[*Error](err); ok {
		return appErr
	}
	return Internal()
}

func New(code string, message string, status int) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Status:  status,
	}
}

func BadRequest(message string) *Error {
	return New("BAD_REQUEST", message, http.StatusBadRequest)
}

func Unauthorized(message string) *Error {
	return New("UNAUTHORIZED", message, http.StatusUnauthorized)
}

func NotFound(message string) *Error {
	return New("NOT_FOUND", message, http.StatusNotFound)
}

func Conflict(message string) *Error {
	return New("CONFLICT", message, http.StatusConflict)
}

func Internal() *Error {
	return New("INTERNAL_SERVER_ERROR", "something went wrong", http.StatusInternalServerError)
}
