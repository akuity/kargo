package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

var noCacheHeaders = map[string]string{
	"Expires":         time.Unix(0, 0).Format(time.RFC1123),
	"Cache-Control":   "no-cache, private, max-age=0",
	"Pragma":          "no-cache",
	"X-Accel-Expires": "0",
}

func SetNoCacheHeaders(w http.ResponseWriter) {
	if w == nil {
		return
	}
	for k, v := range noCacheHeaders {
		w.Header().Set(k, v)
	}
}

func SetCacheHeaders(w http.ResponseWriter, maxAge time.Duration, timeUntilExpiry time.Duration) {
	if w == nil {
		return
	}
	w.Header().Set("Cache-Control", "public, max-age="+maxAge.String())
	w.Header().Set("Expires", time.Now().Add(timeUntilExpiry).Format(time.RFC1123))
}

// Error is similar to the standard libaries implementation
// but writes the error as json to the response writer.
func Error(w http.ResponseWriter, statusCode int, err any) {
	var msg string
	switch t := err.(type) {
	case error:
		msg = t.Error()
	case string:
		msg = t
	default:
		msg = fmt.Sprintf("%v", t)
	}
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(
		map[string]string{"error": msg},
	)
}

// Errorf writes a formatted error as json to the response writer
func Errorf(w http.ResponseWriter, statusCode int, format string, args ...any) {
	Error(w, statusCode, fmt.Errorf(format, args...))
}
