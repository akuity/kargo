package http

import (
	"encoding/json"
	"errors"
	"net/http"
)

type HTTPError struct { // nolint: revive
	code int
	err  error
}

func (e *HTTPError) Error() string {
	return e.err.Error()
}

func (e *HTTPError) Code() int {
	return e.code
}

// Error returns an error that can be used to write an
// HTTP response with an error message and a specific status code.
func Error(err error, code int) error {
	return &HTTPError{
		code: code,
		err:  err,
	}
}

// ErrorStr is like Error but takes a string message instead of an error.
func ErrorStr(err string, code int) error {
	return Error(errors.New(err), code)
}

// WriteErrorJSON writes an error response in JSON format to the provided
// http.ResponseWriter. If the error is an *httpError, it uses the code
// and error message from that error. Otherwise, it defaults to
// http.StatusInternalServerError, obfuscating the error message in that case.
func WriteErrorJSON(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	if httpErr, ok := err.(*HTTPError); ok {
		code = httpErr.code
		err = httpErr.err
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	resp := struct {
		Error string `json:"error,omitempty"`
	}{}
	if code != http.StatusInternalServerError && err != nil {
		resp.Error = err.Error()
	}
	_ = json.NewEncoder(w).Encode(resp)
}
