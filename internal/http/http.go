package http

import (
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
