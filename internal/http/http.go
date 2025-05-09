package http

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

const maxBytes = 2 << 20

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

func LimitRead(r io.Reader) ([]byte, error) {
	lr := io.LimitReader(r, maxBytes)

	// Read as far as we are allowed to
	bodyBytes, err := io.ReadAll(lr)
	if err != nil {
		return nil, Error(
			fmt.Errorf(
				"failed to read from reader: %w", err,
			),
			http.StatusBadRequest,
		)
	}

	// If we read exactly the maximum, the body might be larger
	if len(bodyBytes) == maxBytes {
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
				fmt.Errorf("response body exceeds maximum size of %d bytes", maxBytes),
				http.StatusRequestEntityTooLarge,
			)
		}
	}
	return bodyBytes, nil
}
