package http

import (
	"encoding/json"
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

func LimitRead(r io.ReadCloser, limit int64) ([]byte, error) {
	defer r.Close()
	lr := io.LimitReader(r, limit)

	// Read as far as we are allowed to
	bodyBytes, err := io.ReadAll(lr)
	if err != nil {
		return nil, fmt.Errorf("failed to read from reader: %w", err)
	}

	// If we read exactly the limit, the body might be larger
	if int64(len(bodyBytes)) == limit {
		// Try to read one more byte
		buf := make([]byte, 1)
		var n int
		if n, err = r.Read(buf); err != nil && err != io.EOF {
			return nil, fmt.Errorf(
				"failed to check for additional content: %w",
				err,
			)
		}
		if n > 0 {
			return nil, Error(
				fmt.Errorf("response body exceeds limit of %d bytes", limit),
				http.StatusRequestEntityTooLarge,
			)
		}
	}
	return bodyBytes, nil
}

func WriteResponseJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if body == nil {
		body = struct{}{}
	}
	_ = json.NewEncoder(w).Encode(body)
}
