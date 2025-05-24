package http

import (
	"encoding/json"
	"net/http"
)

type httpError struct {
	code int
	err  error
}

func (e *httpError) Error() string {
	return e.err.Error()
}

func Error(err error, code int) error {
	return &httpError{
		code: code,
		err:  err,
	}
}

func WriteErrorJSON(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	if httpErr, ok := err.(*httpError); ok {
		code = httpErr.code
		err = httpErr.err
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	resp := struct {
		Error string `json:"error,omitempty"`
	}{}
	if code != http.StatusInternalServerError {
		resp.Error = err.Error()
	}
	_ = json.NewEncoder(w).Encode(resp)
}
