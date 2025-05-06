package http

import (
	"bytes"
	"fmt"
	"io"
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

// PeakBody reads from req without advancing the byte-cursor;
// allowing an additional future read to occur without erroring
// out on an EOF.
func PeakBody(req *http.Request) ([]byte, error) {
	b, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}
	_ = req.Body.Close()
	req.Body = io.NopCloser(bytes.NewBuffer(b))
	return b, nil
}
