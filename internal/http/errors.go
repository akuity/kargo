package http

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// BadRequestError returns an error
// wrapped in a 400 status code.
func BadRequestError(err any) error {
	code := http.StatusBadRequest
	return &Error{
		Code:       code,
		StatusText: http.StatusText(code),
		Msg:        toString(err),
	}
}

// BadRequestErrorf returns a formatted error
// wrapped in a 400 status code.
func BadRequestErrorf(format string, args ...any) error {
	return BadRequestError(fmt.Errorf(format, args...))
}

// UnauthorizedError returns an error
// wrapped in a 401 status code.
func UnauthorizedError(err any) error {
	code := http.StatusUnauthorized
	return &Error{
		Code:       code,
		StatusText: http.StatusText(code),
		Msg:        toString(err),
	}
}

// UnauthorizedErrorf returns a formatted error
// wrapped in a 401 status code.
func UnauthorizedErrorf(format string, args ...any) error {
	return UnauthorizedError(fmt.Errorf(format, args...))
}

// ForbiddenError returns an error
// wrapped in a 403 status code.
func ForbiddenError(err any) error {
	code := http.StatusForbidden
	return &Error{
		Code:       code,
		StatusText: http.StatusText(code),
		Msg:        toString(err),
	}
}

// ForbiddenError returns a formatted error
// wrapped in a 403 status code.
func ForbiddenErrorf(format string, args ...any) error {
	return ForbiddenError(fmt.Errorf(format, args...))
}

// ServerError returns an error
// wrapped in a 500 status code.
func ServerError(err any) error {
	code := http.StatusInternalServerError
	return &Error{
		Code:       code,
		StatusText: http.StatusText(code),
		Msg:        toString(err),
	}
}

// ServerErrorf returns a formatted error
// wrapped in a 500 status code.
func ServerErrorf(format string, args ...any) error {
	return ServerError(fmt.Errorf(format, args...))
}

// WriteBadRequestError sets a 400 status code header
// before writing the error as json.
func WriteBadRequestError(w http.ResponseWriter, err any) {
	WriteError(w,
		http.StatusBadRequest,
		BadRequestError(err),
	)
}

// WriteBadRequestErrorf sets a 400 status code header
// before writing the formatted error as json.
func WriteBadRequestErrorf(w http.ResponseWriter, format string, args ...any) {
	WriteError(w,
		http.StatusBadRequest,
		BadRequestErrorf(format, args...),
	)
}

// WriteUnauthorizedError sets a 401 status code header
// before writing the error as json.
func WriteUnauthorizedError(w http.ResponseWriter, err any) {
	WriteError(w,
		http.StatusUnauthorized,
		UnauthorizedError(err),
	)
}

// WriteUnauthorizedErrorf sets a 401 status code header
// before writing the formatted error as json.
func WriteUnauthorizedErrorf(w http.ResponseWriter, format string, args ...any) {
	WriteError(w,
		http.StatusUnauthorized,
		UnauthorizedErrorf(format, args...),
	)
}

// WriteForbiddenError sets a 403 status code header
// before writing the error as json.
func WriteForbiddenError(w http.ResponseWriter, err any) {
	WriteError(w,
		http.StatusForbidden,
		ForbiddenError(err),
	)
}

// WriteForbiddenErrorf sets a 403 status code header
// before writing the formatted error as json.
func WriteForbiddenErrorf(w http.ResponseWriter, format string, args ...any) {
	WriteError(w,
		http.StatusForbidden,
		ForbiddenErrorf(format, args...),
	)
}

// WriteServerError sets a 500 status code header
// before writing the error as json.
func WriteServerError(w http.ResponseWriter, err any) {
	WriteError(w,
		http.StatusInternalServerError,
		ServerError(err),
	)
}

// WriteServerErrorf sets a 500 status code header
// before writing the formatted error as json.
func WriteServerErrorf(w http.ResponseWriter, format string, args ...any) {
	WriteError(w,
		http.StatusInternalServerError,
		ServerErrorf(format, args...),
	)
}

// WriteError sets the Content-Type header to application/json
// and then writes the error as json to the response writer
func WriteError(w http.ResponseWriter, statusCode int, err any) {
	w.WriteHeader(statusCode)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(
		map[string]string{"error": toString(err)},
	)
}

// WriteError sets the Content-Type header to application/json
// and then writes a formatted error as json to the resopnse writer.
func WriteErrorf(w http.ResponseWriter, statusCode int, format string, args ...any) {
	WriteError(w, statusCode, fmt.Errorf(format, args...))
}

// Write sets the statusCode on w before writing msg as json.
func Write(w http.ResponseWriter, statusCode int, msg string) {
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(
		map[string]string{
			"msg": msg,
		},
	)
}

// Writef sets the statusCode on w before writing a formatted error
// to the response writer as json.
func Writef(w http.ResponseWriter, statusCode int, format string, args ...any) {
	Write(w, statusCode, fmt.Sprintf(format, args...))
}

// implement error
type Error struct {
	Code       int
	StatusText string
	Msg        string
}

func (e *Error) Error() string {
	if e.Msg == "" {
		return e.StatusText
	}
	return fmt.Sprintf("%s: %s", e.StatusText, e.Msg)
}

// CodeFrom returns the http status code for the given error.
// If err is not of type xhttp.Error an internal server error is returned.
func CodeFrom(err error) int {
	apiErr, ok := err.(*Error)
	if !ok {
		return http.StatusInternalServerError
	}
	return apiErr.Code
}

func toString(err any) string {
	var msg string
	switch t := err.(type) {
	case error:
		msg = t.Error()
	case string:
		msg = t
	default:
		msg = fmt.Sprintf("%v", t)
	}
	return msg
}
